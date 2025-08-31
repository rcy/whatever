package commands

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rcy/whatever/commands/service"
)

type NotesCmd struct {
	Ls ListCmd `cmd:"" default:"withargs"`
	//Show     ShowCmd     `cmd:""`
	Add      AddCmd      `cmd:""`
	Rm       DeleteCmd   `cmd:""`
	Undelete UndeleteCmd `cmd:""`
}

type ListCmd struct {
	Deleted bool `help:"Show deleted notes"`
}

func (c *ListCmd) Run(s *service.Service) error {
	type id string
	type note struct {
		text    string
		deleted bool
	}
	notes := make(map[id]note)
	var events []Event
	err := s.DB.Select(&events, `select * from events where aggregate_type = 'note' order by created_at asc`)
	if err != nil {
		return fmt.Errorf("Select events: %w", err)
	}
	for _, event := range events {
		switch event.EventType {
		case "NoteCreated":
			payload := struct{ Text string }{}
			err := json.Unmarshal(event.EventData, &payload)
			if err != nil {
				return fmt.Errorf("Unmarshal: %w", err)
			}
			notes[id(event.AggregateID)] = note{text: payload.Text}
		case "NoteDeleted":
			note, ok := notes[id(event.AggregateID)]
			if ok {
				note.deleted = true
				notes[id(event.AggregateID)] = note
			}
		case "NoteUndeleted":
			note, ok := notes[id(event.AggregateID)]
			if ok {
				note.deleted = false
				notes[id(event.AggregateID)] = note
			}
		default:
			return fmt.Errorf("unhandled event.EventType: %s", event.EventType)
		}
	}

	for id, note := range notes {
		if c.Deleted && note.deleted || !c.Deleted && !note.deleted {
			fmt.Printf("%s %s\n", id[0:7], note.text)
		}
	}

	return nil
}

type AddCmd struct {
	Text []string `arg:""`
}

func (c *AddCmd) Run(s *service.Service) error {
	payload := struct {
		Text string
	}{
		Text: strings.Join(c.Text, " "),
	}

	bytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}

	_, err = s.DB.Exec(`insert into events(aggregate_id, aggregate_type, event_type, event_data) values (?,?,?,?)`, makeID(), "note", "NoteCreated", string(bytes))
	if err != nil {
		return fmt.Errorf("db.Exec: %w", err)
	}

	return nil
}

type DeleteCmd struct {
	ID string `arg:""`
}

func (c *DeleteCmd) Run(s *service.Service) error {
	aggID, err := s.GetAggregateID(strings.ToLower(c.ID))
	if err != nil {
		return err
	}

	_, err = s.DB.Exec(`insert into events(aggregate_id, aggregate_type, event_type, event_data) values (?,?,?,?)`, aggID, "note", "NoteDeleted", "{}")
	if err != nil {
		return fmt.Errorf("db.Exec: %w", err)
	}

	return nil
}

type UndeleteCmd struct {
	ID string `arg:""`
}

func (c *UndeleteCmd) Run(s *service.Service) error {
	aggID, err := s.GetAggregateID(strings.ToLower(c.ID))
	if err != nil {
		return err
	}

	_, err = s.DB.Exec(`insert into events(aggregate_id, aggregate_type, event_type, event_data) values (?,?,?,?)`, aggID, "note", "NoteUndeleted", "{}")
	if err != nil {
		return fmt.Errorf("db.Exec: %w", err)
	}

	return nil
}
