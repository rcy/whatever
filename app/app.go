package app

import (
	"log"

	"github.com/rcy/evoke"
	"github.com/rcy/whatever/commands"
	"github.com/rcy/whatever/projections/notes"
	"github.com/rcy/whatever/projections/realms"
)

type App struct {
	Events   *evoke.Service
	Commands *commands.Service
	Notes    *notes.Projection
	Realms   *realms.Projection
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

	realms, err := realms.New(events)
	if err != nil {
		log.Fatal(err)
	}

	err = realms.Subscribe(events)
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
		Realms:   realms,
	}
}
