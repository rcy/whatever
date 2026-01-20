package web

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/kkdai/youtube/v2"
	"github.com/rcy/evoke"
	"github.com/rcy/whatever/app"
	"github.com/rcy/whatever/catalog/notesmeta"
	"github.com/rcy/whatever/commands"
	"github.com/rcy/whatever/projections/note"
	"github.com/rcy/whatever/projections/realm"
	"github.com/starfederation/datastar-go/datastar"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

//go:embed style.css
var styles string

type Config struct {
	BaseURL            string
	GoogleClientID     string
	GoogleClientSecret string
	SessionSecret      string
}

type webservice struct {
	app         *app.App
	oauthConfig *oauth2.Config
	sessions    *sessionManager
	states      stateManager
	baseURL     string
}

func Server(app *app.App, cfg Config) (*chi.Mux, error) {
	if cfg.BaseURL == "" {
		return nil, errors.New("web: base URL is required for oauth redirect")
	}
	if cfg.GoogleClientID == "" || cfg.GoogleClientSecret == "" {
		return nil, errors.New("web: google oauth client id and secret are required")
	}
	sessions, err := newSessionManager(cfg.SessionSecret)
	if err != nil {
		return nil, fmt.Errorf("web: %w", err)
	}

	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	svc := webservice{
		app: app,
		oauthConfig: &oauth2.Config{
			ClientID:     cfg.GoogleClientID,
			ClientSecret: cfg.GoogleClientSecret,
			RedirectURL:  baseURL + "/auth/callback",
			Scopes: []string{
				"https://www.googleapis.com/auth/userinfo.email",
				"https://www.googleapis.com/auth/userinfo.profile",
				"openid",
			},
			Endpoint: google.Endpoint,
		},
		sessions: sessions,
		states:   stateManager{},
		baseURL:  baseURL,
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(svc.realmMiddleware)

	r.Get("/auth", svc.authHandler)
	r.Get("/auth/callback", svc.authCallbackHandler)
	r.Get("/logout", svc.logoutHandler)

	r.Group(func(r chi.Router) {
		r.Use(svc.authMiddleware)

		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/dsnotes/"+notesmeta.DefaultCategory.Name, http.StatusSeeOther)
		})

		r.Post("/realm", func(w http.ResponseWriter, r *http.Request) {
			svc.setRealmCookie(w, r, r.FormValue("realm"))
			w.Header().Set("HX-Redirect", "")
		})

		r.Get("/note/{id}", svc.showNote)

		r.Get("/events", svc.eventsIndex)

		r.Get("/dsnotes/{category}", svc.notesIndexRedirect)
		r.Get("/dsnotes/{category}/{subcategory}", svc.notesIndex)

		r.Post("/dsnotes", svc.postNotesHandler)
		r.Post("/refile/{noteID}/{category}", svc.postRefileNote)
		r.Post("/subfile/{noteID}/{subcategory}", svc.postSubfileNote)
	})

	return r, nil
}

type signals struct {
	Body            string `json:"body"`
	ViewCategory    string `json:"viewCategory"`
	ViewSubcategory string `json:"viewSubcategory"`
}

func (s *webservice) postNotesHandler(w http.ResponseWriter, r *http.Request) {
	userInfo := getUserInfo(r)

	realmID := realmFromRequest(r)

	var signals signals
	err := datastar.ReadSignals(r, &signals)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if signals.Body != "" {
		err := s.app.Commander.Send(commands.CreateNote{
			Owner:       userInfo.Id,
			NoteID:      uuid.New(),
			RealmID:     realmID,
			Text:        signals.Body,
			Category:    notesmeta.Inbox.Name,
			Subcategory: notesmeta.Inbox.DefaultSubcategory().Name,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	sse := datastar.NewSSE(w, r)

	// if we are looking at the inbox, reload the page to immediately show the item that was added to the inbox
	if signals.ViewCategory == "inbox" {
		sse.Redirect("")
		return
	}

	// ...otherwise, patch up the ui

	signals.Body = ""
	sse.MarshalAndPatchSignals(signals)

	headerEl, err := s.header(r, signals.ViewCategory, signals.ViewSubcategory)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sse.PatchElementGostar(headerEl)
}

// Wrap ui header element with data fetching
func (s *webservice) header(r *http.Request, viewCategory string, viewSubcategory string) (g.Node, error) {
	realmID := realmFromRequest(r)
	owner := getUserInfo(r)

	categoryCounts, err := s.app.Notes.CategoryCounts(owner.Id, realmID)
	if err != nil {
		return nil, fmt.Errorf("Notes.CategoryCounts: %w", err)
	}
	realmList, err := s.app.Realms.FindAll()
	if err != nil {
		return nil, fmt.Errorf("Realms.FindAll: %w", err)
	}

	subcategoryCounts, err := s.app.Notes.SubcategoryCounts(owner.Id, realmID, viewCategory)
	if err != nil {
		return nil, fmt.Errorf("Notes.SubcategoryCounts: %w", err)
	}

	return header(realmID, realmList, viewCategory, viewSubcategory, categoryCounts, subcategoryCounts), nil
}

func (s *webservice) eventsIndex(w http.ResponseWriter, r *http.Request) {
	events, err := s.app.EventDebugger.DebugEvents()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	slices.Reverse(events)

	h.HTML(
		h.Body(
			h.H1(g.Text("events")),
			g.Map(events, func(ev evoke.RecordedEvent) g.Node {
				data, err := json.Marshal(ev)
				if err != nil {
					return h.Div(h.Style("color: red"), g.Text(err.Error()))
				}
				return h.Div(g.Text(string(data)))
			}),
		),
	).Render(w)
}

func (s *webservice) showNote(w http.ResponseWriter, r *http.Request) {
	noteID := chi.URLParam(r, "id")
	note, err := s.app.Notes.FindOne(noteID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// show the note
	// show some buttons
	// edit / delete / archive

	links, err := noteLinksEl(note)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	actions := h.Div(
		h.Div(h.A(g.Text("torrent"), h.Href("https://thepiratebay11.com/search/"+url.PathEscape(note.Text)))),
		h.Div(h.A(g.Text("ddg"), h.Href("https://duckduckgo.com/?q="+url.QueryEscape(note.Text)))),
		h.Div(h.A(g.Text("goog"), h.Href("https://www.youtube.com/results?search_query="+url.PathEscape(note.Text)))),
		h.Div(h.A(g.Text("wiki"), h.Href("https://en.wikipedia.org/w/index.php?title=Special:Search&search="+url.QueryEscape(note.Text)))),
	)

	page := h.Div(
		noteEl(note),
		// h.Form(
		// 	g.Attr("data-on:submit", fmt.Sprintf("@post('/note/%s/comment', {contentType: 'form'})", note.ID)),
		// 	h.Textarea(
		// 		h.Name("bodyx"),
		// 		h.Rows("3"),
		// 		h.Style("width:100%"),
		// 	),
		// 	h.Button(g.Text("submit")),
		// ),
		links,
		actions,
		//youtubeDownloadButton(note),
	)

	content, err := s.page(r, note.Category, note.Subcategory, page)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	content.Render(w)
}

func noteLinksEl(note note.Note) (g.Node, error) {
	links := getLinks(note.Text)

	return h.Div(
		g.Map(links, func(link string) g.Node {
			embed, err := youtubeEmbed(link)
			if err != nil {
				h.Div(g.Text("error DKNh"))
			}

			return h.Div(
				h.Div(g.Text(link)),
				embed)
		})), nil
}

func youtubeEmbed(link string) (g.Node, error) {
	videoID, err := youtube.ExtractVideoID(link)
	if err != nil {
		return nil, err
	}
	return h.IFrame(h.Width("560"), h.Height("315"),
		h.Src("https://www.youtube.com/embed/"+videoID),
		g.Attr("allowfullscreen")), nil
}

// redirect to the default subcategory
func (s *webservice) notesIndexRedirect(w http.ResponseWriter, r *http.Request) {
	category := chi.URLParam(r, "category")
	defaultSubcategory := notesmeta.Categories.Get(category).DefaultSubcategory().Name
	http.Redirect(w, r, fmt.Sprintf("%s/%s", category, defaultSubcategory), http.StatusSeeOther)
}

func (s *webservice) notesIndex(w http.ResponseWriter, r *http.Request) {
	realmID := realmFromRequest(r)
	category := chi.URLParam(r, "category")
	subcategory := chi.URLParam(r, "subcategory")
	owner := getUserInfo(r)

	var noteList []note.Note
	var err error

	if subcategory == "all" {
		noteList, err = s.app.Notes.FindAllInRealmByCategory(owner.Id, realmID, category)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		noteList, err = s.app.Notes.FindAllInRealmByCategoryAndSubcategory(owner.Id, realmID, category, subcategory)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	slices.Reverse(noteList)

	content, err := s.page(r, category, subcategory, notes(noteList))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	content.Render(w)
}

func (s *webservice) page(r *http.Request, category string, subcategory string, node g.Node) (g.Node, error) {
	headerEl, err := s.header(r, category, subcategory)
	if err != nil {
		return nil, err
	}
	return h.HTML(
		h.Head(
			h.Script(h.Type("module"), h.Src("https://cdn.jsdelivr.net/gh/starfederation/datastar@1.0.0-RC.7/bundles/datastar.js")),
			h.StyleEl(g.Raw(styles)),
		),
		h.Body(
			h.Div(g.Attr("data-signals", fmt.Sprintf("{viewCategory: '%s', viewSubcategory: '%s'}", category, subcategory))),
			h.Div(h.Style("display:flex;flex-direction:column;gap:10px"),
				h.Div(headerEl),
				h.Div(input()),
				h.Div(node),
			),
		),
	), nil
}

func (s *webservice) postRefileNote(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "noteID")
	categoryName := chi.URLParam(r, "category")

	var signals signals
	err := datastar.ReadSignals(r, &signals)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	noteID, err := uuid.Parse(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = s.app.Commander.Send(commands.SetNoteCategory{NoteID: noteID, Category: categoryName})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sse := datastar.NewSSE(w, r)

	headerEl, err := s.header(r, signals.ViewCategory, signals.ViewSubcategory)
	if err != nil {
		sse.ConsoleError(err)
		return
	}
	sse.PatchElementGostar(headerEl)

	note, err := s.app.Notes.FindOne(noteID.String())
	if err != nil {
		sse.ConsoleError(err)
		return
	}
	noteEl := noteEl(note)

	sse.PatchElementGostar(noteEl)
}

func (s *webservice) postSubfileNote(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "noteID")
	subcategory := chi.URLParam(r, "subcategory")

	var signals signals
	err := datastar.ReadSignals(r, &signals)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	sse := datastar.NewSSE(w, r)

	noteID, err := uuid.Parse(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = s.app.Commander.Send(commands.SetNoteSubcategory{NoteID: noteID, Subcategory: subcategory})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	headerEl, err := s.header(r, signals.ViewCategory, signals.ViewSubcategory)
	if err != nil {
		sse.ConsoleError(err)
		return
	}
	sse.PatchElementGostar(headerEl)

	note, err := s.app.Notes.FindOne(noteID.String())
	if err != nil {
		sse.ConsoleError(err)
		return
	}
	noteEl := noteEl(note)

	sse.PatchElementGostar(noteEl)
}

func input() g.Node {
	return h.Form(h.ID("input-form"), g.Attr("data-on:submit", "@post('/dsnotes')"), h.Style("margin:0"),
		h.Input(
			g.Attr("data-bind", "body"),
			h.Style("width:100%"),
			h.Placeholder("add a note..."),
			h.AutoFocus(),
		),
	)
}

func header(realmID uuid.UUID, realmList []realm.Realm, category string, subcategory string, categoryCounts []note.CategoryCount, subcategoryCounts []note.SubcategoryCount) g.Node {
	return h.Div(h.ID("header"),
		h.Div(h.Style("background: lime; padding: 5px; display:flex; justify-content:space-between"),
			h.Div(h.Style("display:flex; gap:5px"),
				h.Div(h.Style("font-weight: bold"), g.Text("Not Now")),
				h.Div(h.Style("display: flex; gap: 5px"),
					g.Map(notesmeta.Categories, func(c notesmeta.Category) g.Node {
						text := fmt.Sprintf("[%s]", c.Name)
						if c.Name == category {
							return h.Div(
								h.A(h.Style("font-weight: bold"),
									g.Text(text),
									h.Href("/dsnotes/"+c.Name)))
						} else {
							return h.Div(h.A(g.Text(text), h.Href("/dsnotes/"+c.Name)))
						}
					}))),
			h.Div(
				h.Select(g.Attr("hx-post", "/realm"), h.Name("realm"),
					g.Map(realmList, func(realm realm.Realm) g.Node {
						return h.Option(
							h.Value(realm.ID.String()),
							g.Text(realm.Name),
							g.If(realmID == realm.ID, h.Selected()),
						)
					})),
			)),

		g.If(len(notesmeta.Categories.Get(category).Subcategories) > 1,
			h.Div(h.Style("background: pink; padding: 5px; display:flex; justify-content: space-between;"),
				h.Div(h.Style("display: flex; gap: 5px"),
					g.Map(notesmeta.Categories.Get(category).Subcategories, func(s notesmeta.Subcategory) g.Node {
						text := fmt.Sprintf("[%s]", g.Text(s.Name))
						var style g.Node
						if s.Name == subcategory {
							style = h.Style("font-weight: bold")
						}
						return h.Div(h.A(style, g.Text(text), h.Href(fmt.Sprintf("/dsnotes/%s/%s", category, s.Name))))
					}),
				),
				h.Div(h.A(g.Text("[all]"), h.Href(fmt.Sprintf("/dsnotes/%s/all", category)))),
			)),
	)
}

func notes(noteList []note.Note) g.Node {
	return h.Div(h.Style("display:flex; flex-direction:column; gap:10px"),
		g.Map(noteList, func(note note.Note) g.Node {
			return noteEl(note)
		}),
	)
}

func noteID(note note.Note) string {
	return fmt.Sprintf("note-%s", note.ID)
}

// Return a link to the note
func noteLink(note note.Note) string {
	return fmt.Sprintf("/note/%s", note.ID)
}

func noteEl(note note.Note) g.Node {
	return h.Div(h.ID(noteID(note)),
		h.Div(
			h.A(h.Href(noteLink(note)),
				linkifyNode(note.Status+" "+note.Text)),
		),
		h.Div(h.Style("color: gray; font-size: 70%; margin-top: -3px;"),
			h.Div(h.Style("display:flex; gap:2px"),
				refile(note),
			),
		),
	)
}

func refile(note note.Note) g.Node {
	if note.Category == "inbox" {
		return h.Div(h.Style("display:flex; gap:2px"),
			g.Map(notesmeta.Categories, func(c notesmeta.Category) g.Node {
				if c.Name == "inbox" {
					return nil
				}
				return refileButton(note.ID, c.Name, c.Name)
			}))
	}

	transitions := notesmeta.Categories.Get(note.Category).Subcategories.Get(note.Subcategory).Transitions

	return h.Div(h.Style("display:flex; gap:2px"),
		g.Group{
			g.Map(transitions,
				func(t notesmeta.Transition) g.Node {
					return subfileButton(note.ID, t)
				},
			),
			g.If(len(transitions) > 0, g.Text(" | ")),
			refileButton(note.ID, "inbox", "refile"),
		},
	)
}

func refileButton(noteID uuid.UUID, category string, label string) g.Node {
	url := fmt.Sprintf("/refile/%s/%s", noteID, category)
	return h.Button(h.Class("link"), g.Attr("data-on:click", fmt.Sprintf("@post('%s')", url)), g.Text(label))
}

func subfileButton(noteID uuid.UUID, t notesmeta.Transition) g.Node {
	url := fmt.Sprintf("/subfile/%s/%s", noteID, t.Target)
	return h.Button(h.Class("link"), g.Attr("data-on:click", fmt.Sprintf("@post('%s')", url)), g.Text(t.Event))
}
