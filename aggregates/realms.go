package aggregates

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/rcy/evoke"
	"github.com/rcy/whatever/commands"
	"github.com/rcy/whatever/events"
)

type realmAggregate struct {
	id      uuid.UUID
	name    string
	deleted bool
}

func NewRealmAggregate(id uuid.UUID) *realmAggregate {
	return &realmAggregate{id: id}
}

func (a *realmAggregate) HandleCommand(cmd evoke.Command) ([]evoke.Event, error) {
	aggregateID := cmd.AggregateID()
	if aggregateID == uuid.Nil {
		return nil, fmt.Errorf("no aggregateID")
	}

	if a.id != aggregateID {
		panic("id mismatch")
	}

	switch c := cmd.(type) {
	case commands.CreateRealm:
		if c.RealmID == uuid.Nil {
			return nil, fmt.Errorf("realm cannot be empty")
		}

		name := strings.TrimSpace(c.Name)
		if name == "" {
			return nil, fmt.Errorf("name cannot be empty")
		}

		return []evoke.Event{
			events.RealmCreated{
				RealmID: aggregateID,
				Name:    name,
			},
		}, nil
	}
	return nil, fmt.Errorf("unhandled")
}

func (a *realmAggregate) Apply(e evoke.Event) error {
	switch evt := e.(type) {
	case events.RealmCreated:
		a.id = evt.RealmID
		a.name = evt.Name
	default:
		return fmt.Errorf("not handled")
	}
	return nil
}
