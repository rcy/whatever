package events

import (
	"database/sql"
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

type EventInserter interface {
	InsertEvent(eventType string, aggregateType string, aggregateID string, payload any) error
}

type ExecGetter interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Get(dest interface{}, query string, args ...interface{}) error
}

type HandlerFunc func(i EventInserter, event Model, replay bool) error

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

func (s *Service) InsertEventTx(e ExecGetter, eventType string, aggregateType string, aggregateID string, payload any) error {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}

	var event Model
	err = e.Get(&event, `insert into events(aggregate_id, aggregate_type, event_type, event_data) values (?,?,?,?) returning *`,
		aggregateID,
		aggregateType,
		eventType, string(bytes))
	if err != nil {
		return fmt.Errorf("db.Exec: %w", err)
	}

	err = s.runEventHandlers(e, event, false)
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) InsertEvent(eventType string, aggregateType string, aggregateID string, payload any) error {
	tx, err := s.DBTodo.Beginx()
	if err != nil {
		return fmt.Errorf("Begin: %w", err)
	}
	defer tx.Rollback()

	err = s.InsertEventTx(tx, eventType, aggregateType, aggregateID, payload)
	if err != nil {
		return fmt.Errorf("InsertEventTx: %w", err)
	}

	return tx.Commit()
}

type InsertEventTxWrapper struct {
	e ExecGetter
	s *Service
}

func (i InsertEventTxWrapper) InsertEvent(eventType string, aggregateType string, aggregateID string, payload any) error {
	return i.s.InsertEventTx(i.e, eventType, aggregateType, aggregateID, payload)
}

func (s *Service) runEventHandlers(e ExecGetter, event Model, replay bool) error {
	if handlers, ok := s.handlers[event.EventType]; ok {
		for _, h := range handlers {
			w := InsertEventTxWrapper{e: e, s: s}

			err := h(w, event, replay)
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
		err := s.runEventHandlers(s.DBTodo, event, true)
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
