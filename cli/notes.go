package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rcy/whatever/app"
	"github.com/rcy/whatever/models"
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

func (c *ListCmd) Run(app *app.Service) error {
	type id string
	type note struct {
		text    string
		deleted bool
	}
	notes := make(map[id]note)
	var events []models.Event
	err := app.ES.DBTodo.Select(&events, `select * from events where aggregate_type = 'note' order by created_at asc`)
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

func (c *AddCmd) Run(app *app.Service) error {
	_, err := app.CS.CreateNote(strings.Join(c.Text, " "))
	return err
}

type DeleteCmd struct {
	ID string `arg:""`
}

func (c *DeleteCmd) Run(app *app.Service) error {
	return app.CS.DeleteNote(c.ID)
}

type UndeleteCmd struct {
	ID string `arg:""`
}

func (c *UndeleteCmd) Run(app *app.Service) error {
	return app.CS.UndeleteNote(c.ID)
}
