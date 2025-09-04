package app

import (
	"encoding/json"
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

	_, err := es.DBTodo.Exec(`drop table if exists notes`)
	if err != nil {
		log.Fatal(err)
	}
	_, err = es.DBTodo.Exec(`create table notes(id, ts timestamp, text)`)
	if err != nil {
		log.Fatal(err)
	}

	err = es.ReplayEvents()
	if err != nil {
		log.Fatal(err)
	}

	return &Service{CS: cs, ES: es}
}

func unmarshalPayload[T any](event events.Model) (T, error) {
	var payload T
	err := json.Unmarshal([]byte(event.EventData), &payload)
	return payload, err
}

func updateNotesProjection(e sqlx.Execer, event events.Model) error {
	note, err := unmarshalPayload[events.NoteCreatedPayload](event)
	if err != nil {
		return err
	}
	_, err = e.Exec(`insert into notes(id, ts, text) values(?,?,?)`, event.AggregateID, event.CreatedAt, note.Text)
	if err != nil {
		return err
	}
	return nil
}
