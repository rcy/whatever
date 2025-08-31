package commands

import (
	"fmt"
	"strings"

	"github.com/rcy/whatever/commands/service"
)

type EchoCmd struct {
	Arg []string `arg:"" optional:""`
}

func (c *EchoCmd) Run(s *service.Service) error {
	fmt.Println(strings.Join(c.Arg, " "))
	return nil
}
