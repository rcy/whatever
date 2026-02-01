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
	"github.com/rcy/whatever/workers/enrich"
)

type App struct {
	Commander     evoke.CommandSender
	Notes         *note.Projection
	EventDebugger interface {
		DebugEvents() ([]evoke.RecordedEvent, error)
	}
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
	evoke.RegisterEvent(eventStore, &events.NoteOwnerSet{})
	evoke.RegisterEvent(eventStore, &events.NoteEnrichmentRequested{})
	evoke.RegisterEvent(eventStore, &events.RealmCreated{}) // deprecated
	evoke.RegisterEvent(eventStore, &events.NoteDeleted{})
	evoke.RegisterEvent(eventStore, &events.NoteUndeleted{})
	evoke.RegisterEvent(eventStore, &events.NoteTextUpdated{})
	evoke.RegisterEvent(eventStore, &events.NoteCategoryChanged{})
	evoke.RegisterEvent(eventStore, &events.NoteSubcategoryChanged{})
	evoke.RegisterEvent(eventStore, &events.NoteEnriched{})
	evoke.RegisterEvent(eventStore, &events.NoteEnrichmentFailed{})

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
	commandBus.RegisterHandler(commands.SetNoteSubcategory{}, noteHandler)
	commandBus.RegisterHandler(commands.CompleteNoteEnrichment{}, noteHandler)
	commandBus.RegisterHandler(commands.FailNoteEnrichment{}, noteHandler)

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
	eventBus.Subscribe(events.NoteEnrichmentRequested{}, noteProjection)
	eventBus.Subscribe(events.NoteEnriched{}, noteProjection)
	eventBus.Subscribe(events.NoteEnrichmentFailed{}, noteProjection)

	// replay old events through the bus
	err = eventStore.ReplayFrom(0, eventBus.Publish)
	if err != nil {
		log.Fatal(fmt.Errorf("ReplayFrom: %w", err))
	}

	// connect the event bus to the store for live events
	eventStore.RegisterPublisher(eventBus)

	// live-only, async workers
	enrichWorker := enrich.NewWorker(commandBus)
	eventBus.Subscribe(events.NoteEnrichmentRequested{}, enrichWorker)

	err = migrateOwnerlessNotes(noteProjection, commandBus)
	if err != nil {
		return nil, err
	}

	return &App{
		Commander:     commandBus,
		Notes:         noteProjection,
		EventDebugger: eventStore,
	}, nil
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
