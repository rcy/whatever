package commands

import (
	"fmt"
	"strings"

	"github.com/rcy/whatever/events"
	"github.com/rcy/whatever/ids"
)

type Service struct {
	ES *events.Service
}

func New(es *events.Service) *Service {
	return &Service{ES: es}
}

func (s *Service) CreateNote(text string) (string, error) {
	aggID := ids.New()
	text = strings.TrimSpace(text)
	if text == "" {
		return "", fmt.Errorf("text cannot be empty")
	}
	err := s.ES.InsertEvent("NoteCreated", "note", aggID, events.NoteCreatedPayload{Text: text})
	if err != nil {
		return "", err
	}
	return aggID, nil
}

func (s *Service) DeleteNote(id string) error {
	aggID, err := s.ES.GetAggregateID(id)
	if err != nil {
		return err
	}

	return s.ES.InsertEvent("NoteDeleted", "note", aggID, nil)
}

func (s *Service) UndeleteNote(id string) error {
	aggID, err := s.ES.GetAggregateID(id)
	if err != nil {
		return err
	}

	return s.ES.InsertEvent("NoteUndeleted", "note", aggID, nil)
}
