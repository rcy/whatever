package app

import (
	"log"

	"github.com/rcy/whatever/app/notes"
	"github.com/rcy/whatever/commands"
	"github.com/rcy/whatever/flog"
)

type Service struct {
	Commands *commands.Service
	Events   *flog.Service
	Notes    *notes.Service
}

func New(cmds *commands.Service, events *flog.Service) *Service {
	notes, err := notes.Init()
	if err != nil {
		log.Fatal(err)
	}

	events.RegisterProjection(notes)

	err = events.Replay()
	if err != nil {
		log.Fatal(err)
	}

	return &Service{
		Commands: cmds,
		Events:   events,
		Notes:    notes,
	}
}
