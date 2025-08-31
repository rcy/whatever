package commands

import (
	"fmt"
	"strings"
	"time"
)

type EventsCmd struct {
	ID string
}

type Event struct {
	EventID       int       `db:"event_id"`
	CreatedAt     time.Time `db:"created_at"`
	AggregateType string    `db:"aggregate_type"`
	AggregateID   string    `db:"aggregate_id"`
	EventType     string    `db:"event_type"`
	EventData     []byte    `db:"event_data"`
}

func (c *EventsCmd) Run(ctx *Context) error {
	var events []Event
	if c.ID != "" {
		aggID, err := ctx.GetAggregateID(strings.ToLower(c.ID))
		if err != nil {
			return err
		}
		err = ctx.DB.Select(&events, `select * from events where aggregate_id = ? order by event_id `, aggID)
		if err != nil {
			return err
		}
	} else {

		err := ctx.DB.Select(&events, `select * from events order by event_id`)
		if err != nil {
			return fmt.Errorf("Select: %w", err)
		}
	}

	for _, e := range events {
		fmt.Printf("%s %-14s %s %s\n", e.AggregateID[0:5], e.EventType, e.CreatedAt.Format(time.RFC3339), e.EventData)
	}
	return nil
}
