package commands

import (
	"fmt"

	"github.com/rcy/whatever/commands/service"
	"github.com/rcy/whatever/ids"
)

type IDCmd struct {
	Aggregate string `arg:"" default:"note"`
}

func (c *IDCmd) Run(s *service.Service) error {
	fmt.Println(ids.New())
	return nil
}
