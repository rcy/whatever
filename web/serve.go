package web

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/rcy/whatever/app"
	"github.com/rcy/whatever/commands"
	"github.com/rcy/whatever/projections/note"
	"github.com/rcy/whatever/projections/realm"
	"github.com/starfederation/datastar-go/datastar"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

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
			http.Redirect(w, r, "/dsnotes/inbox", http.StatusSeeOther)
		})

		r.Get("/deleted_notes", svc.deletedNotesHandler)
		//r.Get("/events", svc.eventsHandler)

		r.Post("/realm", func(w http.ResponseWriter, r *http.Request) {
			svc.setRealmCookie(w, r, r.FormValue("realm"))
			w.Header().Set("HX-Redirect", "")
		})

		r.Get("/dsnotes/{category}", svc.notesIndex)
		r.Post("/dsnotes", svc.postNotesHandler2)
		r.Post("/refile/{noteID}/{category}", svc.postRefileNote)

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
			})
			r.Post("/{id}/undelete", svc.undeleteNoteHandler)
		})
	})

	return r, nil
}

type signals struct {
	Body         string `json:"body"`
	ViewCategory string `json:"viewCategory"`
}

func (s *webservice) postNotesHandler2(w http.ResponseWriter, r *http.Request) {
	realmID := realmFromRequest(r)

	var signals signals
	err := datastar.ReadSignals(r, &signals)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if signals.Body != "" {
		err := s.app.Commander.Send(commands.CreateNote{NoteID: uuid.New(), RealmID: realmID, Text: signals.Body})
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

	headerEl, err := s.header(r, signals)
	if err != nil {
		fmt.Println("DEBUGX onAk 0", err)

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Println("headerEl", headerEl)
	sse.PatchElementGostar(headerEl)
}

func (s *webservice) header(r *http.Request, signals signals) (g.Node, error) {
	realmID := realmFromRequest(r)

	categoryCounts, err := s.app.Notes.CategoryCounts(realmID)
	if err != nil {
		return nil, err
	}
	realmList, err := s.app.Realms.FindAll()
	if err != nil {
		return nil, err
	}

	return header(realmID, realmList, signals.ViewCategory, categoryCounts), nil
}

func (s *webservice) notesIndex(w http.ResponseWriter, r *http.Request) {
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

	h.HTML(
		h.Head(
			h.Script(h.Type("module"), h.Src("https://cdn.jsdelivr.net/gh/starfederation/datastar@1.0.0-RC.7/bundles/datastar.js")),
			h.StyleEl(g.Raw(styles)),
		),
		h.Body(
			h.Div(g.Attr("data-signals", fmt.Sprintf("{viewCategory: '%s'}", category))),
			h.Div(h.Style("display:flex;flex-direction:column;gap:10px"),
				h.Div(header(realmID, realmList, category, categoryCounts)),
				h.Div(input()),
				h.Div(notes(noteList)),
			),
		),
	).Render(w)
}

func (s *webservice) postRefileNote(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "noteID")
	category := chi.URLParam(r, "category")

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

	err = s.app.Commander.Send(commands.SetNoteCategory{NoteID: noteID, Category: category})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sse := datastar.NewSSE(w, r)

	headerEl, err := s.header(r, signals)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	sse.PatchElementGostar(headerEl)

	note, err := s.app.Notes.FindOne(noteID.String())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	noteEl := noteEl(note)
	fmt.Println("DEBUGX Twfq 1", noteEl)

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

func header(realmID uuid.UUID, realmList []realm.Realm, category string, categoryCounts []note.CategoryCount) g.Node {
	return h.Div(h.ID("header"), h.Style("background: lime; padding: 5px; display:flex; justify-content:space-between"),
		h.Div(h.Style("display:flex; gap:5px"),
			h.Div(h.ID("foobar"), h.Style("font-weight: bold"), g.Text("Not Now")),
			h.Div(h.Style("display: flex; gap: 5px"),
				g.Map(categoryCounts, func(cc note.CategoryCount) g.Node {
					text := fmt.Sprintf("%s %d", g.Text(cc.Category), cc.Count)
					if cc.Category == category {
						return h.Div(h.A(h.Style("background: white"), g.Text(text), h.Href("/dsnotes/"+cc.Category)))
					} else {
						return h.Div(h.A(g.Text(text), h.Href("/dsnotes/"+cc.Category)))
					}
				})),
		),
		h.Div(
			h.Select(g.Attr("hx-post", "/realm"), h.Name("realm"),
				g.Map(realmList, func(realm realm.Realm) g.Node {
					return h.Option(
						h.Value(realm.ID.String()),
						g.Text(realm.Name),
						g.If(realmID == realm.ID, h.Selected()),
					)
				})),
		),
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

func noteEl(note note.Note) g.Node {
	return h.Div(h.ID(noteID(note)),
		h.Div(linkifyNode(note.Status+" "+note.Text)),
		h.Div(h.Style("color: gray; font-size: 70%; line-height:.5em"),
			h.Div(h.Style("display:flex; gap:2px"),
				h.Div(g.Text(note.Category)),
				h.Div(g.Text(ago(note.Ts))),
				h.Div(g.Text("|")),
				refile(note),
			),
		),
	)
}

func refile(note note.Note) g.Node {
	if note.Category == "inbox" {
		return h.Div(h.Style("display:flex; gap:2px"),
			g.Text("move to: "),
			g.Map(categories, func(newCategory string) g.Node {
				return refileButton(note.ID, newCategory, newCategory)
			}))
	}

	return refileButton(note.ID, "inbox", "refile")
}

func refileButton(noteID uuid.UUID, category string, label string) g.Node {
	url := fmt.Sprintf("/refile/%s/%s", noteID, category)
	return h.Button(h.Class("link"), g.Attr("data-on:click", fmt.Sprintf("@post('%s')", url)), g.Text(label))
}
