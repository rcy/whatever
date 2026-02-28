package aggregates

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rcy/evoke"
	"github.com/rcy/whatever/catalog/notesmeta"
	"github.com/rcy/whatever/commands"
	"github.com/rcy/whatever/events"
)

var location = func() *time.Location {
	loc, err := time.LoadLocation("America/Creston")
	if err != nil {
		panic(err)
	}
	return loc
}()

type noteAggregate struct {
	id          uuid.UUID
	owner       string
	deleted     bool
	text        string
	category    string
	subcategory string
	due         *time.Time
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
		if c.Owner == "" {
			return nil, fmt.Errorf("owner cannot be empty")
		}

		text := strings.TrimSpace(c.Text)
		if text == "" {
			return nil, fmt.Errorf("text cannot be empty")
		}

		if c.Category == "" {
			return nil, fmt.Errorf("category cannot be empty")
		}

		if c.Subcategory == "" {
			return nil, fmt.Errorf("subcategory cannot be empty")
		}

		eventList := []evoke.Event{
			events.NoteCreated{
				Owner:       c.Owner,
				NoteID:      aggregateID,
				CreatedAt:   time.Now(),
				Text:        text,
				Category:    c.Category,
				Subcategory: c.Subcategory,
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
	case commands.SetNoteOwner:
		return []evoke.Event{
			events.NoteOwnerSet{
				NoteID: c.NoteID,
				Owner:  c.Owner,
			},
		}, nil
	case commands.SetNoteCategory:
		categoryName := strings.TrimSpace(c.Category)
		if a.category == categoryName {
			return nil, fmt.Errorf("note already set to category: %s", categoryName)
		}

		subcategory := notesmeta.Categories.Get(categoryName).Inbox()

		return []evoke.Event{
			events.NoteCategoryChanged{
				NoteID:      aggregateID,
				Category:    categoryName,
				Subcategory: string(subcategory.Slug),
			},
		}, nil
	case commands.TransitionNoteSubcategory:
		transitionEvent := strings.TrimSpace(c.TransitionEvent)
		subcategory := notesmeta.Categories.Get(a.category).Subcategories.Get(a.subcategory)
		ok, transition := subcategory.Transitions.Get(transitionEvent)
		if !ok {
			return nil, fmt.Errorf("invalid transition event %s", c.TransitionEvent)
		}

		if a.subcategory == transition.TargetSlug {
			return nil, fmt.Errorf("note already set to subcategory: %s", subcategory)
		}
		eventList := []evoke.Event{events.NoteSubcategoryChanged{
			NoteID:      aggregateID,
			Subcategory: transition.TargetSlug,
		}}

		if transition.DaysUntilDue != nil {
			due := notesmeta.Midnight(time.Now().In(location)).AddDate(0, 0, transition.DaysUntilDue())
			eventList = append(eventList, events.NoteDueChanged{NoteID: aggregateID, Due: due})
		} else if a.due != nil {
			eventList = append(eventList, events.NoteDueCleared{NoteID: aggregateID})
		}

		return eventList, nil
	case commands.SetNoteDue:
		return []evoke.Event{events.NoteDueChanged{
			NoteID: aggregateID,
			Due:    c.Due,
		}}, nil
	case commands.ClearNoteDue:
		return []evoke.Event{events.NoteDueCleared{
			NoteID: aggregateID,
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
		a.owner = evt.Owner
	case events.NoteOwnerSet:
		a.owner = evt.Owner
	case events.NoteDeleted:
		a.deleted = true
	case events.NoteUndeleted:
		a.deleted = false
	case events.NoteTextUpdated:
		a.text = evt.Text
	case events.NoteCategoryChanged:
		a.category = evt.Category
		a.subcategory = evt.Subcategory
	case events.NoteSubcategoryChanged:
		a.subcategory = evt.Subcategory
	case events.NoteDueChanged:
		a.due = &evt.Due
	case events.NoteDueCleared:
		a.due = nil
	case events.NoteEnrichmentRequested:
	case events.NoteEnriched:
	case events.NoteEnrichmentFailed:
	default:
		return fmt.Errorf("not handled")
	}
	return nil
}
