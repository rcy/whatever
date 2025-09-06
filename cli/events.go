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

func (c *EventsCmd) Run(app *app.Service) error {
	var events []flog.Model
	if c.ID != "" {
		aggID, err := app.ES.GetAggregateID(c.ID)
		if err != nil {
			return err
		}
		events, err = app.ES.LoadAggregateEvents(aggID)
		if err != nil {
			return err
		}
	} else {
		var err error
		events, err = app.ES.LoadAllEvents()
		if err != nil {
			return err
		}
	}

	for _, e := range events {
		fmt.Printf("%s %-14s %s %s\n", e.AggregateID[0:5], e.EventType, e.CreatedAt.Format(time.RFC3339), e.EventData)
	}
	return nil
}
