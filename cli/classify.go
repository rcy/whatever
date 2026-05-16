package cli

import (
	"fmt"
	"os"

	"github.com/rcy/whatever/app"
	"github.com/rcy/whatever/commands"
	"github.com/rcy/whatever/workers/classify"
)

type ClassifyCmd struct{}

func (c *ClassifyCmd) Run(a *app.App) error {
	ownerID := os.Getenv("OWNER_ID")

	notes, err := a.Notes.FindAllByCategory(ownerID, "inbox")
	if err != nil {
		return err
	}

	if len(notes) == 0 {
		fmt.Println("inbox is empty")
		return nil
	}

	for _, note := range notes {
		category, err := classify.Categorize(note.Text)
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
