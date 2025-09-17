package sagas

import (
	"fmt"

	"github.com/rcy/evoke"
	"github.com/rcy/whatever/commands"
	"github.com/rcy/whatever/events"
	"github.com/rcy/whatever/projections/notes"
)

type appService interface {
	Events() *evoke.Service
	Commands() *commands.Service
	Notes() *notes.Projection
}

type service struct {
	appService
}

func New(app appService) *service {
	return &service{app}
}

func (s *service) Subscribe() {
	s.Events().Subscribe("UNPARENT_NOTES", events.RealmDeleted{}, s.unparentNotes)
	s.Events().Subscribe("PARENT_NOTES", events.RealmCreated{}, s.reparentNotes)
}

func (s *service) reparentNotes(event evoke.Event, replay bool) error {
	if replay {
		return nil
	}

	// find all the unrealmed notes and add commands to set note realms
	notes, err := s.Notes().FindAllInRealm("")
	if err != nil {
		return err
	}

	for _, note := range notes {
		err := s.Commands().SetNoteRealm(note.ID, event.AggregateID)
		if err != nil {
			return fmt.Errorf("SetNoteRealm: %w", err)
		}
	}
	return nil
}

func (s *service) unparentNotes(event evoke.Event, replay bool) error {
	if replay {
		return nil
	}

	// find all the unrealmed notes and add commands to set note realms
	notes, err := s.Notes().FindAllInRealm(event.AggregateID)
	if err != nil {
		return err
	}

	for _, note := range notes {
		err := s.Commands().SetNoteRealm(note.ID, "")
		if err != nil {
			return fmt.Errorf("SetNoteRealm: %w", err)
		}
	}
	return nil
}
