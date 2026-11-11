package aggregates

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rcy/evoke"
	"github.com/rcy/whatever/commands"
	"github.com/rcy/whatever/events"
)

type webhookAggregate struct {
	id uuid.UUID
}

func NewWebhookAggregate(id uuid.UUID) *webhookAggregate {
	return &webhookAggregate{id: id}
}

func (a *webhookAggregate) HandleCommand(cmd evoke.Command) ([]evoke.Event, error) {
	switch c := cmd.(type) {
	case commands.ReceiveWebhook:
		return []evoke.Event{
			events.WebhookReceived{
				WebhookID:  c.WebhookID,
				Owner:      c.Owner,
				ReceivedAt: time.Now(),
				Path:       c.Path,
				Fields:     c.Fields,
			},
		}, nil
	}
	return nil, fmt.Errorf("unhandled")
}

func (a *webhookAggregate) Apply(e evoke.Event) error {
	switch e.(type) {
	case events.WebhookReceived:
		return nil
	}
	return fmt.Errorf("not handled")
}
