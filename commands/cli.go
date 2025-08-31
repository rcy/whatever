package commands

import (
	"fmt"
	"runtime/debug"

	"github.com/rcy/whatever/commands/web"
)

var CLI struct {
	Version VersionCmd   `cmd:""`
	Notes   NotesCmd     `cmd:""`
	Events  EventsCmd    `cmd:""`
	Ddate   DDateCmd     `cmd:""`
	Debug   DebugCmd     `cmd:""`
	Serve   web.ServeCmd `cmd:""`
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
