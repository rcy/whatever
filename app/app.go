package app

import (
	"log"

	"github.com/rcy/evoke"
	"github.com/rcy/whatever/commands"
	"github.com/rcy/whatever/projections/notes"
)

type App struct {
	Events   *evoke.Service
	Commands *commands.Service
	Notes    *notes.Projection
}

func New(cmds *commands.Service, events *evoke.Service) *App {
	notes, err := notes.New(events)
	if err != nil {
		log.Fatal(err)
	}

	err = notes.Subscribe(events)
	if err != nil {
		log.Fatal(err)
	}

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
