package cli

import (
	"fmt"

	"github.com/rcy/whatever/app"
	"github.com/rcy/whatever/version"
)

type VersionCmd struct{}

func (c *VersionCmd) Run(app *app.App) error {
	fmt.Printf("version=%s isRelease=%v dbFile=%s\n", version.Version(), version.IsRelease(), app.Events.Config.DBFile)
	return nil
}
