package notes

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/rcy/whatever/events"
	"github.com/rcy/whatever/evoke"
	_ "modernc.org/sqlite"
)

type Note struct {
	ID       string    `db:"id"`
	Ts       time.Time `db:"ts"`
	Text     string    `db:"text"`
	Category string    `db:"category"`
}

type Projection struct {
	db *sqlx.DB
}

func (p *Projection) FindOne(id string) (Note, error) {
	var note Note
	err := p.db.Get(&note, `select * from notes where id = ?`, id)
	if err != nil {
		return Note{}, err
	}
	return note, nil
}

func (p *Projection) FindAll() ([]Note, error) {
	var noteList []Note
	err := p.db.Select(&noteList, `select * from notes order by ts asc`)
	if err != nil {
		return nil, fmt.Errorf("Select notes: %w", err)
	}
	return noteList, nil
}

func (p *Projection) FindAllDeleted() ([]Note, error) {
	var noteList []Note
	err := p.db.Select(&noteList, `select * from deleted_notes order by ts asc`)
	if err != nil {
		return nil, fmt.Errorf("Select deleted notes: %w", err)
	}
	return noteList, nil
}

func New() (*Projection, error) {
	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`create table notes(id not null, ts timestamp not null, text not null, category not null)`)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`create table deleted_notes(id not null, ts timestamp not null, text not null, category not null)`)
	if err != nil {
		return nil, err
	}

	return &Projection{db: db}, nil
}

// Register this projection with the event system by subscribing to events
func (p *Projection) Register(e evoke.Subscriber) {
	e.Subscribe(events.NoteCreated, p.updateNotes)
	e.Subscribe(events.NoteDeleted, p.updateNotes)
	e.Subscribe(events.NoteUndeleted, p.updateNotes)
	e.Subscribe(events.NoteTextUpdated, p.updateNotes)
	e.Subscribe(events.NoteCategoryChanged, p.updateNotes)
}

func (p *Projection) updateNotes(event evoke.Event, _ evoke.Inserter, _ bool) error {
	switch event.EventType {
	case events.NoteCreated.Name:
		payload, err := evoke.UnmarshalPayload[events.NoteCreatedPayload](event)
		if err != nil {
			return err
		}
		_, err = p.db.Exec(`insert into notes(id, ts, text, category) values(?,?,?,?)`, event.AggregateID, event.CreatedAt, payload.Text, "inbox")
		if err != nil {
			return err
		}
	case events.NoteDeleted.Name:
		_, err := p.db.Exec(`insert into deleted_notes(id, ts, text, category) select id, ts, text, category from notes where id = ?`, event.AggregateID)
		if err != nil {
			return err
		}

		_, err = p.db.Exec(`delete from notes where id = ?`, event.AggregateID)
		return err
	case events.NoteUndeleted.Name:
		_, err := p.db.Exec(`insert into notes(id, ts, text, category) select id, ts, text, category from deleted_notes where id = ?`, event.AggregateID)
		if err != nil {
			return err
		}

		_, err = p.db.Exec(`delete from deleted_notes where id = ?`, event.AggregateID)
		return err
	case events.NoteTextUpdated.Name:
		payload, err := evoke.UnmarshalPayload[events.NoteTextUpdatedPayload](event)
		if err != nil {
			return err
		}

		_, err = p.db.Exec(`update notes set text = ? where id = ?`, payload.Text, event.AggregateID)
		if err != nil {
			return err
		}
		return err
	case events.NoteCategoryChanged.Name:
		payload, err := evoke.UnmarshalPayload[events.NoteCategoryChangedPayload](event)
		if err != nil {
			return err
		}

		_, err = p.db.Exec(`update notes set category = ? where id = ?`, payload.Category, event.AggregateID)
		if err != nil {
			return err
		}
		return err
	default:
		return fmt.Errorf("EventType not handled")
	}
	return nil
}
