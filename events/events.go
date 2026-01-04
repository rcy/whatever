package events

import (
	"time"

	"github.com/google/uuid"
)

type RealmCreated struct {
	RealmID   uuid.UUID
	CreatedAt time.Time
	Name      string
}

type RealmDeleted struct {
	RealmID uuid.UUID
}

type NoteCreated struct {
	NoteID    uuid.UUID
	CreatedAt time.Time
	RealmID   uuid.UUID
	Text      string
}

type NoteEnrichmentRequested struct {
	NoteID      uuid.UUID
	RequestedAt time.Time
	Text        string
}

type NoteTextUpdated struct {
	NoteID uuid.UUID
	Text   string
}

type NoteDeleted struct {
	NoteID uuid.UUID
}

type NoteUndeleted struct {
	NoteID uuid.UUID
}

type NoteCategoryChanged struct {
	NoteID      uuid.UUID
	Category    string
	Subcategory string
}

type NoteSubcategoryChanged struct {
	NoteID      uuid.UUID
	Subcategory string
}

type NoteRealmChanged struct {
	RealmID string
}

type NoteTaskCompleted struct {
	NoteID uuid.UUID
}

type NoteTaskDeferred struct {
	NoteID uuid.UUID
}

type NoteTaskReopened struct {
	NoteID uuid.UUID
}

type NoteEnriched struct {
	NoteID uuid.UUID
	Title  string
}

type NoteEnrichmentFailed struct {
	NoteID uuid.UUID
}
