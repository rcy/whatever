package web

import (
	_ "embed"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rcy/whatever/app"
	"github.com/rcy/whatever/app/notes"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

type webservice struct {
	app *app.Service
}

func Server(app *app.Service) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	svc := webservice{app: app}
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/notes", http.StatusSeeOther)
	})
	r.Get("/notes", svc.notesHandler)
	r.Post("/notes", svc.postNotesHandler)
	r.Get("/events", svc.eventsHandler)
	return r
}

type Params struct {
	Main g.Node
}

func page(main g.Node) g.Node {
	return h.HTML(h.Lang("en"),
		h.Head(
			h.Meta(h.Name("viewport"), h.Content("width=device-width, initial-scale=1")),
			//h.Meta(h.Name("color-scheme"), h.Content("light dark")),
			h.Link(h.Rel("stylesheet"), h.Href("https://cdn.jsdelivr.net/npm/@picocss/pico@2/css/pico.jade.min.css")),
			h.Link(h.Rel("stylesheet"), h.Href("https://cdn.jsdelivr.net/npm/@picocss/pico@2/css/pico.colors.min.css")),
		),
		h.Body( //g.Attr("data-theme", "dark"),
			h.Div(h.Class("container"),
				h.H1(g.Text("whatever")),
				h.Main(main),
			)),
	)
}

func (s *webservice) notesHandler(w http.ResponseWriter, r *http.Request) {
	noteList, err := s.app.NS.FindAll()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	slices.Reverse(noteList)

	page(g.Group{
		h.Form(h.Action("/notes"), h.Method("post"),
			h.Input(h.AutoFocus(), h.Name("text")),
		),
		h.Table(h.Class("striped"), h.TBody(
			g.Map(noteList, func(note notes.Model) g.Node {
				return h.Tr(
					h.Td(h.Code(g.Text(note.ID[0:7]))),
					h.Td(g.Text(note.Ts.Local().Format(time.DateTime))),
					h.Td(g.Text(note.Text)),
				)
			}))),
	}).Render(w)
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
	http.Error(w, "not implemented", http.StatusInternalServerError)

	// var events []flog.Model
	// err := s.app.ES.DBTodo.Select(&events, `select * from events order by event_id desc`)
	// if err != nil {
	// 	http.Error(w, err.Error(), http.StatusInternalServerError)
	// 	return
	// }

	// page(g.Group{
	// 	h.Table(
	// 		h.Body(
	// 			g.Map(events, func(event flog.Model) g.Node {
	// 				return h.Tr(
	// 					h.Td(g.Text(fmt.Sprint(event.EventID))),
	// 					h.Td(h.A(h.Code(g.Text(event.AggregateID[0:7])))),
	// 					h.Td(g.Text(event.EventType)),
	// 					h.Td(h.Code(g.Text(string(event.EventData)))),
	// 				)
	// 			}),
	// 		),
	// 	),
	// },
	// ).Render(w)
}
