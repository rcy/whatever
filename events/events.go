package events

const NoteAggregate = "note"

// NoteCreated
type NoteCreated struct{ Text string }

func (NoteCreated) EventType() string { return "NoteCreated" }
func (NoteCreated) Aggregate() string { return NoteAggregate }

// NoteTextUpdated
type NoteTextUpdated struct{ Text string }

func (NoteTextUpdated) EventType() string { return "NoteTextUpdated" }
func (NoteTextUpdated) Aggregate() string { return NoteAggregate }

// NoteDeleted
type NoteDeleted struct{}

func (NoteDeleted) EventType() string { return "NoteDeleted" }
func (NoteDeleted) Aggregate() string { return NoteAggregate }

// NoteUndeleted
type NoteUndeleted struct{}

func (NoteUndeleted) EventType() string { return "NoteUndeleted" }
func (NoteUndeleted) Aggregate() string { return NoteAggregate }

// NoteCategoryChanged
type NoteCategoryChanged struct{ Category string }

func (NoteCategoryChanged) EventType() string { return "NoteCategoryChanged" }
func (NoteCategoryChanged) Aggregate() string { return NoteAggregate }
