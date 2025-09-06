package web

import (
	_ "embed"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rcy/whatever/app"
	"github.com/rcy/whatever/app/notes"
	"github.com/rcy/whatever/flog"
	"github.com/rcy/whatever/version"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
	"mvdan.cc/xurls/v2"
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
	r.Get("/deleted_notes", svc.deletedNotesHandler)
	r.Get("/notes/{id}", svc.showNoteHandler)
	r.Post("/notes/{id}/delete", svc.deleteNoteHandler)
	r.Post("/notes/{id}/undelete", svc.undeleteNoteHandler)
	r.Post("/notes", svc.postNotesHandler)
	r.Get("/events", svc.eventsHandler)
	return r
}

type Params struct {
	Main g.Node
}

func page(main g.Node) g.Node {
	var color string
	if version.IsRelease() {
		color = "jade"
	} else {
		color = "slate"
	}

	return h.HTML(h.Lang("en"),
		h.Head(
			h.Meta(h.Name("viewport"), h.Content("width=device-width, initial-scale=1")),
			//h.Meta(h.Name("color-scheme"), h.Content("light dark")),
			h.Link(h.Rel("stylesheet"), h.Href(fmt.Sprintf("https://cdn.jsdelivr.net/npm/@picocss/pico@2/css/pico.%s.min.css", color))),
			//h.Link(h.Rel("stylesheet"), h.Href("https://cdn.jsdelivr.net/npm/@picocss/pico@2/css/pico.colors.min.css")),
		),
		h.Body( //g.Attr("data-theme", "dark"),
			h.Div(h.Class("container"),
				h.Div(h.Style("display:flex; gap:1em; align-items:base-line"),
					h.H2(g.Text("whatever")),
					h.H2(h.A(h.Href("/notes"), g.Text("notes"))),
					h.H2(h.A(h.Href("/events"), g.Text("events"))),
					h.H2(h.A(h.Href("/deleted_notes"), g.Text("deleted"))),
				),
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
					h.Td(h.A(h.Href("/notes/"+note.ID), g.Text(note.ID[0:7]))),
					h.Td(g.Text(note.Ts.Local().Format(time.DateTime))),
					h.Td(linkifyNode(note.Text)),
				)
			}))),
	}).Render(w)
}

func (s *webservice) deletedNotesHandler(w http.ResponseWriter, r *http.Request) {
	noteList, err := s.app.NS.FindAllDeleted()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	slices.Reverse(noteList)

	page(g.Group{
		h.Table(h.Class("striped"), h.TBody(
			g.Map(noteList, func(note notes.Model) g.Node {
				return h.Tr(
					h.Td(g.Text(note.ID[0:7])),
					h.Td(g.Text(note.Ts.Local().Format(time.DateTime))),
					h.Td(linkifyNode(note.Text)),
					h.Td(
						h.Form(h.Style("margin:0"),
							h.Method("post"), h.Action(fmt.Sprintf("/notes/%s/undelete", note.ID)),
							h.Button(h.Class("outline secondary"), h.Style("padding:0 1em"), g.Text("undelete"))),
					),
				)
			}))),
	}).Render(w)
}

func (s *webservice) showNoteHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	note, err := s.app.NS.FindOne(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	page(g.Group{
		h.Div(h.Style("display:flex; align-items:baseline; justify-content:space-between"),
			h.H2(g.Text(id[0:7])),
			h.Form(h.Method("post"), h.Action(fmt.Sprintf("/notes/%s/delete", id)),
				h.Button(
					h.Class("outline secondary"),
					g.Text("delete"),
				),
			),
		),
		h.P(linkifyNode(note.Text)),
	}).Render(w)
}

func (s *webservice) deleteNoteHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	err := s.app.CS.DeleteNote(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/notes", http.StatusSeeOther)
}

func (s *webservice) undeleteNoteHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	err := s.app.CS.UndeleteNote(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/notes/%s", id), http.StatusSeeOther)
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
	events, err := s.app.ES.LoadAllEvents(true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	page(g.Group{
		h.Table(
			h.Body(
				g.Map(events, func(event flog.Model) g.Node {
					return h.Tr(
						h.Td(g.Text(fmt.Sprint(event.EventID))),
						h.Td(h.A(h.Code(g.Text(event.AggregateID[0:7])))),
						h.Td(g.Text(event.EventType)),
						h.Td(h.Code(g.Text(string(event.EventData)))),
					)
				}),
			),
		),
	},
	).Render(w)
}

func linkify(text string) string {
	re := xurls.Relaxed()
	return re.ReplaceAllStringFunc(text, func(match string) string {
		if strings.Contains(match, "@") {
			idxEmail := re.SubexpIndex("relaxedEmail")
			matches := re.FindStringSubmatch(match)
			if matches[idxEmail] != "" {
				// return email as is
				return matches[idxEmail]
			}
		}
		url, err := url.Parse(match)
		if err != nil {
			return match
		}
		if url.Scheme == "" {
			url.Scheme = "https"
		}
		return fmt.Sprintf(`<a href="%s">%s</a>`, url.String(), match)
	})
}

func linkifyNode(text string) g.Node {
	return g.Raw(linkify(text))
}
