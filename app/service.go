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

func New(cs *commands.Service, es *flog.Service) *Service {
	ns, err := notes.Init(es)
	if err != nil {
		log.Fatal(err)
	}

	err = es.ReplayEvents()
	if err != nil {
		log.Fatal(err)
	}

	return &Service{Notes: ns, Commands: cs, Events: es}
}
