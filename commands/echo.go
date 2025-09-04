package commands

import (
	"fmt"
	"strings"

	"github.com/rcy/whatever/events"
)

type EchoCmd struct {
	Arg []string `arg:"" optional:""`
}

func (c *EchoCmd) Run(es *events.Service) error {
	fmt.Println(strings.Join(c.Arg, " "))
	return nil
}
