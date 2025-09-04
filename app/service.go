package app

import (
	"github.com/rcy/whatever/commands"
	"github.com/rcy/whatever/events"
)

type Service struct {
	CS *commands.Service
	ES *events.Service
}

func New(cs *commands.Service, es *events.Service) *Service {
	return &Service{CS: cs, ES: es}
}
