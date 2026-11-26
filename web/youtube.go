package web

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"log"
	"net/http"

	"github.com/rcy/whatever/workers/youtubesync"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const youtubeStateCookie = "whatever.ytstate"

func (s *webservice) youtubeOAuthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     s.oauthConfig.ClientID,
		ClientSecret: s.oauthConfig.ClientSecret,
		RedirectURL:  s.baseURL + "/youtube/callback",
		Scopes:       []string{"https://www.googleapis.com/auth/youtube.readonly"},
		Endpoint:     google.Endpoint,
	}
}

func (s *webservice) youtubeConnect(w http.ResponseWriter, r *http.Request) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	state := base64.RawURLEncoding.EncodeToString(buf)
	http.SetCookie(w, &http.Cookie{
		Name:     youtubeStateCookie,
		Value:    state,
		Path:     "/youtube",
		MaxAge:   600,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.secureCookie(r),
	})
	url := s.youtubeOAuthConfig().AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	http.Redirect(w, r, url, http.StatusSeeOther)
}

func (s *webservice) youtubeCallback(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(youtubeStateCookie)
	if err != nil || r.URL.Query().Get("state") == "" ||
		subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(r.URL.Query().Get("state"))) != 1 {
		http.Error(w, "invalid state", http.StatusBadRequest)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:   youtubeStateCookie,
		Value:  "",
		Path:   "/youtube",
		MaxAge: -1,
	})

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "missing code", http.StatusBadRequest)
		return
	}

	token, err := s.youtubeOAuthConfig().Exchange(r.Context(), code)
	if err != nil {
		http.Error(w, "oauth exchange failed", http.StatusBadRequest)
		return
	}

	userID := getUserInfo(r).Id
	if err := s.app.SaveYouTubeToken(userID, token); err != nil {
		log.Printf("save youtube token: %v", err)
		http.Error(w, "failed to save token", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/settings", http.StatusSeeOther)
}

func (s *webservice) youtubeSync(w http.ResponseWriter, r *http.Request) {
	userID := getUserInfo(r).Id
	err := s.app.River.Insert(r.Context(), youtubesync.JobArgs{UserID: userID}, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/settings", http.StatusSeeOther)
}
