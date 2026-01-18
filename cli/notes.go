package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rcy/whatever/app"
	"github.com/rcy/whatever/commands"
	"github.com/rcy/whatever/projections/note"
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
	var noteList []note.Note
	var err error
	if c.Deleted {
		noteList, err = app.Notes.FindAllDeleted()
	} else {
		noteList, err = app.Notes.FindAll("FIXME OWNER")
	}
	if err != nil {
		return err
	}
	for _, note := range noteList {
		fmt.Printf("%s %s %s %s\n", note.ID, note.RealmID, note.Category, note.Text)
	}
	return nil
}

type ShowCmd struct {
	ID string `arg:""`
}

func (c *ShowCmd) Run(app *app.App) error {
	note, err := app.Notes.FindOne(c.ID)
	if err != nil {
		return err
	}
	fmt.Printf("%s %s %s\n", note.ID[0:7], note.Ts.Local().Format(time.DateTime), note.Text)
	return nil
}

type AddCmd struct {
	Realm string
	Text  []string `arg:""`
}

func (c *AddCmd) Run(app *app.App) error {
	realm, err := app.Realms.FindByName(c.Realm)
	if err != nil {
		return err
	}

	noteID := uuid.New()
	err = app.Commander.Send(commands.CreateNote{
		NoteID:  noteID,
		RealmID: realm.ID,
		Text:    strings.Join(c.Text, " "),
	})
	if err != nil {
		return err
	}
	fmt.Println(noteID)
	return nil
}

type EditCmd struct {
	NoteID uuid.UUID `arg:""`
	Text   []string  `arg:""`
}

func (c *EditCmd) Run(app *app.App) error {
	return app.Commander.Send(commands.UpdateNoteText{
		NoteID: c.NoteID,
		Text:   strings.Join(c.Text, " "),
	})
}

type DeleteCmd struct {
	ID uuid.UUID `arg:""`
}

func (c *DeleteCmd) Run(app *app.App) error {
	return app.Commander.Send(commands.DeleteNote{
		NoteID: c.ID,
	})
}

type UndeleteCmd struct {
	ID uuid.UUID `arg:""`
}

func (c *UndeleteCmd) Run(app *app.App) error {
	return app.Commander.Send(commands.UndeleteNote{
		NoteID: c.ID,
	})
}
