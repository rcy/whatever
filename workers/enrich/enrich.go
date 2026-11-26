package enrich

import (
	"context"
	"time"

	"github.com/alfarisi/urlmeta"
	"github.com/google/uuid"
	"github.com/rcy/evoke"
	"github.com/rcy/whatever/commands"
	"github.com/riverqueue/river"
)

type JobArgs struct {
	NoteID uuid.UUID `json:"note_id"`
	Text   string    `json:"text"`
}

func (JobArgs) Kind() string { return "enrich" }

type JobWorker struct {
	river.WorkerDefaults[JobArgs]
	cmdSender evoke.CommandSender
}

func NewJobWorker(cmdSender evoke.CommandSender) *JobWorker {
	return &JobWorker{cmdSender: cmdSender}
}

func (w *JobWorker) Work(ctx context.Context, job *river.Job[JobArgs]) error {
	meta, err := urlmeta.Extract(job.Args.Text)
	if err != nil {
		return w.cmdSender.Send(commands.FailNoteEnrichment{
			NoteID:   job.Args.NoteID,
			FailedAt: time.Now(),
		})
	}

	var thumb string
	if meta.OEmbed != nil {
		thumb = meta.OEmbed.ThumbnailURL
	}

	return w.cmdSender.Send(commands.CompleteNoteEnrichment{
		NoteID:      job.Args.NoteID,
		CompletedAt: time.Now(),
		Title:       meta.Title,
		Thumb:       thumb,
	})
}
