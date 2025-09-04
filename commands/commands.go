package commands

import (
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
	payload := struct {
		Text string
	}{
		Text: text,
	}

	aggID := ids.New()

	err := s.ES.InsertEvent("NoteCreated", "note", aggID, payload)
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
