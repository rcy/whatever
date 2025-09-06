package cli

import (
	"github.com/rcy/whatever/cli/reminders"
)

var CLI struct {
	Version   VersionCmd    `cmd:""`
	Notes     NotesCmd      `cmd:""`
	Reminders reminders.Cmd `cmd:""`
	Events    EventsCmd     `cmd:""`
	Ddate     DDateCmd      `cmd:""`
	Debug     DebugCmd      `cmd:""`
	Serve     ServeCmd      `cmd:""`
}
