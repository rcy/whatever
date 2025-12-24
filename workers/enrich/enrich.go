package enrich

import (
	"fmt"
	"time"

	"github.com/alfarisi/urlmeta"
	"github.com/rcy/evoke"
	"github.com/rcy/whatever/commands"
	"github.com/rcy/whatever/events"
)

type worker struct {
	cmdSender evoke.CommandSender
}

func NewWorker(cmdSender evoke.CommandSender) *worker {
	return &worker{cmdSender: cmdSender}
}

func (w worker) Handle(e evoke.Event, replay bool) error {
	evt, ok := e.(events.NoteEnrichmentRequested)
	if !ok {
		return fmt.Errorf("not a NoteEnrichmentRequested event")
	}

	go func() {
		fmt.Println("enriching...", evt)

		meta, err := urlmeta.Extract(evt.Text)
		if err != nil {
			w.cmdSender.MustSend(commands.FailNoteEnrichment{
				NoteID:   evt.NoteID,
				FailedAt: time.Now(),
			})
			return
		}

		var thumb string
		if meta.OEmbed != nil {
			thumb = meta.OEmbed.ThumbnailURL
		}

		w.cmdSender.MustSend(commands.CompleteNoteEnrichment{
			NoteID:      evt.NoteID,
			CompletedAt: time.Now(),
			Title:       meta.Title,
			Thumb:       thumb,
		})

		fmt.Println("enriching...done", evt)
	}()

	return nil
}
