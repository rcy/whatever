package commands

import "fmt"

type SQLCmd struct {
}

func (c *SQLCmd) Run(ctx *Context) error {
	fmt.Printf("sqlite3 '%s'\n", ctx.DBFile)
	return nil
}
