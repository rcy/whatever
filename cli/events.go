package cli

import (
	"fmt"
	"time"

	"github.com/rcy/whatever/app"
	"github.com/rcy/whatever/flog"
)

type EventsCmd struct {
	ID string
}

func (c *EventsCmd) Run(app *app.App) error {
	var events []flog.Event
	if c.ID != "" {
		aggID, err := app.Events.GetAggregateID(c.ID)
		if err != nil {
			return err
		}
		events, err = app.Events.LoadAggregateEvents(aggID)
		if err != nil {
			return err
		}
	} else {
		var err error
		events, err = app.Events.LoadAllEvents(true)
		if err != nil {
			return err
		}
	}

	for _, e := range events {
		fmt.Printf("%d\t%s\t%s\t%s\t%s\n", e.EventID, e.AggregateID[0:5], e.EventType, e.CreatedAt.Format(time.RFC3339), e.EventData)
	}
	return nil
}
