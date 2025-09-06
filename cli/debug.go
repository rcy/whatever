package cli

type DebugCmd struct {
	Echo EchoCmd `cmd:"" help:"Echo the args back to stdout"`
	ID   IDCmd   `cmd:"" help:"Generate a random aggregate id"`
}
