package events

import "github.com/rcy/whatever/evoke"

var (
	NoteCreated         = evoke.RegisterEvent("NoteCreated", "note", NoteCreatedPayload{})
	NoteDeleted         = evoke.RegisterEvent("NoteDeleted", "note", nil)
	NoteUndeleted       = evoke.RegisterEvent("NoteUndeleted", "note", nil)
	NoteTextUpdated     = evoke.RegisterEvent("NoteTextUpdated", "note", NoteTextUpdatedPayload{})
	NoteCategoryChanged = evoke.RegisterEvent("NoteCategoryChanged", "note", NoteCategoryChangedPayload{})
)

type NoteTextUpdatedPayload struct {
	Text string
}

type NoteCreatedPayload struct {
	Text string
}

type NoteCategoryChangedPayload struct {
	Category string
}
