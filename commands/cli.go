package commands

import (
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
