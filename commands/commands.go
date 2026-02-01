package commands

import (
	"time"

	"github.com/google/uuid"
)

type CreateNote struct {
	Owner       string
	NoteID      uuid.UUID
	Text        string
	Category    string
	Subcategory string
}

func (c CreateNote) AggregateID() uuid.UUID { return c.NoteID }

type DeleteNote struct {
	NoteID uuid.UUID
}

func (c DeleteNote) AggregateID() uuid.UUID { return c.NoteID }

type UndeleteNote struct {
	NoteID uuid.UUID
}

func (c UndeleteNote) AggregateID() uuid.UUID { return c.NoteID }

type UpdateNoteText struct {
	NoteID uuid.UUID
	Text   string
}

func (c UpdateNoteText) AggregateID() uuid.UUID { return c.NoteID }

type SetNoteOwner struct {
	NoteID uuid.UUID
	Owner  string
}

func (c SetNoteOwner) AggregateID() uuid.UUID { return c.NoteID }

type SetNoteCategory struct {
	NoteID   uuid.UUID
	Category string
}

func (c SetNoteCategory) AggregateID() uuid.UUID { return c.NoteID }

type SetNoteSubcategory struct {
	NoteID      uuid.UUID
	Subcategory string
}

func (c SetNoteSubcategory) AggregateID() uuid.UUID { return c.NoteID }

type CompleteNoteEnrichment struct {
	NoteID      uuid.UUID
	CompletedAt time.Time
	Title       string
	Thumb       string
}

func (c CompleteNoteEnrichment) AggregateID() uuid.UUID { return c.NoteID }

type FailNoteEnrichment struct {
	NoteID   uuid.UUID
	FailedAt time.Time
}

func (c FailNoteEnrichment) AggregateID() uuid.UUID { return c.NoteID }
