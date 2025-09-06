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
	ID   string    `db:"id"`
	Text string    `db:"text"`
	Ts   time.Time `db:"ts"`
}

func (m Model) String() string {
	return fmt.Sprintf("%s %s %s", m.ID[0:7], m.Ts.Local().Format(time.DateTime), m.Text)
}

type EventHandlerRegisterer interface {
	RegisterHandler(string, flog.HandlerFunc)
}

type Service struct {
	db *sqlx.DB
}

func (s *Service) FindOne(id string) (Model, error) {
	var note Model
	err := s.db.Get(&note, `select * from notes where id = ?`, id)
	if err != nil {
		return Model{}, err
	}
	return note, nil
}

func (s *Service) FindAll() ([]Model, error) {
	var noteList []Model
	err := s.db.Select(&noteList, `select * from notes order by ts asc`)
	if err != nil {
		return nil, fmt.Errorf("Select notes: %w", err)
	}
	return noteList, nil
}

func (s *Service) FindAllDeleted() ([]Model, error) {
	var noteList []Model
	err := s.db.Select(&noteList, `select * from deleted_notes order by ts asc`)
	if err != nil {
		return nil, fmt.Errorf("Select notes: %w", err)
	}
	return noteList, nil
}

func Init(e EventHandlerRegisterer) (*Service, error) {
	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`create table notes(id, ts timestamp, text)`)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`create table deleted_notes(id, ts timestamp, text)`)
	if err != nil {
		return nil, err
	}

	s := &Service{db: db}

	e.RegisterHandler("NoteCreated", s.updateNotesProjection)
	e.RegisterHandler("NoteDeleted", s.updateNotesProjection)
	e.RegisterHandler("NoteUndeleted", s.updateNotesProjection)

	return s, nil
}

func (s *Service) updateNotesProjection(i flog.EventInserter, event flog.Model, _ bool) error {
	switch event.EventType {
	case "NoteCreated":
		note, err := flog.UnmarshalPayload[payloads.NoteCreatedPayload](event)
		if err != nil {
			return err
		}
		_, err = s.db.Exec(`insert into notes(id, ts, text) values(?,?,?)`, event.AggregateID, event.CreatedAt, note.Text)
		if err != nil {
			return err
		}
	case "NoteDeleted":
		_, err := s.db.Exec(`insert into deleted_notes(id, ts, text) select id, ts, text from notes where id = ?`, event.AggregateID)
		if err != nil {
			return err
		}

		_, err = s.db.Exec(`delete from notes where id = ?`, event.AggregateID)
		return err
	case "NoteUndeleted":
		_, err := s.db.Exec(`insert into notes(id, ts, text) select id, ts, text from deleted_notes where id = ?`, event.AggregateID)
		if err != nil {
			return err
		}

		_, err = s.db.Exec(`delete from deleted_notes where id = ?`, event.AggregateID)
		return err
	default:
		return fmt.Errorf("EventType not handled")
	}
	return nil
}
