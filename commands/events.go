package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/rcy/whatever/events"
	"github.com/rcy/whatever/models"
)

type EventsCmd struct {
	ID string
}

func (c *EventsCmd) Run(es *events.Service) error {
	var events []models.Event
	if c.ID != "" {
		aggID, err := es.GetAggregateID(strings.ToLower(c.ID))
		if err != nil {
			return err
		}
		err = es.DBTodo.Select(&events, `select * from events where aggregate_id = ? order by event_id `, aggID)
		if err != nil {
			return err
		}
	} else {
		err := es.DBTodo.Select(&events, `select * from events order by event_id`)
		if err != nil {
			return fmt.Errorf("Select: %w", err)
		}
	}

	for _, e := range events {
		fmt.Printf("%s %-14s %s %s\n", e.AggregateID[0:5], e.EventType, e.CreatedAt.Format(time.RFC3339), e.EventData)
	}
	return nil
}
