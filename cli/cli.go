package cli

import (
	"github.com/rcy/whatever/cli/reminders"
	"github.com/rcy/whatever/cli/web"
)

var CLI struct {
	Version   VersionCmd    `cmd:""`
	Notes     NotesCmd      `cmd:""`
	Reminders reminders.Cmd `cmd:""`
	Events    EventsCmd     `cmd:""`
	Ddate     DDateCmd      `cmd:""`
	Debug     DebugCmd      `cmd:""`
	Serve     web.ServeCmd  `cmd:""`
}
