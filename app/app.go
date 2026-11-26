package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/rcy/evoke"
	"github.com/rcy/whatever/aggregates"
	"github.com/rcy/whatever/commands"
	"github.com/rcy/whatever/events"
	"github.com/rcy/whatever/projections/note"
	appriver "github.com/rcy/whatever/river"
	"github.com/rcy/whatever/workers/classify"
	"github.com/rcy/whatever/workers/enrich"
	"github.com/rcy/whatever/workers/webhook"
	"github.com/rcy/whatever/workers/bskysync"
	"github.com/rcy/whatever/workers/youtubesync"
	"github.com/riverqueue/river"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type App struct {
	Commander     evoke.CommandSender
	Notes         *note.Projection
	River         *appriver.Client
	DataDir       string
	EventDebugger interface {
		DebugEvents() ([]evoke.RecordedEvent, error)
	}
}

type Config struct {
	GoogleClientID     string
	GoogleClientSecret string
}

func New(filename string, cfg Config) (*App, error) {
	if filename == "" {
		return nil, errors.New("filename is empty")
	}

	eventStore, err := evoke.NewFileStore(filename)
	if err != nil {
		log.Fatal(err)
	}
	evoke.RegisterEvent(eventStore, &events.NoteCreated{})
	evoke.RegisterEvent(eventStore, &events.NoteOwnerSet{})
	evoke.RegisterEvent(eventStore, &events.NoteEnrichmentRequested{})
	evoke.RegisterEvent(eventStore, &events.RealmCreated{}) // deprecated
	evoke.RegisterEvent(eventStore, &events.NoteDeleted{})
	evoke.RegisterEvent(eventStore, &events.NoteUndeleted{})
	evoke.RegisterEvent(eventStore, &events.NoteTextUpdated{})
	evoke.RegisterEvent(eventStore, &events.NoteCategoryChanged{})
	evoke.RegisterEvent(eventStore, &events.NoteSubcategoryChanged{})
	evoke.RegisterEvent(eventStore, &events.NoteDueChanged{})
	evoke.RegisterEvent(eventStore, &events.NoteDueCleared{})
	evoke.RegisterEvent(eventStore, &events.NoteEnriched{})
	evoke.RegisterEvent(eventStore, &events.NoteEnrichmentFailed{})
	evoke.RegisterEvent(eventStore, &events.NoteStarred{})
	evoke.RegisterEvent(eventStore, &events.NoteUnstarred{})
	evoke.RegisterEvent(eventStore, &events.NoteTagAdded{})
	evoke.RegisterEvent(eventStore, &events.NoteTagRemoved{})
	evoke.RegisterEvent(eventStore, &events.WebhookReceived{})

	//
	// COMMANDS
	//
	commandBus := evoke.NewCommandBus()

	noteFactory := func(id uuid.UUID) evoke.Aggregate { return aggregates.NewNoteAggregate(id) }
	noteHandler := evoke.NewAggregateHandler(eventStore, noteFactory)
	commandBus.RegisterHandler(commands.CreateNote{}, noteHandler)
	commandBus.RegisterHandler(commands.SetNoteOwner{}, noteHandler)
	commandBus.RegisterHandler(commands.DeleteNote{}, noteHandler)
	commandBus.RegisterHandler(commands.UndeleteNote{}, noteHandler)
	commandBus.RegisterHandler(commands.UpdateNoteText{}, noteHandler)
	commandBus.RegisterHandler(commands.SetNoteCategory{}, noteHandler)
	commandBus.RegisterHandler(commands.TransitionNoteSubcategory{}, noteHandler)
	commandBus.RegisterHandler(commands.SetNoteDue{}, noteHandler)
	commandBus.RegisterHandler(commands.ClearNoteDue{}, noteHandler)
	commandBus.RegisterHandler(commands.CompleteNoteEnrichment{}, noteHandler)
	commandBus.RegisterHandler(commands.FailNoteEnrichment{}, noteHandler)
	commandBus.RegisterHandler(commands.StarNote{}, noteHandler)
	commandBus.RegisterHandler(commands.UnstarNote{}, noteHandler)
	commandBus.RegisterHandler(commands.AddNoteTag{}, noteHandler)
	commandBus.RegisterHandler(commands.RemoveNoteTag{}, noteHandler)

	webhookFactory := func(id uuid.UUID) evoke.Aggregate { return aggregates.NewWebhookAggregate(id) }
	webhookHandler := evoke.NewAggregateHandler(eventStore, webhookFactory)
	commandBus.RegisterHandler(commands.ReceiveWebhook{}, webhookHandler)

	//
	// PROJECTIONS
	//
	eventBus := evoke.NewEventBus()

	noteProjection, err := note.New()
	if err != nil {
		log.Fatal(err)
	}
	eventBus.Subscribe(events.NoteCreated{}, noteProjection)
	eventBus.Subscribe(events.NoteOwnerSet{}, noteProjection)
	eventBus.Subscribe(events.NoteDeleted{}, noteProjection)
	eventBus.Subscribe(events.NoteUndeleted{}, noteProjection)
	eventBus.Subscribe(events.NoteTextUpdated{}, noteProjection)
	eventBus.Subscribe(events.NoteCategoryChanged{}, noteProjection)
	eventBus.Subscribe(events.NoteSubcategoryChanged{}, noteProjection)
	eventBus.Subscribe(events.NoteDueChanged{}, noteProjection)
	eventBus.Subscribe(events.NoteDueCleared{}, noteProjection)
	eventBus.Subscribe(events.NoteEnrichmentRequested{}, noteProjection)
	eventBus.Subscribe(events.NoteEnriched{}, noteProjection)
	eventBus.Subscribe(events.NoteEnrichmentFailed{}, noteProjection)
	eventBus.Subscribe(events.NoteStarred{}, noteProjection)
	eventBus.Subscribe(events.NoteUnstarred{}, noteProjection)
	eventBus.Subscribe(events.NoteTagAdded{}, noteProjection)
	eventBus.Subscribe(events.NoteTagRemoved{}, noteProjection)

	// replay old events through the bus
	err = eventStore.ReplayFrom(0, eventBus.Publish)
	if err != nil {
		log.Fatal(fmt.Errorf("ReplayFrom: %w", err))
	}

	// connect the event bus to the store for live events
	eventStore.RegisterPublisher(eventBus)

	// live-only, async workers
	webhookWorker := webhook.NewWorker(commandBus)
	eventBus.Subscribe(events.WebhookReceived{}, webhookWorker)

	err = migrateOwnerlessNotes(noteProjection, commandBus)
	if err != nil {
		return nil, err
	}

	// River job queue
	dataDir := filepath.Dir(filename)

	workers := river.NewWorkers()
	river.AddWorker(workers, classify.NewJobWorker(commandBus))
	river.AddWorker(workers, enrich.NewJobWorker(commandBus))

	ytOAuthConfig := &oauth2.Config{
		ClientID:     cfg.GoogleClientID,
		ClientSecret: cfg.GoogleClientSecret,
		Endpoint:     google.Endpoint,
		Scopes:       []string{"https://www.googleapis.com/auth/youtube.readonly"},
	}
	river.AddWorker(workers, youtubesync.NewJobWorker(commandBus, ytOAuthConfig, dataDir))
	river.AddWorker(workers, bskysync.NewJobWorker(commandBus, dataDir))

	ytSyncJob := river.NewPeriodicJob(
		river.PeriodicInterval(6*time.Hour),
		func() (river.JobArgs, *river.InsertOpts) {
			return youtubesync.JobArgs{}, nil
		},
		&river.PeriodicJobOpts{ID: "youtube_sync"},
	)

	bskySyncJob := river.NewPeriodicJob(
		river.PeriodicInterval(6*time.Hour),
		func() (river.JobArgs, *river.InsertOpts) {
			return bskysync.JobArgs{}, nil
		},
		&river.PeriodicJobOpts{ID: "bsky_sync"},
	)

	riverClient, err := appriver.New(dataDir, workers, ytSyncJob, bskySyncJob)
	if err != nil {
		return nil, fmt.Errorf("river: %w", err)
	}

	// Enqueue classify jobs for new inbox notes
	eventBus.Subscribe(events.NoteCreated{}, &classifyEnqueuer{riverClient: riverClient})
	// Enqueue enrich jobs for enrichment requests
	eventBus.Subscribe(events.NoteEnrichmentRequested{}, &enrichEnqueuer{riverClient: riverClient})

	return &App{
		Commander:     commandBus,
		Notes:         noteProjection,
		River:         riverClient,
		DataDir:       dataDir,
		EventDebugger: eventStore,
	}, nil
}

type classifyEnqueuer struct {
	riverClient *appriver.Client
}

func (e *classifyEnqueuer) Handle(ev evoke.Event, replay bool) error {
	evt := ev.(events.NoteCreated)
	if evt.Category != "inbox" {
		return nil
	}
	return e.riverClient.Insert(context.Background(), classify.JobArgs{
		NoteID: evt.NoteID,
		Text:   evt.Text,
	}, nil)
}

type enrichEnqueuer struct {
	riverClient *appriver.Client
}

func (e *enrichEnqueuer) Handle(ev evoke.Event, replay bool) error {
	evt := ev.(events.NoteEnrichmentRequested)
	return e.riverClient.Insert(context.Background(), enrich.JobArgs{
		NoteID: evt.NoteID,
		Text:   evt.Text,
	}, nil)
}

func (a *App) youtubeTokenDir() string {
	return filepath.Join(a.DataDir, "youtube_tokens")
}

func (a *App) youtubeTokenPath(userID string) string {
	return filepath.Join(a.youtubeTokenDir(), userID+".json")
}

func (a *App) SaveYouTubeToken(userID string, token *oauth2.Token) error {
	if err := os.MkdirAll(a.youtubeTokenDir(), 0700); err != nil {
		return err
	}
	data, err := json.Marshal(token)
	if err != nil {
		return err
	}
	return os.WriteFile(a.youtubeTokenPath(userID), data, 0600)
}

func (a *App) LoadYouTubeToken(userID string) (*oauth2.Token, error) {
	data, err := os.ReadFile(a.youtubeTokenPath(userID))
	if err != nil {
		return nil, err
	}
	var token oauth2.Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, err
	}
	return &token, nil
}

func (a *App) HasYouTubeToken(userID string) bool {
	_, err := os.Stat(a.youtubeTokenPath(userID))
	return err == nil
}

type BlueskyCredentials struct {
	Handle      string `json:"handle"`
	AppPassword string `json:"app_password"`
}

func (a *App) bskyCredPath(userID string) string {
	return filepath.Join(a.DataDir, "bsky_tokens", userID+".json")
}

func (a *App) SaveBlueskyCredentials(userID string, creds BlueskyCredentials) error {
	dir := filepath.Join(a.DataDir, "bsky_tokens")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	data, err := json.Marshal(creds)
	if err != nil {
		return err
	}
	return os.WriteFile(a.bskyCredPath(userID), data, 0600)
}

func (a *App) LoadBlueskyCredentials(userID string) (*BlueskyCredentials, error) {
	data, err := os.ReadFile(a.bskyCredPath(userID))
	if err != nil {
		return nil, err
	}
	var creds BlueskyCredentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, err
	}
	return &creds, nil
}

func (a *App) HasBlueskyCredentials(userID string) bool {
	_, err := os.Stat(a.bskyCredPath(userID))
	return err == nil
}

// One time migration to add an owner to notes without one
func migrateOwnerlessNotes(p *note.Projection, cmd evoke.CommandSender) error {
	// find notes with no owner
	noteList, err := p.FindAll("")
	if err != nil {
		return err
	}
	for _, note := range noteList {
		fmt.Printf("%s %s %s\n", note.ID, note.Category, note.Text)
		err := cmd.Send(commands.SetNoteOwner{
			NoteID: note.ID,
			Owner:  "114909697912906591341",
		})
		if err != nil {
			return err
		}
	}
	return nil

}
