package events

const RealmAggregate = "realm"

// RealmCreated
type RealmCreated struct {
	Name string
}

func (RealmCreated) EventType() string { return "RealmCreated" }
func (RealmCreated) Aggregate() string { return RealmAggregate }

type RealmDeleted struct{}

func (RealmDeleted) EventType() string { return "RealmDeleted" }
func (RealmDeleted) Aggregate() string { return RealmAggregate }

const NoteAggregate = "note"

// NoteCreated
type NoteCreated struct {
	RealmID string
	Text    string
}

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

// NoteRealmChanged
type NoteRealmChanged struct{ RealmID string }

func (NoteRealmChanged) EventType() string { return "NoteRealmChanged" }
func (NoteRealmChanged) Aggregate() string { return NoteAggregate }
