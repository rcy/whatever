package commands

import (
	"fmt"
	"strings"

	"github.com/rcy/whatever/events"
	"github.com/rcy/whatever/evoke"
)

type Service struct {
	Events evoke.Inserter
}

func New(events evoke.Inserter) *Service {
	return &Service{Events: events}
}

func (s *Service) CreateNote(text string) (string, error) {
	aggID := evoke.ID()
	text = strings.TrimSpace(text)
	if text == "" {
		return "", fmt.Errorf("text cannot be empty")
	}
	err := s.Events.Insert(aggID, events.NoteCreated{Text: text})
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

	return s.Events.Insert(aggID, events.NoteDeleted{})
}

func (s *Service) UndeleteNote(id string) error {
	aggID, err := s.Events.GetAggregateID(id)
	if err != nil {
		return err
	}

	return s.Events.Insert(aggID, events.NoteUndeleted{})
}

func (s *Service) UpdateNoteText(id string, text string) error {
	aggID, err := s.Events.GetAggregateID(id)
	if err != nil {
		return err
	}

	err = s.Events.Insert(aggID, events.NoteTextUpdated{Text: text})
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

	err = s.Events.Insert(aggID, events.NoteCategoryChanged{Category: text})
	if err != nil {
		return err
	}
	return nil
}
