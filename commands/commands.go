package commands

import (
	"time"

	"github.com/google/uuid"
)

type CreateRealm struct {
	RealmID uuid.UUID
	Name    string
}

func (c CreateRealm) AggregateID() uuid.UUID { return c.RealmID }

type DeleteRealm struct {
	RealmID uuid.UUID
}

func (c DeleteRealm) AggregateID() uuid.UUID { return c.RealmID }

type CreateNote struct {
	Owner       string
	NoteID      uuid.UUID
	RealmID     uuid.UUID
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

type SetNoteRealm struct {
	NoteID  uuid.UUID
	RealmID string
}

func (c SetNoteRealm) AggregateID() uuid.UUID { return c.NoteID }

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
