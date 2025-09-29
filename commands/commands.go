package commands

import (
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
	NoteID  uuid.UUID
	RealmID uuid.UUID
	Text    string
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
