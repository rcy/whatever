package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/rcy/whatever/app"
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
	type note struct {
		ID   string    `db:"id"`
		Text string    `db:"text"`
		Ts   time.Time `db:"ts"`
	}
	var notes []note
	err := app.ES.DBTodo.Select(&notes, `select * from notes order by ts asc`)
	if err != nil {
		return fmt.Errorf("Select notes: %w", err)
	}
	for _, note := range notes {
		fmt.Printf("%s %s %s\n", note.ID[0:7], note.Ts.Local().Format(time.DateTime), note.Text)
	}

	return nil
}

type AddCmd struct {
	Text []string `arg:""`
}

func (c *AddCmd) Run(app *app.Service) error {
	aggID, err := app.CS.CreateNote(strings.Join(c.Text, " "))
	fmt.Println(aggID)
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
