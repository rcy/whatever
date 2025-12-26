package web

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/hako/durafmt"
	"github.com/rcy/whatever/app"
	"github.com/rcy/whatever/projections/note"
	"github.com/rcy/whatever/projections/realm"
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
			http.Redirect(w, r, "/dsnotes", http.StatusSeeOther)
		})

		r.Get("/deleted_notes", svc.deletedNotesHandler)
		//r.Get("/events", svc.eventsHandler)

		r.Post("/realm", func(w http.ResponseWriter, r *http.Request) {
			svc.setRealmCookie(w, r, r.FormValue("realm"))
			w.Header().Set("HX-Redirect", "")
		})

		r.Get("/dsnotes/{category}", svc.notesIndex)

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
			h.StyleEl(g.Raw(styles)),
		),
		h.Body(
			h.Div(h.Style("display:flex;flex-direction:column;gap:10px"),
				h.Div(header(realmID, realmList, category, categoryCounts)),
				h.Div(notes(noteList)),
			),
		),
	).Render(w)
}

func header(realmID uuid.UUID, realmList []realm.Realm, category string, categoryCounts []note.CategoryCount) g.Node {
	return h.Div(h.Style("background: lime; padding: 5px; display:flex; justify-content:space-between"),
		h.Div(h.Style("display:flex; gap:5px"),
			h.Div(h.Style("font-weight: bold"), g.Text("Not Now")),
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
	var counter int

	return h.Table(
		// g.Attr("cellpadding", "0"),
		// g.Attr("cellspacing", "0"),
		//g.Attr("border", "0"),
		h.TBody(
			g.Map(noteList, func(note note.Note) g.Node {
				counter += 1
				return g.Group{
					h.Tr(
						h.Td(g.Attr("valign", "top"), g.Text(fmt.Sprintf("%d. ", counter))),
						h.Td(linkifyNode(note.Status+" "+note.Text)),
					),
					h.Tr(h.Style("color: gray; font-size: 80%; line-height:.5em"),
						h.Td(h.ColSpan("1")),
						h.Td(g.Attr("valign", "top"), g.Text(ago(note.Ts))),
					),
					h.Tr(h.Style("height: 10px")),
				}
			}),
		))
}

func ago(ts time.Time) string {
	return durafmt.Parse(time.Since(ts)).LimitFirstN(1).String() + " ago"
}
