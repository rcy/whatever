package app

import (
	"log"

	"github.com/rcy/whatever/commands"
	"github.com/rcy/whatever/evoke"
	"github.com/rcy/whatever/projections/notes"
)

type App struct {
	Events   *evoke.Service
	Commands *commands.Service
	Notes    *notes.Projection
}

func New(cmds *commands.Service, events *evoke.Service) *App {
	notes, err := notes.New()
	if err != nil {
		log.Fatal(err)
	}

	events.RegisterProjection(notes)

	err = events.Replay()
	if err != nil {
		log.Fatal(err)
	}

	return &App{
		Commands: cmds,
		Events:   events,
		Notes:    notes,
	}
}
