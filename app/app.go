package app

import (
	"log"

	"github.com/rcy/whatever/app/notes"
	"github.com/rcy/whatever/commands"
	"github.com/rcy/whatever/flog"
)

type App struct {
	Commands *commands.Service
	Events   *flog.Service
	Notes    *notes.Projection
}

func New(cmds *commands.Service, events *flog.Service) *App {
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
