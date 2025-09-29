package aggregates

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rcy/evoke"
	"github.com/rcy/whatever/commands"
	"github.com/rcy/whatever/events"
)

type noteAggregate struct {
	id       uuid.UUID
	deleted  bool
	text     string
	category string
}

func NewNoteAggregate(id uuid.UUID) *noteAggregate {
	return &noteAggregate{id: id}
}

func (a *noteAggregate) HandleCommand(cmd evoke.Command) ([]evoke.Event, error) {
	aggregateID := cmd.AggregateID()
	if aggregateID == uuid.Nil {
		return nil, fmt.Errorf("no aggregateID: %v", cmd)
	}

	if a.id != aggregateID {
		panic("id mismatch")
	}

	switch c := cmd.(type) {
	case commands.CreateNote:
		text := strings.TrimSpace(c.Text)
		if text == "" {
			return nil, fmt.Errorf("text cannot be empty")
		}

		if c.RealmID == uuid.Nil {
			return nil, fmt.Errorf("realm cannot be empty")
		}

		return []evoke.Event{events.NoteCreated{
			NoteID:    aggregateID,
			CreatedAt: time.Now(),
			RealmID:   c.RealmID,
			Text:      text,
		}}, nil
	case commands.DeleteNote:
		if a.deleted {
			return nil, fmt.Errorf("note already deleted")
		}
		return []evoke.Event{events.NoteDeleted{
			NoteID: aggregateID,
		}}, nil
	case commands.UndeleteNote:
		if !a.deleted {
			return nil, fmt.Errorf("note not deleted: %s %s %v", a.id, a.text, a.deleted)
		}
		return []evoke.Event{events.NoteUndeleted{
			NoteID: aggregateID,
		}}, nil
	case commands.UpdateNoteText:
		text := strings.TrimSpace(c.Text)
		if text == "" {
			return nil, fmt.Errorf("text cannot be empty")
		}

		return []evoke.Event{events.NoteTextUpdated{
			NoteID: aggregateID,
			Text:   c.Text,
		}}, nil
	case commands.SetNoteCategory:
		category := strings.TrimSpace(c.Category)

		return []evoke.Event{events.NoteCategoryChanged{
			NoteID:   aggregateID,
			Category: category,
		}}, nil
	}
	return nil, fmt.Errorf("unhandled")
}

func (a *noteAggregate) Apply(e evoke.Event) error {
	switch evt := e.(type) {
	case events.NoteCreated:
		a.id = evt.NoteID // should already be set?
		a.text = evt.Text
	case events.NoteDeleted:
		a.deleted = true
	case events.NoteUndeleted:
		a.deleted = false
	case events.NoteTextUpdated:
		a.text = evt.Text
	case events.NoteCategoryChanged:
		a.category = evt.Category
	default:
		return fmt.Errorf("not handled")
	}
	return nil
}
