package commands

import (
	"fmt"
	"runtime/debug"

	"github.com/jmoiron/sqlx"
)

type Context struct {
	DB     *sqlx.DB
	DBFile string
}

func (c *Context) GetAggregateIDs(prefix string) ([]string, error) {
	var aggIDs []string
	query := fmt.Sprintf("%s%%", prefix)
	err := c.DB.Select(&aggIDs, `select distinct aggregate_id from events where aggregate_id like ?`, query)
	if err != nil {
		return nil, fmt.Errorf("Select: %w", err)
	}
	return aggIDs, nil
}

func (c *Context) GetAggregateID(prefix string) (string, error) {
	aggIDs, err := c.GetAggregateIDs(prefix)
	if err != nil {
		return "", fmt.Errorf("GetAggregateID: %w", err)
	}

	if len(aggIDs) == 0 {
		return "", fmt.Errorf("ID not found")
	}
	if len(aggIDs) > 1 {
		return "", fmt.Errorf("ID is ambiguous: %s", aggIDs)
	}
	return aggIDs[0], nil
}

var CLI struct {
	Version VersionCmd `cmd:""`
	Notes   NotesCmd   `cmd:""`
	Events  EventsCmd  `cmd:""`
	Ddate   DDateCmd   `cmd:""`
	Debug   DebugCmd   `cmd:""`
}

type VersionCmd struct{}

func (c *VersionCmd) Run() error {
	if info, ok := debug.ReadBuildInfo(); ok {
		fmt.Println(info.Main.Version)
	} else {
		fmt.Println("unknown")
	}
	return nil
}
