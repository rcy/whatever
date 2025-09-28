package realm

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/rcy/evoke"
	"github.com/rcy/whatever/events"
	_ "modernc.org/sqlite"
)

type Realm struct {
	ID   uuid.UUID `db:"id"`
	Ts   string    `db:"ts"`
	Name string    `db:"name"`
}

type Projection struct {
	db *sqlx.DB
}

func New() (*Projection, error) {
	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`create table realms(id not null unique, ts timestamp not null, name not null unique)`)
	if err != nil {
		return nil, err
	}

	return &Projection{db: db}, nil
}

func (p *Projection) FindAll() ([]Realm, error) {
	var realmList []Realm
	err := p.db.Select(&realmList, `select * from realms order by ts asc`)
	if err != nil {
		return nil, fmt.Errorf("Find all realms: %w", err)
	}
	return realmList, nil
}

func (p *Projection) FindOldest() (*Realm, error) {
	var realm Realm

	err := p.db.Get(&realm, `select * from realms order by ts asc limit 1`)
	if err != nil {
		return nil, fmt.Errorf("Find oldest realm: %w", err)
	}
	return &realm, nil
}

func (p *Projection) FindByID(realmID uuid.UUID) (*Realm, error) {
	var realm Realm

	err := p.db.Get(&realm, `select * from realms where id = ?`, realmID)
	if err != nil {
		return nil, fmt.Errorf("Find realm by id: %w", err)
	}
	return &realm, nil
}

func (p *Projection) FindByName(name string) (*Realm, error) {
	var realm Realm

	err := p.db.Get(&realm, `select * from realms where name = ?`, name)
	if err != nil {
		return nil, fmt.Errorf("Find realm by id: %w", err)
	}
	return &realm, nil
}

// func (p *Projection) Subscribe(e *evoke.Service) {
// 	e.SubscribeSync(events.RealmCreated{}, p.realmCreated)
// 	e.SubscribeSync(events.RealmDeleted{}, p.realmDeleted)
// }

func (p *Projection) Handle(evt evoke.Event, replaying bool) error {
	switch e := evt.(type) {
	case events.RealmCreated:
		q := `insert into realms(id, ts, name) values(?,?,?)`
		_, err := p.db.Exec(q, e.RealmID, e.CreatedAt, e.Name)
		if err != nil {
			return err
		}

	case events.RealmDeleted:
		q := `delete from realms where id = ?`
		_, err := p.db.Exec(q, e.RealmID)
		if err != nil {
			return err
		}
	}
	return nil
}

// func (p *Projection) realmCreated(event evoke.Event, _ bool) error {
// 	payload, err := evoke.UnmarshalPayload[events.RealmCreated](event)
// 	if err != nil {
// 		return err
// 	}

// 	q := `insert into realms(id, ts, name) values(?,?,?)`
// 	_, err = p.db.Exec(q, event.AggregateID, event.CreatedAt, payload.Name)
// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }

// func (p *Projection) realmDeleted(event evoke.Event, _ bool) error {
// 	q := `delete from realms where id = ?`
// 	_, err := p.db.Exec(q, event.AggregateID)
// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }
