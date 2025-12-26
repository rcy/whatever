package web

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rcy/whatever/app"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
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
			http.Redirect(w, r, "/notes", http.StatusSeeOther)
		})

		r.Get("/deleted_notes", svc.deletedNotesHandler)
		//r.Get("/events", svc.eventsHandler)

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
			})
			r.Post("/{id}/undelete", svc.undeleteNoteHandler)
		})
	})

	return r, nil
}
