package webhook

import (
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"
	"github.com/rcy/evoke"
	"github.com/rcy/whatever/catalog/notesmeta"
	"github.com/rcy/whatever/commands"
	"github.com/rcy/whatever/events"
)

type Worker struct {
	cmdSender evoke.CommandSender
}

func NewWorker(cmdSender evoke.CommandSender) *Worker {
	return &Worker{cmdSender: cmdSender}
}

func (w *Worker) Handle(e evoke.Event, replay bool) error {
	evt, ok := e.(events.WebhookReceived)
	if !ok {
		return fmt.Errorf("not a WebhookReceived event")
	}

	switch evt.Path {
	case "pebble":
		return w.handlePebble(evt)
	default:
		log.Printf("webhook worker: unhandled path %q", evt.Path)
		return nil
	}
}

func (w *Worker) handlePebble(evt events.WebhookReceived) error {
	text := strings.TrimSpace(evt.Fields["transcription"])
	if text == "" {
		text = strings.TrimSpace(evt.Fields["text"])
	}
	if text == "" {
		log.Printf("webhook worker: no text in webhook %s", evt.WebhookID)
		return nil
	}

	noteID := uuid.New()
	err := w.cmdSender.Send(commands.CreateNote{
		Owner:       evt.Owner,
		NoteID:      noteID,
		Text:        text,
		Category:    notesmeta.Note.Slug,
		Subcategory: notesmeta.Note.Inbox().Slug,
	})
	if err != nil {
		return err
	}

	return w.cmdSender.Send(commands.AddNoteTag{
		NoteID: noteID,
		Tag:    "pebble",
	})
}
