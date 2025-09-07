package cli

import (
	"fmt"
	"time"

	"github.com/rcy/disco"
	"github.com/rcy/whatever/app"
)

type DDateCmd struct {
}

func (c *DDateCmd) Run(app *app.App) error {
	fmt.Println(disco.NowIn(time.Local).Format(true))
	return nil
}
