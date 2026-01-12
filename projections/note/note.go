package note

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/rcy/evoke"
	"github.com/rcy/whatever/events"
	_ "modernc.org/sqlite"
)

type Note struct {
	ID          uuid.UUID `db:"id"`
	Ts          time.Time `db:"ts"`
	Text        string    `db:"text"`
	Category    string    `db:"category"`
	Subcategory string    `db:"subcategory"`
	RealmID     uuid.UUID `db:"realm_id"`
	State       string    `db:"state"`
	Status      string    `db:"status"`
}

type Projection struct {
	db *sqlx.DB
}

func New() (*Projection, error) {
	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`create table notes(id not null unique, ts timestamp not null, text not null, category not null, subcategory not null, realm_id not null, state not null, status not null)`)
	if err != nil {
		return nil, fmt.Errorf("create table notes: %w", err)
	}
	_, err = db.Exec(`create table deleted_notes(id not null unique, ts timestamp not null, text not null, category not null, subcategory not null, realm_id not null, state not null, status not null)`)
	if err != nil {
		return nil, fmt.Errorf("create table deleted_notes: %w", err)
	}

	return &Projection{db: db}, nil
}

func (p *Projection) Handle(evt evoke.Event, replaying bool) error {
	fmt.Println(evt)

	switch e := evt.(type) {
	case events.NoteCreated:
		q := `insert into notes(id, ts, text, realm_id, category, subcategory, state, status) values(?,?,?,?,?,?,?,?)`
		_, err := p.db.Exec(q, e.NoteID, e.CreatedAt, e.Text, e.RealmID, "inbox", "", "open", "")
		if err != nil {
			return err
		}
	case events.NoteDeleted:
		q := `insert into deleted_notes(id, ts, text, realm_id, category, subcategory, state, status) select id, ts, text, realm_id, category, subcategory, state, status from notes where id = ?`
		_, err := p.db.Exec(q, e.NoteID)
		if err != nil {
			return err
		}

		_, err = p.db.Exec(`delete from notes where id = ?`, e.NoteID)
		return err
	case events.NoteUndeleted:
		q := `insert into notes(id, ts, text, realm_id, category, subcategory, state, status) select id, ts, text, realm_id, category, subcategory, state, status from deleted_notes where id = ?`
		_, err := p.db.Exec(q, e.NoteID)
		if err != nil {
			return err
		}

		_, err = p.db.Exec(`delete from deleted_notes where id = ?`, e.NoteID)
		return err
	case events.NoteTextUpdated:
		_, err := p.db.Exec(`update notes set text = ? where id = ?`, e.Text, e.NoteID)
		return err
	case events.NoteCategoryChanged:
		_, err := p.db.Exec(`update notes set category = ? where id = ?`, e.Category, e.NoteID)
		return err
	case events.NoteSubcategoryChanged:
		_, err := p.db.Exec(`update notes set subcategory = ? where id = ?`, e.Subcategory, e.NoteID)
		return err
	case events.NoteEnrichmentRequested:
		_, err := p.db.Exec(`update notes set status = 'enriching' where id = ?`, e.NoteID)
		return err
	case events.NoteEnriched:
		_, err := p.db.Exec(`update notes set status = '', text = ? || ' ' || text where id = ?`, e.Title, e.NoteID)
		return err
	case events.NoteEnrichmentFailed:
		_, err := p.db.Exec(`update notes set status = 'failure' where id = ?`, e.NoteID)
		return err
	default:
		return fmt.Errorf("note projection event not handled: %T", evt)
	}
	return nil
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

func (p *Projection) FindAllInRealmByCategory(realm uuid.UUID, category string) ([]Note, error) {
	var noteList []Note
	err := p.db.Select(&noteList, `select * from notes where realm_id = ? and category = ? order by ts asc`, realm, category)
	if err != nil {
		return nil, fmt.Errorf("Select notes: %w", err)
	}
	return noteList, nil
}

func (p *Projection) FindAllInRealmByCategoryAndSubcategory(realm uuid.UUID, category string, subcategory string) ([]Note, error) {
	var noteList []Note
	err := p.db.Select(&noteList, `select * from notes where realm_id = ? and category = ? and subcategory = ? order by ts asc`, realm, category, subcategory)
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

func (p *Projection) CategoryCounts(realmID uuid.UUID) ([]CategoryCount, error) {
	var categories []CategoryCount
	err := p.db.Select(&categories, `select count(*) count, category from notes where realm_id = ? group by category`, realmID)
	if err != nil {
		return nil, fmt.Errorf("select categories: %w", err)
	}
	return categories, nil
}

type SubcategoryCount struct {
	Subcategory string
	Count       int `db:"count"`
}

func (p *Projection) SubcategoryCounts(realmID uuid.UUID, category string) ([]SubcategoryCount, error) {
	var categories []SubcategoryCount
	err := p.db.Select(&categories, `select count(*) count, subcategory from notes where realm_id = ? and category = ? group by subcategory`, realmID, category)
	if err != nil {
		return nil, fmt.Errorf("select subcategories: %w", err)
	}
	return categories, nil
}
