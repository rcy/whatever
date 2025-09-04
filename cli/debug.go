package cli

type DebugCmd struct {
	Echo EchoCmd `cmd:"" help:"Echo the args back to stdout"`
	SQL  SQLCmd  `cmd:"" help:"Show where the sqlite database is stored and how to connect using sqlite3"`
	ID   IDCmd   `cmd:"" help:"Generate a random aggregate id"`
}
