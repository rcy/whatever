package commands

type InternalCmd struct {
	Echo EchoCmd `cmd:""`
	SQL  SQLCmd  `cmd:""`
	ID   IDCmd   `cmd:""`
}
