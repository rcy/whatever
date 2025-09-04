package events

import (
	"encoding/json"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type Service struct {
	DBTodo *sqlx.DB
	DBFile string
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
	aggIDs, err := s.GetAggregateIDs(prefix)
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

func (s *Service) InsertEvent(eventType string, aggregateType string, aggregateID string, payload any) error {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}

	_, err = s.DBTodo.Exec(`insert into events(aggregate_id, aggregate_type, event_type, event_data) values (?,?,?,?)`,
		aggregateID,
		aggregateType,
		eventType, string(bytes))
	if err != nil {
		return fmt.Errorf("db.Exec: %w", err)
	}
	return nil
}
