package river

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"path/filepath"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riversqlite"
	"github.com/riverqueue/river/rivermigrate"
	_ "modernc.org/sqlite"
	"riverqueue.com/riverui"
)

type Client struct {
	client *river.Client[*sql.Tx]
	db     *sql.DB
	ui     *riverui.Handler
}

// New creates a River client backed by a SQLite database stored next to
// dataDir. The database is created if it doesn't exist, and migrations
// are applied automatically. The caller provides the workers bundle with
// all job workers already registered.
func New(dataDir string, workers *river.Workers) (*Client, error) {
	dbPath := filepath.Join(dataDir, "river.db")
	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(wal)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("river: open db: %w", err)
	}
	db.SetMaxOpenConns(1)

	driver := riversqlite.New(db)

	migrator, err := rivermigrate.New(driver, nil)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("river: new migrator: %w", err)
	}

	_, err = migrator.Migrate(context.Background(), rivermigrate.DirectionUp, nil)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("river: migrate: %w", err)
	}

	riverClient, err := river.NewClient(driver, &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 2},
		},
		Workers: workers,
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("river: new client: %w", err)
	}

	endpoints := riverui.NewEndpoints(riverClient, nil)
	uiHandler, err := riverui.NewHandler(&riverui.HandlerOpts{
		Endpoints: endpoints,
		Logger:    slog.Default(),
		Prefix:    "/river",
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("river: new ui handler: %w", err)
	}

	return &Client{
		client: riverClient,
		db:     db,
		ui:     uiHandler,
	}, nil
}

func (c *Client) Insert(ctx context.Context, args river.JobArgs, opts *river.InsertOpts) error {
	_, err := c.client.Insert(ctx, args, opts)
	return err
}

func (c *Client) UIHandler() http.Handler {
	return c.ui
}

func (c *Client) Start(ctx context.Context) error {
	if err := c.ui.Start(ctx); err != nil {
		return fmt.Errorf("river ui start: %w", err)
	}
	return c.client.Start(ctx)
}

func (c *Client) Stop(ctx context.Context) error {
	err := c.client.Stop(ctx)
	c.db.Close()
	return err
}
