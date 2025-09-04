package commands

import (
	"fmt"
	"time"

	"github.com/rcy/disco"
	"github.com/rcy/whatever/events"
)

type DDateCmd struct {
}

func (c *DDateCmd) Run(es *events.Service) error {
	fmt.Println(disco.NowIn(time.Local).Format(true))
	return nil
}
