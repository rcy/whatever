package notes

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/rcy/whatever/flog"
	"github.com/rcy/whatever/payloads"
	_ "modernc.org/sqlite"
)

type Model struct {
	ID       string    `db:"id"`
	Text     string    `db:"text"`
	Category string    `db:"category"`
	Ts       time.Time `db:"ts"`
}

func (m Model) String() string {
	return fmt.Sprintf("%s %s %s", m.ID[0:7], m.Ts.Local().Format(time.DateTime), m.Text)
}

type Projection struct {
	db *sqlx.DB
}

func (s *Projection) FindOne(id string) (Model, error) {
	var note Model
	err := s.db.Get(&note, `select * from notes where id = ?`, id)
	if err != nil {
		return Model{}, err
	}
	return note, nil
}

func (s *Projection) FindAll() ([]Model, error) {
	var noteList []Model
	err := s.db.Select(&noteList, `select * from notes order by ts asc`)
	if err != nil {
		return nil, fmt.Errorf("Select notes: %w", err)
	}
	return noteList, nil
}

func (s *Projection) FindAllDeleted() ([]Model, error) {
	var noteList []Model
	err := s.db.Select(&noteList, `select * from deleted_notes order by ts asc`)
	if err != nil {
		return nil, fmt.Errorf("Select deleted notes: %w", err)
	}
	return noteList, nil
}

func Init() (*Projection, error) {
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

	s := &Projection{db: db}

	return s, nil
}

// Register this projection with the event system by subscribing to events
func (s *Projection) RegisterEventSubscriptions(e flog.EventSubscriber) {
	e.Subscribe(payloads.NoteCreated, s.updateNotesProjection)
	e.Subscribe(payloads.NoteDeleted, s.updateNotesProjection)
	e.Subscribe(payloads.NoteUndeleted, s.updateNotesProjection)
	e.Subscribe(payloads.NoteTextUpdated, s.updateNotesProjection)
	e.Subscribe(payloads.NoteCategoryChanged, s.updateNotesProjection)
}

func (s *Projection) updateNotesProjection(event flog.Model, _ flog.EventInserter, _ bool) error {
	switch event.EventType {
	case payloads.NoteCreated:
		payload, err := flog.UnmarshalPayload[payloads.NoteCreatedPayload](event)
		if err != nil {
			return err
		}
		_, err = s.db.Exec(`insert into notes(id, ts, text, category) values(?,?,?,?)`, event.AggregateID, event.CreatedAt, payload.Text, "inbox")
		if err != nil {
			return err
		}
	case payloads.NoteDeleted:
		_, err := s.db.Exec(`insert into deleted_notes(id, ts, text, category) select id, ts, text, category from notes where id = ?`, event.AggregateID)
		if err != nil {
			return err
		}

		_, err = s.db.Exec(`delete from notes where id = ?`, event.AggregateID)
		return err
	case payloads.NoteUndeleted:
		_, err := s.db.Exec(`insert into notes(id, ts, text, category) select id, ts, text, category from deleted_notes where id = ?`, event.AggregateID)
		if err != nil {
			return err
		}

		_, err = s.db.Exec(`delete from deleted_notes where id = ?`, event.AggregateID)
		return err
	case payloads.NoteTextUpdated:
		payload, err := flog.UnmarshalPayload[payloads.NoteTextUpdatedPayload](event)
		if err != nil {
			return err
		}

		_, err = s.db.Exec(`update notes set text = ? where id = ?`, payload.Text, event.AggregateID)
		if err != nil {
			return err
		}
		return err
	case payloads.NoteCategoryChanged:
		payload, err := flog.UnmarshalPayload[payloads.NoteCategoryChangedPayload](event)
		if err != nil {
			return err
		}

		_, err = s.db.Exec(`update notes set category = ? where id = ?`, payload.Category, event.AggregateID)
		if err != nil {
			return err
		}
		return err
	default:
		return fmt.Errorf("EventType not handled")
	}
	return nil
}
