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
	"github.com/rcy/evoke"
	"github.com/rcy/whatever/app"
	"github.com/rcy/whatever/projections/notes"
	"github.com/rcy/whatever/version"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
	"mvdan.cc/xurls/v2"
)

type webservice struct {
	app *app.App
}

func Server(app *app.App) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	svc := webservice{app: app}
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/notes", http.StatusSeeOther)
	})
	r.Get("/notes", svc.notesHandler)
	r.Get("/deleted_notes", svc.deletedNotesHandler)
	r.Get("/notes/{id}", svc.showNoteHandler)
	r.Get("/notes/{id}/edit", svc.showEditNoteHandler)
	r.Post("/notes/{id}/edit", svc.postEditNoteHandler)
	r.Post("/notes/{id}/delete", svc.deleteNoteHandler)
	r.Post("/notes/{id}/undelete", svc.undeleteNoteHandler)
	r.Post("/notes/{id}/set/{category}", svc.postSetNotesCategoryHandler)
	r.Post("/notes", svc.postNotesHandler)
	r.Get("/events", svc.eventsHandler)
	return r
}

type Params struct {
	Main g.Node
}

//go:embed style.css
var styles string

func page(main g.Node) g.Node {
	color := "jade"
	brand := "whatever"
	if !version.IsRelease() {
		brand += "-dev"
		color = "slate"
	}

	return h.HTML(h.Lang("en"),
		h.Head(
			h.Meta(h.Name("viewport"), h.Content("width=device-width, initial-scale=1")),
			//h.Meta(h.Name("color-scheme"), h.Content("light dark")),
			h.Link(h.Rel("stylesheet"), h.Href(fmt.Sprintf("https://cdn.jsdelivr.net/npm/@picocss/pico@2/css/pico.%s.min.css", color))),
			//h.Link(h.Rel("stylesheet"), h.Href("https://cdn.jsdelivr.net/npm/@picocss/pico@2/css/pico.colors.min.css")),
			h.Script(h.Src("https://cdn.jsdelivr.net/npm/htmx.org@2.0.6/dist/htmx.min.js")),
			h.Script(h.Src("https://unpkg.com/htmx-ext-class-tools@2.0.1/class-tools.js")),
			h.StyleEl(g.Raw(styles)),
		),
		h.Body( //g.Attr("data-theme", "dark"),
			h.Div(h.Class("container"),
				h.Div(h.Style("display:flex; gap:1em; align-items:base-line"),
					h.H2(g.Text(brand)),
					h.H2(h.A(h.Href("/notes"), g.Text("notes"))),
					h.H2(h.A(h.Href("/events"), g.Text("events"))),
					h.H2(h.A(h.Href("/deleted_notes"), g.Text("deleted"))),
				),
				h.Main(main),
			)),
	)
}

func (s *webservice) notesHandler(w http.ResponseWriter, r *http.Request) {
	category := r.FormValue("category")

	var noteList []notes.Note
	var err error
	if category == "" {
		noteList, err = s.app.Notes.FindAll()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		noteList, err = s.app.Notes.FindAllByCategory(category)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	slices.Reverse(noteList)

	categoryCounts, err := s.app.Notes.CategoryCounts()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	page(h.Div(h.ID("page"),
		h.Form(h.Input(h.AutoFocus(), h.Name("text")),
			g.Attr("hx-post", "/notes"),
			g.Attr("hx-swap", "outerHTML"),
			g.Attr("hx-target", "#page"),
			g.Attr("hx-select", "#page")),
		h.Div(h.Style("display:flex; gap:1em"),
			//h.A(g.Text("inbox"), h.Href("?category=inbox")),
			g.Map(categoryCounts, func(cc notes.CategoryCount) g.Node {
				if cc.Count > 0 {
					return h.A(g.Text(fmt.Sprintf("%s %d", g.Text(cc.Category), cc.Count)),
						h.Href("?category="+cc.Category))
				} else {
					return g.Text(cc.Category)
				}
			}),
			//h.A(g.Text("all"), h.Href("?category")),
		),
		h.Table(h.Class("striped"), h.TBody(
			g.Map(noteList, func(note notes.Note) g.Node {
				return h.Tr(h.ID("note-"+note.ID),
					h.Td(h.A(h.Href("/notes/"+note.ID), g.Text(note.ID[0:7]))),
					h.Td(g.Text(note.Ts.Local().Format(time.DateTime))),
					h.Td(linkifyNode(note.Text)),
					h.Td(h.Style("padding:0"),
						g.If(category == "inbox",
							h.Div(h.Style("display:flex; gap:5px"),
								g.Map(categories,
									func(category string) g.Node {
										return h.Button(
											h.Style("padding:0 .5em"),
											h.Class("outline"),
											g.Text(category),
											g.Attr("hx-post", fmt.Sprintf("/notes/%s/set/%s", note.ID, category)),
											g.Attr("hx-target", "#note-"+note.ID),
											g.Attr("hx-swap", "delete swap:1s"),
										)
									})))))
			}))),
	)).Render(w)
}

var categories = []string{"task", "reminder", "idea", "reference", "observation"}

func (s *webservice) postSetNotesCategoryHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	category := chi.URLParam(r, "category")

	err := s.app.Commands.SetNoteCategory(id, category)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, r.Header.Get("hx-current-url"), http.StatusSeeOther)
}

func (s *webservice) deletedNotesHandler(w http.ResponseWriter, r *http.Request) {
	noteList, err := s.app.Notes.FindAllDeleted()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	slices.Reverse(noteList)

	page(g.Group{
		h.Table(h.Class("striped"), h.TBody(
			g.Map(noteList, func(note notes.Note) g.Node {
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

	note, err := s.app.Notes.FindOne(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	base := fmt.Sprintf("/notes/%s/set/", note.ID)
	noteNode(note, h.Div(
		h.P(linkifyNode(note.Text)),
		h.Div(h.Class("uwu"), h.Style("display:flex; gap:5px"),
			g.Map(categories,
				func(category string) g.Node {
					return h.Button(
						g.If(category != note.Category, h.Class("outline")),
						h.Style("padding:0 .5em; margin:0"),
						g.Text(category),
						g.Attr("hx-post", base+category),
						g.Attr("hx-target", "#hxnote"),
						g.Attr("hx-select", "#hxnote"),
						g.Attr("hx-swap", "outerHTML"),
					)
				}),
		),
		h.A(h.Href(note.ID+"/edit"), g.Text("edit")),
	)).Render(w)
}

func (s *webservice) showEditNoteHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	note, err := s.app.Notes.FindOne(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	noteNode(note, g.Group{
		h.Form(h.Method("post"),
			h.Input(h.Name("text"), h.Value(note.Text)),
			h.Div(h.Style("display:flex; gap:1em"),
				h.Button(g.Text("save")),
				h.Div(h.A(g.Text("cancel"), h.Href("/notes/"+note.ID)))),
		),
	}).Render(w)
}

func (s *webservice) postEditNoteHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	text := strings.TrimSpace(r.FormValue("text"))
	if text != "" {
		err := s.app.Commands.UpdateNoteText(id, text)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	http.Redirect(w, r, "/notes", http.StatusSeeOther)
}

func noteNode(note notes.Note, slot g.Node) g.Node {
	return page(g.Group{
		h.Div(h.ID("hxnote"),
			h.Div(h.Style("display:flex; align-items:baseline; justify-content:space-between"),
				h.H6(g.Text(note.ID[0:7])),
				h.P(g.Text(note.Category)),
				h.Form(h.Method("post"), h.Action(fmt.Sprintf("/notes/%s/delete", note.ID)),
					h.Button(
						h.Class("outline secondary"),
						g.Text("delete"),
					),
				),
			),
			h.P(g.Text("Created: "+note.Ts.String())),
			slot,
		)})
}

func (s *webservice) deleteNoteHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	err := s.app.Commands.DeleteNote(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/notes", http.StatusSeeOther)
}

func (s *webservice) undeleteNoteHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	err := s.app.Commands.UndeleteNote(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/notes/%s", id), http.StatusSeeOther)
}

func (s *webservice) postNotesHandler(w http.ResponseWriter, r *http.Request) {
	text := strings.TrimSpace(r.FormValue("text"))
	if text != "" {
		_, err := s.app.Commands.CreateNote(text)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	url := r.Header.Get("HX-Current-URL")
	fmt.Println("url", url)
	http.Redirect(w, r, url, http.StatusSeeOther)
}

func (s *webservice) eventsHandler(w http.ResponseWriter, r *http.Request) {
	events, err := s.app.Events.LoadAllEvents(true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	page(g.Group{
		h.Table(
			h.Body(
				g.Map(events, func(event evoke.Event) g.Node {
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
