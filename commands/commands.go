package commands

import (
	"fmt"
	"strings"

	"github.com/rcy/whatever/flog"
	"github.com/rcy/whatever/ids"
	"github.com/rcy/whatever/payloads"
)

type Service struct {
	ES *flog.Service
}

func New(es *flog.Service) *Service {
	return &Service{ES: es}
}

func (s *Service) CreateNote(text string) (string, error) {
	aggID := ids.New()
	text = strings.TrimSpace(text)
	if text == "" {
		return "", fmt.Errorf("text cannot be empty")
	}
	err := s.ES.InsertEvent("NoteCreated", "note", aggID, payloads.NoteCreatedPayload{Text: text})
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

func (s *Service) UpdateNoteText(id string, text string) error {
	aggID, err := s.ES.GetAggregateID(id)
	if err != nil {
		return err
	}

	err = s.ES.InsertEvent("NoteTextUpdated", "note", aggID, payloads.NoteTextUpdatedPayload{Text: text})
	if err != nil {
		return err
	}
	return nil
}
