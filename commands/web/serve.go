package web

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rcy/whatever/events"
	"github.com/rcy/whatever/models"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

type ServeCmd struct {
	Port string `default:"9999"`
}

func (c *ServeCmd) Run(es *events.Service) error {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	svc := webservice{ES: *es}
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/notes", http.StatusSeeOther)
	})
	r.Get("/notes", svc.notesHandler)
	r.Post("/notes", svc.postNotesHandler)
	r.Get("/events", svc.eventsHandler)
	fmt.Printf("listening on http://localhost:%s\n", c.Port)
	return http.ListenAndServe(":"+c.Port, r)
}

type webservice struct {
	ES events.Service
}

func (s *webservice) notesHandler(w http.ResponseWriter, r *http.Request) {
	h.HTML(
		h.Body(
			h.H1(g.Text("whatever")),
			h.Form(h.Action("/notes"), h.Method("post"),
				h.Input(h.AutoFocus()),
			),
		),
	).Render(w)
}

func (s *webservice) postNotesHandler(w http.ResponseWriter, r *http.Request) {

	http.Redirect(w, r, "/notes", http.StatusSeeOther)
}

func (s *webservice) eventsHandler(w http.ResponseWriter, r *http.Request) {
	var events []models.Event
	err := s.ES.DBTodo.Select(&events, `select * from events order by event_id desc`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.HTML(
		h.Body(
			h.H1(g.Text("whatever events")),
			h.Table(
				h.Body(
					g.Map(events, func(event models.Event) g.Node {
						return h.Tr(
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
