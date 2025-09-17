package commands

import (
	"fmt"
	"strings"

	"github.com/rcy/evoke"
	"github.com/rcy/whatever/events"
)

type Service struct {
	Events evoke.Inserter
}

func New(events evoke.Inserter) *Service {
	return &Service{Events: events}
}

func (s *Service) CreateRealm(name string) (string, error) {
	aggID := evoke.ID()
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("name cannot be empty")
	}
	err := s.Events.Insert(aggID, events.RealmCreated{Name: name})
	if err != nil {
		return "", err
	}
	return aggID, nil
}

func (s *Service) DeleteRealm(realmID string) error {
	aggID, err := s.Events.GetAggregateID(realmID)
	if err != nil {
		return err
	}

	err = s.Events.Insert(aggID, events.RealmDeleted{})
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) CreateNote(realmID string, text string) (string, error) {
	if realmID == "" {
		return "", fmt.Errorf("realm cannot be empty")
	}

	aggID := evoke.ID()
	text = strings.TrimSpace(text)
	if text == "" {
		return "", fmt.Errorf("text cannot be empty")
	}
	err := s.Events.Insert(aggID, events.NoteCreated{RealmID: realmID, Text: text})
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

func (s *Service) SetNoteRealm(id string, realmID string) error {
	aggID, err := s.Events.GetAggregateID(id)
	if err != nil {
		return err
	}

	err = s.Events.Insert(aggID, events.NoteRealmChanged{RealmID: realmID})
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
