package app

import (
	"log"

	"github.com/rcy/evoke"
	"github.com/rcy/whatever/commands"
	"github.com/rcy/whatever/projections/notes"
	"github.com/rcy/whatever/projections/realms"
	"github.com/rcy/whatever/sagas"
)

type App struct {
	events   *evoke.Service
	commands *commands.Service
	notes    *notes.Projection
	realms   *realms.Projection
}

func (a *App) Events() *evoke.Service      { return a.events }
func (a *App) Commands() *commands.Service { return a.commands }
func (a *App) Notes() *notes.Projection    { return a.notes }
func (a *App) Realms() *realms.Projection  { return a.realms }

func New(cmds *commands.Service, events *evoke.Service) *App {
	notes, err := notes.New(events)
	if err != nil {
		log.Fatal(err)
	}
	notes.Subscribe(events)

	realms, err := realms.New(events)
	if err != nil {
		log.Fatal(err)
	}
	realms.Subscribe(events)

	err = events.Replay()
	if err != nil {
		log.Fatal(err)
	}

	app := &App{
		commands: cmds,
		events:   events,
		notes:    notes,
		realms:   realms,
	}

	sagas.New(app).Subscribe()

	return app
}
