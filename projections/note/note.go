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
	Owner       string    `db:"owner"`
	Ts          time.Time `db:"ts"`
	Text        string    `db:"text"`
	Category    string    `db:"category"`
	Subcategory string    `db:"subcategory"`
	Due         any       `db:"due"`
	State       string    `db:"state"`
	Status      string    `db:"status"`
}

type Person struct {
	Handle string `db:"handle"`
}

type Projection struct {
	db *sqlx.DB
}

func New() (*Projection, error) {
	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`create table notes(id not null unique, owner not null, ts timestamp not null, text not null, category not null, subcategory not null, due timestamp, state not null, status not null)`)
	if err != nil {
		return nil, fmt.Errorf("create table notes: %w", err)
	}
	_, err = db.Exec(`create table deleted_notes(id not null unique, owner not null, ts timestamp not null, text not null, category not null, subcategory not null, due timestamp, state not null, status not null)`)
	if err != nil {
		return nil, fmt.Errorf("create table deleted_notes: %w", err)
	}

	_, err = db.Exec(`create table note_people(handle string, note_id string)`)
	if err != nil {
		return nil, fmt.Errorf("create table note_people: %w", err)
	}

	return &Projection{db: db}, nil
}

func (p *Projection) Handle(evt evoke.Event, replaying bool) error {
	switch e := evt.(type) {
	case events.NoteCreated:
		q := `insert into notes(id, owner, ts, text, category, subcategory, due, state, status) values(?,?,?,?,?,?,?,?,?)`
		_, err := p.db.Exec(q, e.NoteID, e.Owner, e.CreatedAt, e.Text, e.Category, e.Subcategory, "doo", "open", "")
		if err != nil {
			return err
		}

		for _, mention := range extractMentions(e.Text) {
			_, err = p.db.Exec(`insert into note_people(note_id, handle) values(?,?)`, e.NoteID, mention)
			if err != nil {
				return err
			}
		}
	case events.NoteOwnerSet:
		_, err := p.db.Exec(`update notes set owner = ? where id = ?`, e.Owner, e.NoteID)
		if err != nil {
			return err
		}
	case events.NoteDeleted:
		q := `insert into deleted_notes(id, owner, ts, text, category, subcategory, due, state, status) select id, owner, ts, text, category, subcategory, due, state, status from notes where id = ?`
		_, err := p.db.Exec(q, e.NoteID)
		if err != nil {
			return err
		}

		_, err = p.db.Exec(`delete from notes where id = ?`, e.NoteID)
		if err != nil {
			return err
		}

		return err
	case events.NoteUndeleted:
		q := `insert into notes(id, owner, ts, text, category, subcategory, due, state, status) select id, owner, ts, text, category, subcategory, due, state, status from deleted_notes where id = ?`
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
		_, err := p.db.Exec(`update notes set category = ?, subcategory = ? where id = ?`, e.Category, e.Subcategory, e.NoteID)
		return err
	case events.NoteSubcategoryChanged:
		_, err := p.db.Exec(`update notes set subcategory = ? where id = ?`, e.Subcategory, e.NoteID)
		return err
	case events.NoteDueChanged:
		_, err := p.db.Exec(`update notes set due = ? where id = ?`, e.Due, e.NoteID)
		return err
	case events.NoteDueCleared:
		_, err := p.db.Exec(`update notes set due = null where id = ?`, e.NoteID)
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

func (p *Projection) FindAllPeople(owner string) ([]string, error) {
	var handles []string
	err := p.db.Select(&handles, `select distinct handle from note_people join notes on note_people.note_id = notes.id where owner = ?`, owner)
	if err != nil {
		return nil, fmt.Errorf("Select notes 1: %w", err)
	}
	return handles, nil
}

func (p *Projection) FindAll(owner string) ([]Note, error) {
	var noteList []Note
	err := p.db.Select(&noteList, `select * from notes where owner = ? order by ts asc`, owner)
	if err != nil {
		return nil, fmt.Errorf("Select notes 2: %w", err)
	}
	return noteList, nil
}

func (p *Projection) FindAllByPerson(owner string, handle string) ([]Note, error) {
	var noteList []Note
	err := p.db.Select(&noteList, `select notes.* from notes join note_people on note_people.note_id = notes.id where owner = ? and handle = ? order by ts asc`, owner, handle)
	if err != nil {
		return nil, fmt.Errorf("Select notes 3: %w", err)
	}
	return noteList, nil
}

func (p *Projection) FindAllWithMention(owner string) ([]Note, error) {
	var noteList []Note
	err := p.db.Select(&noteList, `select distinct notes.* from notes join note_people on note_people.note_id = notes.id where owner = ? order by ts asc`, owner)
	if err != nil {
		return nil, fmt.Errorf("Select notes 4: %w", err)
	}
	return noteList, nil
}

func (p *Projection) FindAllByCategory(owner string, category string) ([]Note, error) {
	var noteList []Note
	err := p.db.Select(&noteList, `select * from notes where owner = ? and category = ? order by ts asc`, owner, category)
	if err != nil {
		return nil, fmt.Errorf("Select notes 5: %w", err)
	}
	return noteList, nil
}

func (p *Projection) FindAllByCategoryAndSubcategory(owner string, category string, subcategory string) ([]Note, error) {
	var noteList []Note
	err := p.db.Select(&noteList, `select * from notes where owner = ? and category = ? and subcategory = ? order by ts asc`, owner, category, subcategory)
	if err != nil {
		return nil, fmt.Errorf("Select notes 6: %w", err)
	}
	return noteList, nil
}

func (p *Projection) FindAllDeleted(owner string) ([]Note, error) {
	var noteList []Note
	err := p.db.Select(&noteList, `select * from deleted_notes where owner = ? order by ts asc`, owner)
	if err != nil {
		return nil, fmt.Errorf("Select deleted notes: %w", err)
	}
	return noteList, nil
}

type CategoryCount struct {
	Category string
	Count    int `db:"count"`
}

func (p *Projection) CategoryCounts(owner string) ([]CategoryCount, error) {
	var categories []CategoryCount
	err := p.db.Select(&categories, `select count(*) count, category from notes where owner = ? group by category`, owner)
	if err != nil {
		return nil, fmt.Errorf("select categories: %w", err)
	}
	return categories, nil
}

type SubcategoryCount struct {
	Subcategory string
	Count       int `db:"count"`
}

func (p *Projection) SubcategoryCounts(owner string, category string) ([]SubcategoryCount, error) {
	var categories []SubcategoryCount
	err := p.db.Select(&categories, `select count(*) count, subcategory from notes where owner = ? and category = ? group by subcategory`, owner, category)
	if err != nil {
		return nil, fmt.Errorf("select subcategories: %w", err)
	}
	return categories, nil
}
