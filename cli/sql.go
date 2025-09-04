package cli

import (
	"fmt"

	"github.com/rcy/whatever/app"
)

type SQLCmd struct {
}

func (c *SQLCmd) Run(app *app.Service) error {
	fmt.Printf("sqlite3 '%s'\n", app.ES.DBFile)
	return nil
}
