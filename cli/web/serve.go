package web

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rcy/whatever/app"
	"github.com/rcy/whatever/events"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

type webservice struct {
	app *app.Service
}

type ServeCmd struct {
	Port string `default:"9999"`
}

func (c *ServeCmd) Run(app *app.Service) error {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	svc := webservice{app: app}
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/notes", http.StatusSeeOther)
	})
	r.Get("/notes", svc.notesHandler)
	r.Post("/notes", svc.postNotesHandler)
	r.Get("/events", svc.eventsHandler)
	fmt.Printf("listening on http://localhost:%s\n", c.Port)
	return http.ListenAndServe(":"+c.Port, r)
}

func (s *webservice) notesHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: this is duplicated in cli/notes.go
	type note struct {
		ID   string    `db:"id"`
		Text string    `db:"text"`
		Ts   time.Time `db:"ts"`
	}
	var notes []note
	err := s.app.ES.DBTodo.Select(&notes, `select * from notes order by ts desc`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.HTML(h.Body(
		h.H1(g.Text("whatever")),
		h.Form(h.Action("/notes"), h.Method("post"),
			h.Input(h.AutoFocus(), h.Name("text")),
		),
		h.Table(h.TBody(
			g.Map(notes, func(note note) g.Node {
				return h.Tr(
					h.Td(h.Code(g.Text(note.ID[0:7]))),
					h.Td(g.Text(note.Ts.Local().Format(time.DateTime))),
					h.Td(g.Text(note.Text)),
				)
			}))),
	)).Render(w)
}

func (s *webservice) postNotesHandler(w http.ResponseWriter, r *http.Request) {
	text := strings.TrimSpace(r.FormValue("text"))
	if text != "" {
		_, err := s.app.CS.CreateNote(text)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	http.Redirect(w, r, "/notes", http.StatusSeeOther)
}

func (s *webservice) eventsHandler(w http.ResponseWriter, r *http.Request) {
	type model events.Model
	var events []model
	err := s.app.ES.DBTodo.Select(&events, `select * from events order by event_id desc`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.HTML(
		h.Body(
			h.H1(g.Text("whatever events")),
			h.Table(
				h.Body(
					g.Map(events, func(event model) g.Node {
						return h.Tr(
							h.Td(g.Text(fmt.Sprint(event.EventID))),
							h.Td(h.A(h.Code(g.Text(event.AggregateID[0:7])))),
							h.Td(g.Text(event.EventType)),
							h.Td(h.Code(g.Text(string(event.EventData)))),
						)
					}),
				),
			),
		),
	).Render(w)
}
