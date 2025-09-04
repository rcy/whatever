package commands

import (
	"fmt"

	"github.com/rcy/whatever/events"
)

type SQLCmd struct {
}

func (c *SQLCmd) Run(es *events.Service) error {
	fmt.Printf("sqlite3 '%s'\n", es.DBFile)
	return nil
}
