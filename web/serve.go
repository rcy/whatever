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
	"github.com/rcy/whatever/projections/realms"
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
	svc := webservice{app: app}
	r.Use(middleware.Logger)
	r.Use(svc.realmMiddleware)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/notes", http.StatusSeeOther)
	})

	r.Get("/deleted_notes", svc.deletedNotesHandler)
	r.Get("/events", svc.eventsHandler)

	r.Post("/realm", func(w http.ResponseWriter, r *http.Request) {
		svc.setRealmCookie(w, r, r.FormValue("realm"))
		w.Header().Set("HX-Redirect", "")
	})

	r.Route("/notes", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/notes/inbox", http.StatusSeeOther)
		})
		r.Post("/", svc.postNotesHandler)
		r.Post("/{id}/set/{category}", svc.postSetNotesCategoryHandler)
		r.Route("/{category}", func(r chi.Router) {
			r.Get("/", svc.notesHandler)
			r.Get("/{id}", svc.showNoteHandler)

			r.HandleFunc("/{id}/edit", svc.showEditNoteHandler)

			r.Post("/{id}/delete", svc.deleteNoteHandler)
			r.Post("/{id}/undelete", svc.undeleteNoteHandler)
		})
	})

	return r
}

type Params struct {
	Main g.Node
}

//go:embed style.css
var styles string

func page(currentRealmID string, realmList []realms.Realm, main g.Node) g.Node {
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
				h.Div(
					h.Select(g.Attr("hx-post", "/realm"), h.Name("realm"),
						g.Map(realmList, func(realm realms.Realm) g.Node {
							return h.Option(
								h.Value(realm.ID),
								g.Text(realm.Name),
								g.If(currentRealmID == realm.ID, h.Selected()),
							)
						})),
				),
				h.Main(main),
			),
		))
}

func page2(realmID string, realmList []realms.Realm, category string, categoryCounts []notes.CategoryCount, content g.Node) g.Node {
	return page(realmID, realmList, h.Div(h.ID("page"),
		h.Form(h.Input(h.Name("text"), h.Placeholder("add note...")),
			g.Attr("hx-post", "/notes"),
			g.Attr("hx-swap", "outerHTML"),
			g.Attr("hx-target", "#page"),
			g.Attr("hx-select", "#page")),
		h.Div(h.Style("display:flex; gap: 2em"),
			h.Div(h.Style("display:flex;flex-direction:column; margin-top: 2em; white-space: nowrap"),
				g.Map(categoryCounts, func(cc notes.CategoryCount) g.Node {
					text := fmt.Sprintf("%s %d", g.Text(cc.Category), cc.Count)
					if cc.Category == category {
						return h.Div(h.B(g.Text(text)))
					} else {
						return h.Div(h.A(g.Text(text), h.Href("/notes/"+cc.Category)))
					}
				})),
			content,
		)))
}

func (s *webservice) notesHandler(w http.ResponseWriter, r *http.Request) {
	realmID := realmFromRequest(r)
	category := chi.URLParam(r, "category")

	categoryCounts, err := s.app.Notes().CategoryCounts(realmID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	realmList, err := s.app.Realms().FindAll()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	noteList, err := s.app.Notes().FindAllInRealmByCategory(realmID, category)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	slices.Reverse(noteList)

	page2(realmID, realmList, category, categoryCounts,
		h.Table(h.Class("striped"),
			h.THead(
				h.Th(g.Text("id")),
				h.Th(g.Text("text")),
				h.Th(g.Text("created")),
			),
			h.TBody(
				g.Map(noteList, func(note notes.Note) g.Node {
					return h.Tr(h.ID("note-"+note.ID),
						h.Td(h.A(h.Href("/notes/"+category+"/"+note.ID), g.Text(note.ID[0:7]))),
						h.Td(linkifyNode(note.Text)),
						h.Td(g.Text(note.Ts.Local().Format(time.DateTime))),
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
				})))).Render(w)
}

var categories = []string{"task", "reminder", "idea", "reference", "observation"}

func (s *webservice) postSetNotesCategoryHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	category := chi.URLParam(r, "category")

	err := s.app.Commands().SetNoteCategory(id, category)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, r.Header.Get("hx-current-url"), http.StatusSeeOther)
}

func (s *webservice) deletedNotesHandler(w http.ResponseWriter, r *http.Request) {
	realm := realmFromRequest(r)

	noteList, err := s.app.Notes().FindAllDeleted()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	slices.Reverse(noteList)

	realmList, err := s.app.Realms().FindAll()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	page(realm, realmList, g.Group{
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
	realmID := realmFromRequest(r)

	id := chi.URLParam(r, "id")

	note, err := s.app.Notes().FindOne(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	realmList, err := s.app.Realms().FindAll()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	categoryCounts, err := s.app.Notes().CategoryCounts(realmID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	page2(realmID, realmList, note.Category, categoryCounts,
		noteNode(realmID, realmList, note, h.Div(
			h.P(linkifyNode(note.Text)),
			h.A(h.Href(note.ID+"/edit"), g.Text("edit")),
		))).Render(w)
}

func (s *webservice) showEditNoteHandler(w http.ResponseWriter, r *http.Request) {
	realmID := realmFromRequest(r)

	id := chi.URLParam(r, "id")

	note, err := s.app.Notes().FindOne(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	switch r.Method {
	case "GET":
		realmList, err := s.app.Realms().FindAll()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		categoryCounts, err := s.app.Notes().CategoryCounts(realmID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		page2(realmID, realmList, note.Category, categoryCounts,
			noteNode(realmID, realmList, note,
				h.Form(h.Method("post"),
					h.Input(h.Name("text"), h.Value(note.Text)),
					h.Div(h.Style("display:flex; gap:1em"),
						h.Button(h.Style("padding: 0 .5em"), g.Text("save")),
						h.Div(h.A(g.Text("cancel"), h.Href("/notes/"+note.Category+"/"+note.ID)))),
				),
			)).Render(w)
	case "POST":
		text := strings.TrimSpace(r.FormValue("text"))
		err := s.app.Commands().UpdateNoteText(id, text)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		http.Redirect(w, r, fmt.Sprintf("/notes/%s/%s", note.Category, id), http.StatusSeeOther)
	default:
		http.Error(w, "", http.StatusMethodNotAllowed)
	}
}

func noteNode(realm string, realmList []realms.Realm, note notes.Note, slot g.Node) g.Node {
	base := fmt.Sprintf("/notes/%s/set/", note.ID)
	return h.Div(h.ID("hxnote"),
		h.Div(h.Style("display:flex; align-items:baseline; justify-content:space-between"),
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

			h.Form(h.Method("post"), h.Action(fmt.Sprintf("/notes/%s/%s/delete", note.Category, note.ID)),
				h.Button(
					h.Style("padding:0 .5em"),
					h.Class("outline secondary"),
					g.Text("delete"),
				),
			),
		),
		h.P(g.Text("Created: "+note.Ts.String())),
		h.Hr(),
		slot,
	)
}

func (s *webservice) deleteNoteHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	err := s.app.Commands().DeleteNote(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/notes/"+chi.URLParam(r, "category"), http.StatusSeeOther)
}

func (s *webservice) undeleteNoteHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	err := s.app.Commands().UndeleteNote(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/notes/%s", id), http.StatusSeeOther)
}

func (s *webservice) postNotesHandler(w http.ResponseWriter, r *http.Request) {
	realmID := realmFromRequest(r)
	text := strings.TrimSpace(r.FormValue("text"))
	if text != "" {
		_, err := s.app.Commands().CreateNote(realmID, text)
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
	realm := realmFromRequest(r)

	events, err := s.app.Events().LoadAllEvents(true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	realmList, err := s.app.Realms().FindAll()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	page(realm, realmList, g.Group{
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
