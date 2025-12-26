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

		eventList := []evoke.Event{
			events.NoteCreated{
				NoteID:    aggregateID,
				CreatedAt: time.Now(),
				RealmID:   c.RealmID,
				Text:      text,
			},
		}

		// TODO: better matching here
		if strings.HasPrefix(text, "http") {
			eventList = append(eventList, events.NoteEnrichmentRequested{
				NoteID:      aggregateID,
				RequestedAt: time.Now(),
				Text:        text,
			})
		}

		return eventList, nil
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

		eventList := []evoke.Event{events.NoteTextUpdated{
			NoteID: aggregateID,
			Text:   c.Text,
		}}

		// TODO: better matching here
		if strings.HasPrefix(text, "http") {
			eventList = append(eventList, events.NoteEnrichmentRequested{
				NoteID:      aggregateID,
				RequestedAt: time.Now(),
				Text:        text,
			})
		}

		return eventList, nil
	case commands.SetNoteCategory:
		category := strings.TrimSpace(c.Category)

		return []evoke.Event{events.NoteCategoryChanged{
			NoteID:   aggregateID,
			Category: category,
		}}, nil
	case commands.CompleteNoteEnrichment:
		return []evoke.Event{events.NoteEnriched{
			NoteID: aggregateID,
			Title:  c.Title,
		}}, nil
	case commands.FailNoteEnrichment:
		return []evoke.Event{events.NoteEnrichmentFailed{
			NoteID: aggregateID,
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
	case events.NoteEnrichmentRequested:
	case events.NoteEnriched:
	case events.NoteEnrichmentFailed:
	default:
		return fmt.Errorf("not handled")
	}
	return nil
}
