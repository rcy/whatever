package web

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/oauth2"
	googleoauth "google.golang.org/api/oauth2/v2"
	"google.golang.org/api/option"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

const (
	sessionCookieName = "whatever.session"
	stateCookieName   = "whatever.oauthstate"
	nextCookieName    = "whatever.authnext"
)

type contextKey string

const userContextKey contextKey = "user"

type userInfo struct {
	Email   string
	Name    string
	Picture string
}

type sessionPayload struct {
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Picture   string    `json:"picture"`
	ExpiresAt time.Time `json:"expires_at"`
}

type sessionManager struct {
	secret []byte
}

func newSessionManager(secret string) (*sessionManager, error) {
	if strings.TrimSpace(secret) == "" {
		return nil, errors.New("session secret cannot be empty")
	}
	return &sessionManager{secret: []byte(secret)}, nil
}

func (s *sessionManager) issue(w http.ResponseWriter, r *http.Request, user userInfo, secure bool) error {
	payload := sessionPayload{
		Email:     user.Email,
		Name:      user.Name,
		Picture:   user.Picture,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
	token, err := s.sign(payload)
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
		Expires:  payload.ExpiresAt,
	})
	return nil
}

func (s *sessionManager) currentUser(r *http.Request) (userInfo, error) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return userInfo{}, err
	}
	payload, err := s.parse(cookie.Value)
	if err != nil {
		return userInfo{}, err
	}
	return userInfo{
		Email:   payload.Email,
		Name:    payload.Name,
		Picture: payload.Picture,
	}, nil
}

func (s *sessionManager) clear(w http.ResponseWriter, r *http.Request, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
	})
}

func (s *sessionManager) sign(payload sessionPayload) (string, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, s.secret)
	if _, err := mac.Write(body); err != nil {
		return "", err
	}
	sig := mac.Sum(nil)
	token := append(sig, body...)
	return base64.RawURLEncoding.EncodeToString(token), nil
}

func (s *sessionManager) parse(token string) (sessionPayload, error) {
	var payload sessionPayload
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return payload, fmt.Errorf("decode session: %w", err)
	}
	if len(raw) < sha256.Size {
		return payload, errors.New("session token too short")
	}
	sig := raw[:sha256.Size]
	body := raw[sha256.Size:]

	mac := hmac.New(sha256.New, s.secret)
	if _, err := mac.Write(body); err != nil {
		return payload, err
	}
	expected := mac.Sum(nil)
	if !hmac.Equal(sig, expected) {
		return payload, errors.New("invalid session signature")
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return payload, fmt.Errorf("parse session: %w", err)
	}
	if time.Now().After(payload.ExpiresAt) {
		return payload, errors.New("session expired")
	}
	return payload, nil
}

type stateManager struct{}

func (stateManager) issue(w http.ResponseWriter, r *http.Request, secure bool) (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	state := base64.RawURLEncoding.EncodeToString(buf)
	http.SetCookie(w, &http.Cookie{
		Name:     stateCookieName,
		Value:    state,
		Path:     "/auth",
		MaxAge:   600,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
	})
	return state, nil
}

func (stateManager) validate(r *http.Request, provided string) bool {
	cookie, err := r.Cookie(stateCookieName)
	if err != nil || provided == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(provided)) == 1
}

func (stateManager) clear(w http.ResponseWriter, r *http.Request, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     stateCookieName,
		Value:    "",
		Path:     "/auth",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
	})
}

func setNextCookie(w http.ResponseWriter, r *http.Request, next string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     nextCookieName,
		Value:    next,
		Path:     "/auth",
		MaxAge:   600,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
	})
}

func popNextCookie(w http.ResponseWriter, r *http.Request, secure bool) string {
	next := "/"
	if cookie, err := r.Cookie(nextCookieName); err == nil {
		next = sanitizeRedirect(cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     nextCookieName,
		Value:    "",
		Path:     "/auth",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
	})
	return next
}

func sanitizeRedirect(raw string) string {
	if raw == "" {
		return "/"
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "/"
	}
	if u.Scheme != "" || u.Host != "" {
		return "/"
	}
	if !strings.HasPrefix(u.Path, "/") {
		return "/"
	}
	return u.RequestURI()
}

func (s *webservice) secureCookie(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	if proto := r.Header.Get("X-Forwarded-Proto"); strings.EqualFold(proto, "https") {
		return true
	}
	return strings.HasPrefix(strings.ToLower(s.baseURL), "https://")
}

func (s *webservice) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := s.sessions.currentUser(r)
		if err != nil {
			redirect := "/auth"
			if r.URL.Path != "/auth" {
				redirect = "/auth?next=" + url.QueryEscape(r.URL.RequestURI())
			}
			http.Redirect(w, r, redirect, http.StatusSeeOther)
			return
		}
		ctx := context.WithValue(r.Context(), userContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *webservice) authHandler(w http.ResponseWriter, r *http.Request) {
	if _, err := s.sessions.currentUser(r); err == nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	secure := s.secureCookie(r)
	next := sanitizeRedirect(r.URL.Query().Get("next"))
	setNextCookie(w, r, next, secure)

	state, err := s.states.issue(w, r, secure)
	if err != nil {
		http.Error(w, fmt.Sprintf("oauth state: %s", err), http.StatusInternalServerError)
		return
	}

	authURL := s.oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)

	h.HTML(h.Lang("en"),
		h.Head(
			h.TitleEl(g.Text("Whatever NotNow")),
			h.Meta(h.Name("viewport"), h.Content("width=device-width, initial-scale=1")),
			h.Section(
				h.H2(g.Text("Sign in")),
				h.P(g.Text("Continue with Google to access whatever.")),
				h.A(
					h.Class("contrast"),
					h.Href(authURL),
					g.Text("Login with Google"),
				),
			),
		)).Render(w)
}

func (s *webservice) authCallbackHandler(w http.ResponseWriter, r *http.Request) {
	secure := s.secureCookie(r)
	if !s.states.validate(r, r.URL.Query().Get("state")) {
		http.Error(w, "invalid state", http.StatusBadRequest)
		return
	}
	s.states.clear(w, r, secure)

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "missing code", http.StatusBadRequest)
		return
	}

	token, err := s.oauthConfig.Exchange(r.Context(), code)
	if err != nil {
		http.Error(w, fmt.Sprintf("oauth exchange: %s", err), http.StatusBadRequest)
		return
	}
	client := s.oauthConfig.Client(r.Context(), token)

	oauthSvc, err := googleoauth.NewService(r.Context(), option.WithHTTPClient(client))
	if err != nil {
		http.Error(w, fmt.Sprintf("oauth service: %s", err), http.StatusInternalServerError)
		return
	}

	info, err := googleoauth.NewUserinfoService(oauthSvc).Get().Do()
	if err != nil {
		http.Error(w, fmt.Sprintf("userinfo: %s", err), http.StatusBadRequest)
		return
	}

	user := userInfo{
		Email:   info.Email,
		Name:    info.Name,
		Picture: info.Picture,
	}
	if err := s.sessions.issue(w, r, user, secure); err != nil {
		http.Error(w, fmt.Sprintf("issue session: %s", err), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, popNextCookie(w, r, secure), http.StatusSeeOther)
}

func (s *webservice) logoutHandler(w http.ResponseWriter, r *http.Request) {
	secure := s.secureCookie(r)
	s.sessions.clear(w, r, secure)
	s.states.clear(w, r, secure)
	http.Redirect(w, r, "/auth", http.StatusSeeOther)
}
