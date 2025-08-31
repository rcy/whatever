package commands

import (
	"fmt"

	"github.com/rcy/whatever/commands/service"
)

type SQLCmd struct {
}

func (c *SQLCmd) Run(s *service.Service) error {
	fmt.Printf("sqlite3 '%s'\n", s.DBFile)
	return nil
}
