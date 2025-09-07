package cli

import (
	"fmt"
	"strings"

	"github.com/rcy/whatever/app"
)

type EchoCmd struct {
	Arg []string `arg:"" optional:""`
}

func (c *EchoCmd) Run(app *app.App) error {
	fmt.Println(strings.Join(c.Arg, " "))
	return nil
}
