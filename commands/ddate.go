package commands

import (
	"fmt"
	"time"

	"github.com/rcy/disco"
	"github.com/rcy/whatever/commands/service"
)

type DDateCmd struct {
}

func (c *DDateCmd) Run(s *service.Service) error {
	fmt.Println(disco.NowIn(time.Local).Format(true))
	return nil
}
