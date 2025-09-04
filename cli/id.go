package cli

import (
	"fmt"

	"github.com/rcy/whatever/app"
	"github.com/rcy/whatever/ids"
)

type IDCmd struct {
	Aggregate string `arg:"" default:"note"`
}

func (c *IDCmd) Run(app *app.Service) error {
	fmt.Println(ids.New())
	return nil
}
