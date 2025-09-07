package commands

import (
	"fmt"
	"strings"

	"github.com/rcy/whatever/flog"
	"github.com/rcy/whatever/payloads"
)

type Service struct {
	Events *flog.Service
}

func New(es *flog.Service) *Service {
	return &Service{Events: es}
}

func (s *Service) CreateNote(text string) (string, error) {
	aggID := flog.ID()
	text = strings.TrimSpace(text)
	if text == "" {
		return "", fmt.Errorf("text cannot be empty")
	}
	err := s.Events.InsertEvent(payloads.NoteCreated, "note", aggID, payloads.NoteCreatedPayload{Text: text})
	if err != nil {
		return "", err
	}
	return aggID, nil
}

func (s *Service) DeleteNote(id string) error {
	aggID, err := s.Events.GetAggregateID(id)
	if err != nil {
		return err
	}

	return s.Events.InsertEvent(payloads.NoteDeleted, "note", aggID, nil)
}

func (s *Service) UndeleteNote(id string) error {
	aggID, err := s.Events.GetAggregateID(id)
	if err != nil {
		return err
	}

	return s.Events.InsertEvent(payloads.NoteUndeleted, "note", aggID, nil)
}

func (s *Service) UpdateNoteText(id string, text string) error {
	aggID, err := s.Events.GetAggregateID(id)
	if err != nil {
		return err
	}

	err = s.Events.InsertEvent(payloads.NoteTextUpdated, "note", aggID, payloads.NoteTextUpdatedPayload{Text: text})
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) SetNoteCategory(id string, text string) error {
	aggID, err := s.Events.GetAggregateID(id)
	if err != nil {
		return err
	}

	err = s.Events.InsertEvent(payloads.NoteCategoryChanged, "note", aggID, payloads.NoteCategoryChangedPayload{Category: text})
	if err != nil {
		return err
	}
	return nil
}
