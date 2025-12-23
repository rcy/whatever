package app

import (
	"errors"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/rcy/evoke"
	"github.com/rcy/whatever/aggregates"
	"github.com/rcy/whatever/commands"
	"github.com/rcy/whatever/events"
	"github.com/rcy/whatever/projections/note"
	"github.com/rcy/whatever/projections/realm"
)

type App struct {
	Commander evoke.CommandSender
	Notes     *note.Projection
	Realms    *realm.Projection
}

func New(filename string) (*App, error) {
	if filename == "" {
		return nil, errors.New("filename is empty")
	}

	eventStore, err := evoke.NewFileStore(filename)
	if err != nil {
		log.Fatal(err)
	}
	evoke.RegisterEvent(eventStore, &events.NoteCreated{})
	evoke.RegisterEvent(eventStore, &events.RealmCreated{})
	evoke.RegisterEvent(eventStore, &events.NoteDeleted{})
	evoke.RegisterEvent(eventStore, &events.NoteUndeleted{})
	evoke.RegisterEvent(eventStore, &events.NoteTextUpdated{})
	evoke.RegisterEvent(eventStore, &events.NoteCategoryChanged{})

	//
	// COMMANDS
	//
	commandBus := evoke.NewCommandBus()

	noteFactory := func(id uuid.UUID) evoke.Aggregate { return aggregates.NewNoteAggregate(id) }
	noteHandler := evoke.NewAggregateHandler(eventStore, noteFactory)
	commandBus.RegisterHandler(commands.CreateNote{}, noteHandler)
	commandBus.RegisterHandler(commands.DeleteNote{}, noteHandler)
	commandBus.RegisterHandler(commands.UndeleteNote{}, noteHandler)
	commandBus.RegisterHandler(commands.UpdateNoteText{}, noteHandler)
	commandBus.RegisterHandler(commands.SetNoteCategory{}, noteHandler)

	realmFactory := func(id uuid.UUID) evoke.Aggregate { return aggregates.NewRealmAggregate(id) }
	realmHandler := evoke.NewAggregateHandler(eventStore, realmFactory)
	commandBus.RegisterHandler(commands.CreateRealm{}, realmHandler)

	//
	// PROJECTIONS
	//
	eventBus := evoke.NewEventBus()

	noteProjection, err := note.New()
	if err != nil {
		log.Fatal(err)
	}
	eventBus.Subscribe(events.NoteCreated{}, noteProjection)
	eventBus.Subscribe(events.NoteDeleted{}, noteProjection)
	eventBus.Subscribe(events.NoteUndeleted{}, noteProjection)
	eventBus.Subscribe(events.NoteTextUpdated{}, noteProjection)
	eventBus.Subscribe(events.NoteCategoryChanged{}, noteProjection)

	realmProjection, err := realm.New()
	if err != nil {
		log.Fatal(err)
	}
	eventBus.Subscribe(events.RealmCreated{}, realmProjection)

	// replay old events through the bus
	err = eventStore.ReplayFrom(0, eventBus.Publish)
	if err != nil {
		log.Fatal(fmt.Errorf("ReplayFrom: %w", err))
	}

	// connect the event bus to the store for live events
	eventStore.RegisterPublisher(eventBus)

	return &App{
		Commander: commandBus,
		Notes:     noteProjection,
		Realms:    realmProjection,
	}, nil
}
