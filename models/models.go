package models

import "time"

type Event struct {
	EventID       int       `db:"event_id"`
	CreatedAt     time.Time `db:"created_at"`
	AggregateType string    `db:"aggregate_type"`
	AggregateID   string    `db:"aggregate_id"`
	EventType     string    `db:"event_type"`
	EventData     []byte    `db:"event_data"`
}
