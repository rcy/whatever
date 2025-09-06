package cli

var CLI struct {
	Version VersionCmd `cmd:""`
	Notes   NotesCmd   `cmd:""`
	Events  EventsCmd  `cmd:""`
	Ddate   DDateCmd   `cmd:""`
	Debug   DebugCmd   `cmd:""`
	Serve   ServeCmd   `cmd:""`
	Bug     BugCmd     `cmd:""`
}
