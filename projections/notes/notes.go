package notes

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/rcy/evoke"
	"github.com/rcy/whatever/events"
	_ "modernc.org/sqlite"
)

type Note struct {
	ID       string    `db:"id"`
	Ts       time.Time `db:"ts"`
	Text     string    `db:"text"`
	Category string    `db:"category"`
	RealmID  string    `db:"realm_id"`
}

type Projection struct {
	db *sqlx.DB
}

func (p *Projection) FindOne(id string) (Note, error) {
	var note Note
	err := p.db.Get(&note, `select * from notes where id = ?`, id)
	if err != nil {
		return Note{}, err
	}
	return note, nil
}

func (p *Projection) FindAll() ([]Note, error) {
	var noteList []Note
	err := p.db.Select(&noteList, `select * from notes order by ts asc`)
	if err != nil {
		return nil, fmt.Errorf("Select notes: %w", err)
	}
	return noteList, nil
}

func (p *Projection) FindAllInRealm(realmID string) ([]Note, error) {
	var noteList []Note
	err := p.db.Select(&noteList, `select * from notes where realm_id = ? order by ts asc`, realmID)
	if err != nil {
		return nil, fmt.Errorf("Select notes in realm: %w", err)
	}
	return noteList, nil
}

func (p *Projection) FindAllInRealmByCategory(realm string, category string) ([]Note, error) {
	var noteList []Note
	err := p.db.Select(&noteList, `select * from notes where realm_id = ? and category = ? order by ts asc`, realm, category)
	if err != nil {
		return nil, fmt.Errorf("Select notes: %w", err)
	}
	return noteList, nil
}

func (p *Projection) FindAllDeleted() ([]Note, error) {
	var noteList []Note
	err := p.db.Select(&noteList, `select * from deleted_notes order by ts asc`)
	if err != nil {
		return nil, fmt.Errorf("Select deleted notes: %w", err)
	}
	return noteList, nil
}

type CategoryCount struct {
	Category string
	Count    int `db:"count"`
}

func (p *Projection) CategoryCounts(realmID string) ([]CategoryCount, error) {
	var categories []CategoryCount
	err := p.db.Select(&categories, `select count(*) count, category from notes where realm_id = ? group by category`, realmID)
	if err != nil {
		return nil, fmt.Errorf("select categories: %w", err)
	}
	return categories, nil
}

func New(e *evoke.Service) (*Projection, error) {
	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`create table notes(id not null, ts timestamp not null, text not null, category not null, realm_id not null, state not null, status not null)`)
	if err != nil {
		return nil, fmt.Errorf("create table notes: %w", err)
	}
	_, err = db.Exec(`create table deleted_notes(id not null, ts timestamp not null, text not null, category not null, realm_id not null, state not null, status not null)`)
	if err != nil {
		return nil, fmt.Errorf("create table deleted_notes: %w", err)
	}

	return &Projection{db: db}, nil
}

func (p *Projection) Subscribe(e *evoke.Service) {
	e.SubscribeSync(events.NoteCreated{}, p.noteCreated)
	e.SubscribeSync(events.NoteDeleted{}, p.noteDeleted)
	e.SubscribeSync(events.NoteUndeleted{}, p.noteUndeleted)
	e.SubscribeSync(events.NoteTextUpdated{}, p.noteTextUpdated)
	e.SubscribeSync(events.NoteCategoryChanged{}, p.noteCategoryChanged)
	e.SubscribeSync(events.NoteRealmChanged{}, p.noteRealmChanged)
	e.SubscribeSync(events.NoteTaskCompleted{}, p.noteTaskCompleted)
}

func (p *Projection) noteCreated(event evoke.Event, _ bool) error {
	payload, err := evoke.UnmarshalPayload[events.NoteCreated](event)
	if err != nil {
		return err
	}

	q := `insert into notes(id, ts, text, realm_id, category, state, status) values(?,?,?,?,?,?,?)`
	_, err = p.db.Exec(q, event.AggregateID, event.CreatedAt, payload.Text, payload.RealmID, "inbox", "open", "active")
	if err != nil {
		return err
	}

	return nil
}

func (p *Projection) noteDeleted(event evoke.Event, _ bool) error {
	q := `insert into deleted_notes(id, ts, text, realm_id, category, state, status) select id, ts, text, realm_id, category, state, status from notes where id = ?`
	_, err := p.db.Exec(q, event.AggregateID)
	if err != nil {
		return err
	}

	_, err = p.db.Exec(`delete from notes where id = ?`, event.AggregateID)
	return err
}

func (p *Projection) noteUndeleted(event evoke.Event, _ bool) error {
	q := `insert into notes(id, ts, text, realm_id, category) select id, ts, text, realm_id, category from deleted_notes where id = ?`
	_, err := p.db.Exec(q, event.AggregateID)
	if err != nil {
		return err
	}

	_, err = p.db.Exec(`delete from deleted_notes where id = ?`, event.AggregateID)
	return err
}

func (p *Projection) noteTextUpdated(event evoke.Event, _ bool) error {
	payload, err := evoke.UnmarshalPayload[events.NoteTextUpdated](event)
	if err != nil {
		return err
	}

	q := `update notes set text = ? where id = ?`
	_, err = p.db.Exec(q, payload.Text, event.AggregateID)
	return nil
}

func (p *Projection) noteCategoryChanged(event evoke.Event, _ bool) error {
	payload, err := evoke.UnmarshalPayload[events.NoteCategoryChanged](event)
	if err != nil {
		return err
	}

	q := `update notes set category = ? where id = ?`
	_, err = p.db.Exec(q, payload.Category, event.AggregateID)
	return err
}

func (p *Projection) noteRealmChanged(event evoke.Event, _ bool) error {
	payload, err := evoke.UnmarshalPayload[events.NoteRealmChanged](event)
	if err != nil {
		return err
	}

	q := `update notes set realm_id = ? where id = ?`
	_, err = p.db.Exec(q, payload.RealmID, event.AggregateID)
	return err
}

func (p *Projection) noteTaskCompleted(event evoke.Event, _ bool) error {
	_, err := p.db.Exec(`update notes set state = 'closed', status = 'completed'`)
	return err
}

func (p *Projection) noteTaskReopened(event evoke.Event, _ bool) error {
	_, err := p.db.Exec(`update notes set state = 'open', status = 'open'`)
	return err
}

func (p *Projection) noteTaskDeferred(event evoke.Event, _ bool) error {
	_, err := p.db.Exec(`update notes set state = 'closed', status = 'deferred'`)
	return err
}
