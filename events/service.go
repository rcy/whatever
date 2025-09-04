package events

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

type Model struct {
	EventID       int       `db:"event_id"`
	CreatedAt     time.Time `db:"created_at"`
	AggregateType string    `db:"aggregate_type"`
	AggregateID   string    `db:"aggregate_id"`
	EventType     string    `db:"event_type"`
	EventData     []byte    `db:"event_data"`
}

type Service struct {
	DBTodo   *sqlx.DB
	DBFile   string
	handlers map[string][]HandlerFunc
}

func New(db *sqlx.DB, dbFile string) *Service {
	return &Service{
		DBTodo:   db,
		DBFile:   dbFile,
		handlers: make(map[string][]HandlerFunc),
	}
}

type HandlerFunc func(e sqlx.Execer, event Model) error

func (s *Service) RegisterHandler(eventType string, handler HandlerFunc) {
	s.handlers[eventType] = append(s.handlers[eventType], handler)
}

func (s *Service) GetAggregateIDs(prefix string) ([]string, error) {
	var aggIDs []string
	query := fmt.Sprintf("%s%%", prefix)
	err := s.DBTodo.Select(&aggIDs, `select distinct aggregate_id from events where aggregate_id like ?`, query)
	if err != nil {
		return nil, fmt.Errorf("Select: %w", err)
	}
	return aggIDs, nil
}

func (s *Service) GetAggregateID(prefix string) (string, error) {
	aggIDs, err := s.GetAggregateIDs(strings.ToLower(prefix))
	if err != nil {
		return "", fmt.Errorf("GetAggregateID: %w", err)
	}

	if len(aggIDs) == 0 {
		return "", fmt.Errorf("ID not found")
	}
	if len(aggIDs) > 1 {
		return "", fmt.Errorf("ID is ambiguous: %s", aggIDs)
	}
	return aggIDs[0], nil
}

func (s *Service) LoadAggregateEvents(aggregateID string) ([]Model, error) {
	var events []Model
	err := s.DBTodo.Select(&events, `select * from events where aggregate_id = ? order by event_id asc`, aggregateID)
	if err != nil {
		return nil, err
	}
	return events, nil
}

func (s *Service) InsertEvent(eventType string, aggregateType string, aggregateID string, payload any) error {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}

	tx, err := s.DBTodo.Beginx()
	if err != nil {
		return fmt.Errorf("Begin: %w", err)
	}
	defer tx.Rollback()

	var event Model
	err = tx.Get(&event, `insert into events(aggregate_id, aggregate_type, event_type, event_data) values (?,?,?,?) returning *`,
		aggregateID,
		aggregateType,
		eventType, string(bytes))
	if err != nil {
		return fmt.Errorf("db.Exec: %w", err)
	}

	err = s.runEventHandlers(tx, event)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Service) runEventHandlers(e sqlx.Execer, event Model) error {
	if handlers, ok := s.handlers[event.EventType]; ok {
		for _, h := range handlers {
			err := h(e, event)
			if err != nil {
				return fmt.Errorf("handler for %s failed: %w", event.EventType, err)
			}
		}
	}
	return nil
}

func (s *Service) ReplayEvents() error {
	events := []Model{}
	if err := s.DBTodo.Select(&events, `select * from events order by event_id asc`); err != nil {
		return err
	}

	for i, event := range events {
		err := s.runEventHandlers(s.DBTodo, event)
		if err != nil {
			return fmt.Errorf("replay failed at event %d (%s): %w", i, event.EventType, err)
		}
	}
	return nil
}

func UnmarshalPayload[T any](event Model) (T, error) {
	var payload T
	err := json.Unmarshal([]byte(event.EventData), &payload)
	return payload, err
}
