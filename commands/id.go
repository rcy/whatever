package commands

import (
	"fmt"

	"github.com/rcy/whatever/events"
	"github.com/rcy/whatever/ids"
)

type IDCmd struct {
	Aggregate string `arg:"" default:"note"`
}

func (c *IDCmd) Run(es *events.Service) error {
	fmt.Println(ids.New())
	return nil
}
