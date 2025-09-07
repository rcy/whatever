package payloads

const NoteDeleted = "NoteDeleted"

const NoteUndeleted = "NoteUndeleted"

const NoteCreated = "NoteCreated"

type NoteCreatedPayload struct {
	Text string
}

const NoteTextUpdated = "NoteTextUpdated"

type NoteTextUpdatedPayload struct {
	Text string
}

const NoteCategoryChanged = "NoteCategoryChanged"

type NoteCategoryChangedPayload struct {
	Category string
}
