package cli

import (
	"fmt"
	"os"

	"github.com/rcy/whatever/app"
	"github.com/rcy/whatever/commands"
)

type ClassifyCmd struct{}

func (c *ClassifyCmd) Run(a *app.App) error {
	owner := os.Getenv("USER")

	notes, err := a.Notes.FindAllByCategory(owner, "inbox")
	if err != nil {
		return err
	}

	if len(notes) == 0 {
		fmt.Println("inbox is empty")
		return nil
	}

	for _, note := range notes {
		category, err := categorize(note.Text)
		if err != nil {
			return err
		}

		err = a.Commander.Send(commands.SetNoteCategory{
			NoteID:   note.ID,
			Category: category,
			Actor:    "ai",
		})
		if err != nil {
			return err
		}

		fmt.Printf("%q → %s\n", note.Text, category)
	}
	return nil
}
