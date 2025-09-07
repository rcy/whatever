package evoke

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

type Event struct {
	EventID       int       `db:"event_id"`
	CreatedAt     time.Time `db:"created_at"`
	AggregateType string    `db:"aggregate_type"`
	AggregateID   string    `db:"aggregate_id"`
	EventType     string    `db:"event_type"`
	EventData     []byte    `db:"event_data"`
}

type Service struct {
	db       *sqlx.DB
	handlers map[string][]HandlerFunc
	Config   *Config
}

type Config struct {
	DBFile string
}

func ID() string {
	src := make([]byte, 20)
	_, _ = rand.Read(src)
	return fmt.Sprintf("%x", src)
}

func NewStore(config Config) (*Service, error) {
	err := os.MkdirAll(filepath.Dir(config.DBFile), 0755)
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", config.DBFile)
	if err != nil {
		return nil, err
	}

	// Tune connection pool
	db.SetMaxOpenConns(1) // SQLite supports one writer, so cap to 1
	db.SetMaxIdleConns(1)

	// Enable WAL mode for better concurrency and durability
	if _, err := db.Exec(`PRAGMA journal_mode = WAL;`); err != nil {
		return nil, fmt.Errorf("failed to enable WAL mode:", err)
	}

	// Optional performance pragmas (tweak based on needs):
	if _, err := db.Exec(`PRAGMA synchronous = NORMAL;`); err != nil {
		return nil, err
	}
	if _, err := db.Exec(`PRAGMA foreign_keys = ON;`); err != nil {
		return nil, err
	}

	if _, err := db.Exec(`
		create table if not exists events (
			event_id integer primary key autoincrement,
                        created_at timestamp not null default current_timestamp,
                        aggregate_type text not null,
                        aggregate_id text not null,
                        event_type text not null,
                        event_data text not null
		);
	`); err != nil {
		return nil, err
	}

	sqlxDB := sqlx.NewDb(db, "sqlite3")

	return &Service{
		db:       sqlxDB,
		Config:   &config,
		handlers: make(map[string][]HandlerFunc),
	}, nil
}

func (s *Service) Close() error {
	return s.db.Close()
}

type Inserter interface {
	Insert(aggregateID string, ed EventDefinition, payload any) error
	GetAggregateID(prefix string) (string, error)
}

type ExecGetter interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Get(dest interface{}, query string, args ...interface{}) error
}

type HandlerFunc func(event Event, inserter Inserter, replay bool) error

func (s *Service) Subscribe(ed EventDefinition, handler HandlerFunc) {
	s.handlers[ed.Name] = append(s.handlers[ed.Name], handler)
}

type Subscriber interface {
	Subscribe(EventDefinition, HandlerFunc)
}

type ProjectionRegisterer interface {
	Register(Subscriber)
}

func (s *Service) RegisterProjection(projection ProjectionRegisterer) {
	projection.Register(s)
}

func (s *Service) GetAggregateIDs(prefix string) ([]string, error) {
	var aggIDs []string
	query := fmt.Sprintf("%s%%", prefix)
	err := s.db.Select(&aggIDs, `select distinct aggregate_id from events where aggregate_id like ?`, query)
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

func (s *Service) LoadAggregateEvents(aggregateID string) ([]Event, error) {
	var events []Event
	err := s.db.Select(&events, `select * from events where aggregate_id = ? order by event_id asc`, aggregateID)
	if err != nil {
		return nil, err
	}
	return events, nil
}

func (s *Service) LoadAllEvents(reverse bool) ([]Event, error) {
	var events []Event

	var order string
	if reverse {
		order = "desc"
	} else {
		order = "asc"
	}
	err := s.db.Select(&events, fmt.Sprintf(`select * from events order by event_id %s`, order))
	if err != nil {
		return nil, err
	}
	return events, nil
}

func (s *Service) insertTx(e ExecGetter, aggregateID string, ed EventDefinition, payload any) error {
	if ed.PayloadType != reflect.TypeOf(payload) {
		return fmt.Errorf("payload type mismatch, want:%s got:%s", ed.PayloadType, reflect.TypeOf(payload))
	}

	bytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}

	var event Event
	err = e.Get(&event, `insert into events(aggregate_id, aggregate_type, event_type, event_data) values (?,?,?,?) returning *`,
		aggregateID,
		ed.Aggregate,
		ed.Name,
		string(bytes)) // Q: why string bytes? sqlite thing?
	if err != nil {
		return fmt.Errorf("db.Exec: %w", err)
	}

	err = s.runEventHandlers(e, event, false)
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) Insert(aggregateID string, ed EventDefinition, payload any) error {
	tx, err := s.db.Beginx()
	if err != nil {
		return fmt.Errorf("Begin: %w", err)
	}
	defer tx.Rollback()

	err = s.insertTx(tx, aggregateID, ed, payload)
	if err != nil {
		return fmt.Errorf("InsertTx: %w", err)
	}

	return tx.Commit()
}

type InsertTxWrapper struct {
	e ExecGetter
	s *Service
}

func (i InsertTxWrapper) Insert(aggregateID string, ed EventDefinition, payload any) error {
	return i.s.insertTx(i.e, aggregateID, ed, payload)
}

func (i InsertTxWrapper) GetAggregateID(prefix string) (string, error) {
	return i.s.GetAggregateID(prefix)
}

func (s *Service) runEventHandlers(e ExecGetter, event Event, replay bool) error {
	if handlers, ok := s.handlers[event.EventType]; ok {
		inserter := InsertTxWrapper{e: e, s: s}
		for _, handle := range handlers {
			err := handle(event, inserter, replay)
			if err != nil {
				return fmt.Errorf("handler for %s failed: %w", event.EventType, err)
			}
		}
	}
	return nil
}

func (s *Service) Replay() error {
	events := []Event{}
	if err := s.db.Select(&events, `select * from events order by event_id asc`); err != nil {
		return err
	}

	for i, event := range events {
		err := s.runEventHandlers(s.db, event, true)
		if err != nil {
			return fmt.Errorf("replay failed at event %d (%s): %w", i, event.EventType, err)
		}
	}
	return nil
}

func UnmarshalPayload[T any](event Event) (T, error) {
	var payload T
	err := json.Unmarshal([]byte(event.EventData), &payload)
	return payload, err
}
