package events

import (
	"time"

	"github.com/google/uuid"
)

type RealmCreated struct { // deprecated
	RealmID   uuid.UUID
	CreatedAt time.Time
	Name      string
}

type NoteCreated struct {
	NoteID      uuid.UUID
	Owner       string
	CreatedAt   time.Time
	Text        string
	Category    string
	Subcategory string
}

type NoteOwnerSet struct {
	NoteID uuid.UUID
	Owner  string
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

type NoteDueChanged struct {
	NoteID uuid.UUID
	Due    time.Time
}

type NoteDueCleared struct {
	NoteID uuid.UUID
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
