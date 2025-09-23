package events

type realmEvent struct{}

func (realmEvent) Aggregate() string { return "realm" }

type RealmCreated struct {
	realmEvent
	Name string
}

type RealmDeleted struct{ realmEvent }

type noteEvent struct{}

func (noteEvent) Aggregate() string { return "note" }

type NoteCreated struct {
	noteEvent
	RealmID string
	Text    string
}

type NoteTextUpdated struct {
	noteEvent
	Text string
}

type NoteDeleted struct{ noteEvent }

type NoteUndeleted struct{ noteEvent }

type NoteCategoryChanged struct {
	noteEvent
	Category string
}
type NoteRealmChanged struct {
	noteEvent
	RealmID string
}
type NoteTaskCompleted struct{ noteEvent }
type NoteTaskDeferred struct{ noteEvent }
type NoteTaskReopened struct{ noteEvent }
