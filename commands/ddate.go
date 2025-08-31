package commands

import (
	"fmt"
	"time"

	"github.com/rcy/disco"
)

type DDateCmd struct {
}

func (c *DDateCmd) Run(ctx *Context) error {
	fmt.Println(disco.NowIn(time.Local).Format(true))
	return nil
}
