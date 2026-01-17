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
	"github.com/google/uuid"
	"github.com/rcy/whatever/commands"
	"github.com/rcy/whatever/projections/note"
	"github.com/rcy/whatever/projections/realm"
	"github.com/rcy/whatever/version"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
	"mvdan.cc/xurls/v2"
)

type Params struct {
	Main g.Node
}

//go:embed style.css
var styles string

func page(currentRealmID uuid.UUID, realmList []realm.Realm, main g.Node) g.Node {
	color := "jade"
	brand := "whatever"
	if !version.IsRelease() {
		brand += "-dev"
		color = "slate"
	}

	return h.HTML(h.Lang("en"),
		h.Head(
			h.TitleEl(g.Text("Whatever NotNow")),
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
					//h.H2(h.A(h.Href("/events"), g.Text("events"))),
					h.H2(h.A(h.Href("/deleted_notes"), g.Text("deleted"))),
				),
				h.Div(
					h.Select(g.Attr("hx-post", "/realm"), h.Name("realm"),
						g.Map(realmList, func(realm realm.Realm) g.Node {
							return h.Option(
								h.Value(realm.ID.String()),
								g.Text(realm.Name),
								g.If(currentRealmID == realm.ID, h.Selected()),
							)
						})),
				),
				h.Main(main),
			),
		))
}

func page2(realmID uuid.UUID, realmList []realm.Realm, category string, categoryCounts []note.CategoryCount, content g.Node) g.Node {
	return page(realmID, realmList, h.Div(h.ID("page"),
		h.Form(h.Input(h.Name("text"), h.Placeholder("add note...")),
			g.Attr("hx-post", "/notes"),
			g.Attr("hx-swap", "outerHTML"),
			g.Attr("hx-target", "#page"),
			g.Attr("hx-select", "#page")),
		h.Div(h.Style("display:flex; gap: 2em"),
			h.Div(h.Style("display:flex;flex-direction:column; margin-top: 2em; white-space: nowrap"),
				g.Map(categoryCounts, func(cc note.CategoryCount) g.Node {
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

	categoryCounts, err := s.app.Notes.CategoryCounts(realmID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	realmList, err := s.app.Realms.FindAll()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	noteList, err := s.app.Notes.FindAllInRealmByCategory(realmID, category)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	slices.Reverse(noteList)

	page2(realmID, realmList, category, categoryCounts,
		h.Table(h.Class("striped"),
			h.THead(
				h.Th(g.Text("text")),
				h.Th(g.Text("created")),
			),
			h.TBody(
				g.Map(noteList, func(note note.Note) g.Node {
					url := fmt.Sprintf("/notes/%s/%s", category, note.ID)
					return h.Tr(h.ID(fmt.Sprintf("note-%s", note.ID)),
						h.Td(linkifyNode(note.Status+" "+note.Text)),
						h.Td(h.A(h.Href(url), (g.Text(note.Ts.Local().Format(time.DateTime))))),
						h.Td(h.Style("padding:0"),
							g.If(category == "inbox",
								h.Div(h.Style("display:flex; gap:5px"),
									g.Map(xcategories,
										func(category string) g.Node {
											return h.Button(
												h.Style("padding:0 .5em"),
												h.Class("outline"),
												g.Text(category),
												g.Attr("hx-post", fmt.Sprintf("/notes/%s/set/%s", note.ID, category)),
												g.Attr("hx-target", fmt.Sprintf("#note-%s", note.ID)),
												g.Attr("hx-swap", "delete swap:1s"),
											)
										})))))
				})))).Render(w)
}

var xcategories = []string{"task", "reminder", "idea", "reference", "observation"}

func (s *webservice) postSetNotesCategoryHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	category := chi.URLParam(r, "category")

	noteID, err := uuid.Parse(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = s.app.Commander.Send(commands.SetNoteCategory{NoteID: noteID, Category: category})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, r.Header.Get("hx-current-url"), http.StatusSeeOther)
}

func (s *webservice) deletedNotesHandler(w http.ResponseWriter, r *http.Request) {
	realm := realmFromRequest(r)

	noteList, err := s.app.Notes.FindAllDeleted()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	slices.Reverse(noteList)

	realmList, err := s.app.Realms.FindAll()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	page(realm, realmList, g.Group{
		h.Table(h.Class("striped"), h.TBody(
			g.Map(noteList, func(note note.Note) g.Node {
				return h.Tr(
					h.Td(g.Text(fmt.Sprint(note))),
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

	note, err := s.app.Notes.FindOne(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	realmList, err := s.app.Realms.FindAll()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	categoryCounts, err := s.app.Notes.CategoryCounts(realmID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	page2(realmID, realmList, note.Category, categoryCounts,
		noteNode(realmID, realmList, note, h.Div(
			h.P(linkifyNode(note.Text)),
			h.A(h.Href(fmt.Sprintf("%s/edit", note.ID)), g.Text("edit")),
		))).Render(w)
}

func (s *webservice) showEditNoteHandler(w http.ResponseWriter, r *http.Request) {
	realmID := realmFromRequest(r)

	id := chi.URLParam(r, "id")

	note, err := s.app.Notes.FindOne(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	switch r.Method {
	case "GET":
		realmList, err := s.app.Realms.FindAll()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		categoryCounts, err := s.app.Notes.CategoryCounts(realmID)
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
						h.Div(h.A(g.Text("cancel"), h.Href(fmt.Sprintf("/notes/%s/%s", note.Category, note.ID))))),
				),
			)).Render(w)
	case "POST":
		text := strings.TrimSpace(r.FormValue("text"))
		err := s.app.Commander.Send(commands.UpdateNoteText{NoteID: note.ID, Text: text})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		http.Redirect(w, r, fmt.Sprintf("/notes/%s/%s", note.Category, id), http.StatusSeeOther)
	default:
		http.Error(w, "", http.StatusMethodNotAllowed)
	}
}

func noteNode(realmID uuid.UUID, realmList []realm.Realm, note note.Note, slot g.Node) g.Node {
	base := fmt.Sprintf("/notes/%s/set/", note.ID)
	return h.Div(h.ID("hxnote"),
		h.Div(h.Style("display:flex; align-items:baseline; justify-content:space-between"),
			h.Div(h.Class("uwu"), h.Style("display:flex; gap:5px"),
				g.Map(xcategories,
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
	noteID, err := uuid.Parse(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = s.app.Commander.Send(commands.DeleteNote{NoteID: noteID})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/notes/"+chi.URLParam(r, "category"), http.StatusSeeOther)
}

func (s *webservice) undeleteNoteHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	noteID, err := uuid.Parse(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = s.app.Commander.Send(commands.UndeleteNote{NoteID: noteID})
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
		err := s.app.Commander.Send(commands.CreateNote{NoteID: uuid.New(), RealmID: realmID, Text: text})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	url := r.Header.Get("HX-Current-URL")
	fmt.Println("url", url)
	http.Redirect(w, r, url, http.StatusSeeOther)
}

// func (s *webservice) eventsHandler(w http.ResponseWriter, r *http.Request) {
// 	realm := realmFromRequest(r)

// 	events, err := s.app.Events().DebugEvents()
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}

// 	realmList, err := s.app.Realms().FindAll()
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}

// 	page(realm, realmList, g.Group{
// 		h.Table(
// 			h.Body(
// 				g.Map(events, func(event evoke.RecordedEvent) g.Node {
// 					return h.Tr(
// 						h.Td(g.Text(fmt.Sprint(event.Sequence))),
// 						h.Td(h.A(h.Code(g.Text(event.AggregateID.String())))),
// 						h.Td(g.Text(event.Event.EventType())),
// 						h.Td(h.Code(g.Text(fmt.Sprint(event.Event)))),
// 					)
// 				}),
// 			),
// 		),
// 	},
// 	).Render(w)
// }

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
		domain, err := getDomain(match)
		return fmt.Sprintf(`<a href="%s">%s</a>`, url.String(), "|"+domain+"|")
	})
}

// return the domain from the url with leading www removed
func getDomain(link string) (string, error) {
	if !strings.HasPrefix(link, "http") {
		link = "https://" + link
	}
	url, err := url.Parse(link)
	if err != nil {
		return "", err
	}

	host := strings.TrimLeft(url.Host, "www.")

	return host, nil
}

func linkifyNode(text string) g.Node {
	return g.Raw(linkify(text))
}
