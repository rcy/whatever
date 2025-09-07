package cli

var CLI struct {
	Version VersionCmd `cmd:"" help:"show the build version"`
	Notes   NotesCmd   `cmd:""`
	Events  EventsCmd  `cmd:"" help:"dump the event log"`
	Ddate   DDateCmd   `cmd:"" help:"show current discordian date"`
	Serve   ServeCmd   `cmd:"" help:"start a webserver"`
	Bug     BugCmd     `cmd:"" help:"report a bug"`
}
