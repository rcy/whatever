package app

import (
	"log"

	"github.com/rcy/whatever/app/notes"
	"github.com/rcy/whatever/commands"
	"github.com/rcy/whatever/events"
)

type Service struct {
	NS *notes.Service
	CS *commands.Service
	ES *events.Service
}

func New(cs *commands.Service, es *events.Service) *Service {
	ns, err := notes.Init(es)
	if err != nil {
		log.Fatal(err)
	}

	err = es.ReplayEvents()
	if err != nil {
		log.Fatal(err)
	}

	return &Service{NS: ns, CS: cs, ES: es}
}
