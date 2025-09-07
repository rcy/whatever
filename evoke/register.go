package evoke

import (
	"reflect"
)

type EventDefinition struct {
	Name        string
	Aggregate   string
	PayloadType reflect.Type
}

func RegisterEvent(name string, aggregate string, payload any) EventDefinition {
	return EventDefinition{
		Name:        name,
		Aggregate:   aggregate,
		PayloadType: reflect.TypeOf(payload),
	}
}
