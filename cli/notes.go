package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/rcy/whatever/app"
	"github.com/rcy/whatever/projections/notes"
)

type NotesCmd struct {
	List     ListCmd     `cmd:"" default:"withargs" aliases:"ls"`
	Show     ShowCmd     `cmd:""`
	Add      AddCmd      `cmd:""`
	Delete   DeleteCmd   `cmd:"" aliases:"rm"`
	Undelete UndeleteCmd `cmd:""`
	Edit     EditCmd     `cmd:""`
}

type ListCmd struct {
	Deleted bool `help:"Show deleted notes"`
}

func (c *ListCmd) Run(app *app.App) error {
	var noteList []notes.Note
	var err error
	if c.Deleted {
		noteList, err = app.Notes.FindAllDeleted()
	} else {
		noteList, err = app.Notes.FindAll()
	}
	if err != nil {
		return err
	}
	for _, note := range noteList {
		fmt.Printf("%s %s %s\n", note.ID[0:7], note.Category, note.Text)
	}

	return nil
}

type ShowCmd struct {
	ID string `arg:""`
}

func (c *ShowCmd) Run(app *app.App) error {
	id, _ := app.Events.GetAggregateID(c.ID)
	note, err := app.Notes.FindOne(id)
	eventList, err := app.Events.LoadAggregateEvents(id)
	if err != nil {
		return err
	}
	for _, e := range eventList {
		fmt.Printf("%7s %s %-15s %v\n", "", e.CreatedAt.Local().Format(time.DateTime), e.EventType, string(e.EventData))
	}

	fmt.Printf("%s %s %s\n", note.ID[0:7], note.Ts.Local().Format(time.DateTime), note.Text)
	return nil
}

type AddCmd struct {
	Text []string `arg:""`
}

func (c *AddCmd) Run(app *app.App) error {
	aggID, err := app.Commands.CreateNote(strings.Join(c.Text, " "))
	fmt.Println(aggID)
	return err
}

type EditCmd struct {
	ID   string   `arg:""`
	Text []string `arg:""`
}

func (c *EditCmd) Run(app *app.App) error {
	err := app.Commands.UpdateNoteText(c.ID, strings.Join(c.Text, " "))
	return err
}

type DeleteCmd struct {
	ID string `arg:""`
}

func (c *DeleteCmd) Run(app *app.App) error {
	return app.Commands.DeleteNote(c.ID)
}

type UndeleteCmd struct {
	ID string `arg:""`
}

func (c *UndeleteCmd) Run(app *app.App) error {
	return app.Commands.UndeleteNote(c.ID)
}
