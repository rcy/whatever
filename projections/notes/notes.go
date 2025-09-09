package notes

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/rcy/evoke"
	"github.com/rcy/whatever/events"
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

func New(e *evoke.Service) (*Projection, error) {
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

func (p *Projection) Subscribe(e *evoke.Service) error {
	err := e.Subscribe(events.NoteCreated{}, p.noteCreated)
	if err != nil {
		return err
	}
	err = e.Subscribe(events.NoteDeleted{}, p.noteDeleted)
	if err != nil {
		return err
	}
	err = e.Subscribe(events.NoteUndeleted{}, p.noteUndeleted)
	if err != nil {
		return err
	}
	err = e.Subscribe(events.NoteTextUpdated{}, p.noteTextUpdated)
	if err != nil {
		return err
	}
	err = e.Subscribe(events.NoteCategoryChanged{}, p.noteCategoryChanged)
	if err != nil {
		return err
	}

	return nil
}

func (p *Projection) noteCreated(event evoke.Event, _ evoke.Inserter, _ bool) error {
	payload, err := evoke.UnmarshalPayload[events.NoteCreated](event)
	if err != nil {
		return err
	}
	_, err = p.db.Exec(`insert into notes(id, ts, text, category) values(?,?,?,?)`, event.AggregateID, event.CreatedAt, payload.Text, "inbox")
	return nil
}

func (p *Projection) noteDeleted(event evoke.Event, _ evoke.Inserter, _ bool) error {
	_, err := p.db.Exec(`insert into deleted_notes(id, ts, text, category) select id, ts, text, category from notes where id = ?`, event.AggregateID)
	if err != nil {
		return err
	}

	_, err = p.db.Exec(`delete from notes where id = ?`, event.AggregateID)
	return err
}

func (p *Projection) noteUndeleted(event evoke.Event, _ evoke.Inserter, _ bool) error {
	_, err := p.db.Exec(`insert into notes(id, ts, text, category) select id, ts, text, category from deleted_notes where id = ?`, event.AggregateID)
	if err != nil {
		return err
	}

	_, err = p.db.Exec(`delete from deleted_notes where id = ?`, event.AggregateID)
	return err
}

func (p *Projection) noteTextUpdated(event evoke.Event, _ evoke.Inserter, _ bool) error {
	payload, err := evoke.UnmarshalPayload[events.NoteTextUpdated](event)
	if err != nil {
		return err
	}

	_, err = p.db.Exec(`update notes set text = ? where id = ?`, payload.Text, event.AggregateID)
	return nil
}

func (p *Projection) noteCategoryChanged(event evoke.Event, _ evoke.Inserter, _ bool) error {
	payload, err := evoke.UnmarshalPayload[events.NoteCategoryChanged](event)
	if err != nil {
		return err
	}

	_, err = p.db.Exec(`update notes set category = ? where id = ?`, payload.Category, event.AggregateID)
	return err
}
