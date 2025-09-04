package app

import (
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/rcy/whatever/commands"
	"github.com/rcy/whatever/events"
)

type Service struct {
	CS *commands.Service
	ES *events.Service
}

func New(cs *commands.Service, es *events.Service) *Service {
	es.RegisterHandler("NoteCreated", updateNotesProjection)
	es.RegisterHandler("NoteDeleted", updateNotesProjection)
	es.RegisterHandler("NoteUndeleted", updateNotesProjection)

	_, err := es.DBTodo.Exec(`drop table if exists notes`)
	if err != nil {
		log.Fatal(err)
	}
	_, err = es.DBTodo.Exec(`create table notes(id, ts timestamp, text)`)
	if err != nil {
		log.Fatal(err)
	}
	_, err = es.DBTodo.Exec(`drop table if exists deleted_notes`)
	if err != nil {
		log.Fatal(err)
	}
	_, err = es.DBTodo.Exec(`create table deleted_notes(id, ts timestamp, text)`)
	if err != nil {
		log.Fatal(err)
	}

	err = es.ReplayEvents()
	if err != nil {
		log.Fatal(err)
	}

	return &Service{CS: cs, ES: es}
}

func updateNotesProjection(e sqlx.Execer, event events.Model) error {
	switch event.EventType {
	case "NoteCreated":
		note, err := events.UnmarshalPayload[events.NoteCreatedPayload](event)
		if err != nil {
			return err
		}
		_, err = e.Exec(`insert into notes(id, ts, text) values(?,?,?)`, event.AggregateID, event.CreatedAt, note.Text)
		if err != nil {
			return err
		}
	case "NoteDeleted":
		_, err := e.Exec(`insert into deleted_notes(id, ts, text) select id, ts, text from notes where id = ?`, event.AggregateID)
		if err != nil {
			return err
		}

		_, err = e.Exec(`delete from notes where id = ?`, event.AggregateID)
		return err
	case "NoteUndeleted":
		_, err := e.Exec(`insert into notes(id, ts, text) select id, ts, text from deleted_notes where id = ?`, event.AggregateID)
		if err != nil {
			return err
		}

		_, err = e.Exec(`delete from deleted_notes where id = ?`, event.AggregateID)
		return err
	default:
		return fmt.Errorf("EventType not handled")
	}
	return nil
}
