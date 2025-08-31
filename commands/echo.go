package commands

import (
	"fmt"
	"strings"
)

type EchoCmd struct {
	Arg []string `arg:"" optional:""`
}

func (c *EchoCmd) Run(ctx *Context) error {
	fmt.Println(strings.Join(c.Arg, " "))
	return nil
}
