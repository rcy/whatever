package events

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/rcy/evoke"
	"github.com/rcy/whatever/app"
)

type Cmd struct {
	List ListCmd `cmd:""`
}

type ListCmd struct {
	AggregateID uuid.UUID
}

func (c *ListCmd) Run(app *app.App) error {
	var events []evoke.RecordedEvent
	if c.AggregateID != uuid.Nil {
		var err error
		events, err = app.Events().LoadStream(c.AggregateID)
		if err != nil {
			return err
		}
	} else {
		var err error
		events, err = app.Events().DebugEvents()
		if err != nil {
			return err
		}
	}

	for _, e := range events {
		fmt.Printf("%d\t%s\t%s\t%s\t%s\n", e.Sequence, e.AggregateID.String(), e.Event.EventType(), e.Event)
	}
	return nil
}
