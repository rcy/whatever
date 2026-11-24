package web

// A minimal OAuth 2.1 authorization server so MCP clients (claude.ai, Claude
// Code, ...) can access /mcp. Users authenticate with the existing Google
// login; only OWNER_ID may authorize. Clients register dynamically (RFC 7591)
// and must use PKCE (S256).
//
// Everything is stateless: client ids, authorization codes, access tokens and
// refresh tokens are HMAC-signed JSON blobs keyed off SESSION_SECRET. The only
// in-memory state is the set of redeemed authorization codes. Rotating
// SESSION_SECRET revokes every issued token (and logs out web sessions).

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"slices"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/oauthex"
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

const (
	oauthScope      = "notes:read"
	accessTokenTTL  = time.Hour
	refreshTokenTTL = 90 * 24 * time.Hour
	authCodeTTL     = 5 * time.Minute
	consentTTL      = 10 * time.Minute
)

type oauthServer struct {
	key       []byte
	baseURL   string
	ownerID   string
	usedCodes sync.Map // nonce -> expiry time.Time
}

func newOAuthServer(sessionSecret, baseURL, ownerID string) *oauthServer {
	mac := hmac.New(sha256.New, []byte(sessionSecret))
	mac.Write([]byte("whatever oauth v1"))
	return &oauthServer{key: mac.Sum(nil), baseURL: baseURL, ownerID: ownerID}
}

func (o *oauthServer) resourceURL() string { return o.baseURL + "/mcp" }
func (o *oauthServer) resourceMetadataURL() string {
	return o.baseURL + "/.well-known/oauth-protected-resource/mcp"
}

//
// signed tokens
//

type oauthClient struct {
	Name         string   `json:"n,omitempty"`
	RedirectURIs []string `json:"r"`
}

type oauthConsent struct {
	ClientID    string `json:"c"`
	RedirectURI string `json:"r"`
	Challenge   string `json:"p"`
	State       string `json:"s"`
	User        string `json:"u"`
	Exp         int64  `json:"exp"`
}

type oauthCode struct {
	ClientID    string `json:"c"`
	RedirectURI string `json:"r"`
	Challenge   string `json:"p"`
	User        string `json:"u"`
	Nonce       string `json:"x"`
	Exp         int64  `json:"exp"`
}

type oauthGrant struct {
	User     string `json:"u"`
	ClientID string `json:"c,omitempty"`
	Exp      int64  `json:"exp"`
}

func (o *oauthServer) sign(kind string, v any) string {
	body, err := json.Marshal(v)
	if err != nil {
		panic(err) // only marshals our own plain structs
	}
	mac := hmac.New(sha256.New, o.key)
	mac.Write([]byte(kind + "."))
	mac.Write(body)
	return base64.RawURLEncoding.EncodeToString(append(mac.Sum(nil), body...))
}

func (o *oauthServer) parse(kind, token string, v any) error {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(raw) < sha256.Size {
		return errors.New("malformed token")
	}
	sig, body := raw[:sha256.Size], raw[sha256.Size:]
	mac := hmac.New(sha256.New, o.key)
	mac.Write([]byte(kind + "."))
	mac.Write(body)
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return errors.New("bad signature")
	}
	var exp struct {
		Exp int64 `json:"exp"`
	}
	if err := json.Unmarshal(body, &exp); err != nil {
		return err
	}
	if exp.Exp != 0 && time.Now().Unix() > exp.Exp {
		return errors.New("expired")
	}
	return json.Unmarshal(body, v)
}

func randomString() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(buf)
}

//
// discovery
//

func (o *oauthServer) protectedResourceMetadata() http.Handler {
	return auth.ProtectedResourceMetadataHandler(&oauthex.ProtectedResourceMetadata{
		Resource:               o.resourceURL(),
		AuthorizationServers:   []string{o.baseURL},
		ScopesSupported:        []string{oauthScope},
		BearerMethodsSupported: []string{"header"},
		ResourceName:           "whatever notes",
	})
}

func (o *oauthServer) authServerMetadata(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"issuer":                                o.baseURL,
		"authorization_endpoint":                o.baseURL + "/oauth/authorize",
		"token_endpoint":                        o.baseURL + "/oauth/token",
		"registration_endpoint":                 o.baseURL + "/oauth/register",
		"scopes_supported":                      []string{oauthScope},
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
		"code_challenge_methods_supported":      []string{"S256"},
		"token_endpoint_auth_methods_supported": []string{"none"},
	})
}

//
// bearer auth for /mcp
//

func (o *oauthServer) requireToken(next http.Handler) http.Handler {
	verify := func(ctx context.Context, token string, r *http.Request) (*auth.TokenInfo, error) {
		var grant oauthGrant
		if err := o.parse("access", token, &grant); err != nil || grant.User != o.ownerID {
			return nil, auth.ErrInvalidToken
		}
		return &auth.TokenInfo{UserID: grant.User, Scopes: []string{oauthScope}, Expiration: time.Unix(grant.Exp, 0)}, nil
	}
	return auth.RequireBearerToken(verify, &auth.RequireBearerTokenOptions{
		ResourceMetadataURL: o.resourceMetadataURL(),
	})(next)
}

//
// dynamic client registration (RFC 7591)
//

func validRedirectURI(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Fragment != "" || u.Host == "" {
		return false
	}
	switch u.Scheme {
	case "https":
		return true
	case "http":
		host := u.Hostname()
		if host == "localhost" {
			return true
		}
		ip := net.ParseIP(host)
		return ip != nil && ip.IsLoopback()
	}
	return false
}

func (o *oauthServer) register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RedirectURIs []string `json:"redirect_uris"`
		ClientName   string   `json:"client_name"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&req); err != nil {
		oauthError(w, http.StatusBadRequest, "invalid_client_metadata", "invalid json")
		return
	}
	if len(req.RedirectURIs) == 0 {
		oauthError(w, http.StatusBadRequest, "invalid_redirect_uri", "redirect_uris required")
		return
	}
	for _, u := range req.RedirectURIs {
		if !validRedirectURI(u) {
			oauthError(w, http.StatusBadRequest, "invalid_redirect_uri", "redirect uris must be https or http on loopback: "+u)
			return
		}
	}
	if len(req.ClientName) > 100 {
		req.ClientName = req.ClientName[:100]
	}

	clientID := o.sign("client", oauthClient{Name: req.ClientName, RedirectURIs: req.RedirectURIs})
	writeJSON(w, http.StatusCreated, map[string]any{
		"client_id":                  clientID,
		"client_id_issued_at":        time.Now().Unix(),
		"client_name":                req.ClientName,
		"redirect_uris":              req.RedirectURIs,
		"grant_types":                []string{"authorization_code", "refresh_token"},
		"response_types":             []string{"code"},
		"token_endpoint_auth_method": "none",
	})
}

//
// authorization endpoint
//

// authorize shows the consent page. It runs behind authMiddleware so the user
// is logged in with Google.
func (s *webservice) oauthAuthorize(w http.ResponseWriter, r *http.Request) {
	o := s.oauth
	q := r.URL.Query()

	// Errors before the redirect uri is validated must not redirect.
	var client oauthClient
	if err := o.parse("client", q.Get("client_id"), &client); err != nil {
		http.Error(w, "unknown client_id; re-register the client", http.StatusBadRequest)
		return
	}
	redirectURI := q.Get("redirect_uri")
	if !slices.Contains(client.RedirectURIs, redirectURI) {
		http.Error(w, "redirect_uri does not match registration", http.StatusBadRequest)
		return
	}

	state := q.Get("state")
	if q.Get("response_type") != "code" {
		redirectWithError(w, r, redirectURI, state, "unsupported_response_type")
		return
	}
	challenge := q.Get("code_challenge")
	if challenge == "" || q.Get("code_challenge_method") != "S256" {
		redirectWithError(w, r, redirectURI, state, "invalid_request")
		return
	}

	user := getUserInfo(r)
	if user.Id != o.ownerID {
		redirectWithError(w, r, redirectURI, state, "access_denied")
		return
	}

	consent := o.sign("consent", oauthConsent{
		ClientID:    q.Get("client_id"),
		RedirectURI: redirectURI,
		Challenge:   challenge,
		State:       state,
		User:        user.Id,
		Exp:         time.Now().Add(consentTTL).Unix(),
	})

	name := client.Name
	if name == "" {
		name = "An unnamed application"
	}
	redirectHost := redirectURI
	if u, err := url.Parse(redirectURI); err == nil {
		redirectHost = u.Host
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Content-Security-Policy", "frame-ancestors 'none'")
	h.HTML(h.Lang("en"),
		h.Head(
			h.TitleEl(g.Text("Authorize access")),
			h.Meta(h.Name("viewport"), h.Content("width=device-width, initial-scale=1")),
			h.StyleEl(g.Raw(styles)),
		),
		h.Body(h.Main(
			h.H2(g.Text("Authorize access")),
			h.P(h.Strong(g.Text(name)), g.Text(" wants read-only access to your notes.")),
			h.P(h.Small(g.Text("Signed in as "+user.Email+". You will be returned to "), h.Code(g.Text(redirectHost)), g.Text("."))),
			h.Form(h.Method("post"), h.Action("/oauth/authorize"),
				h.Input(h.Type("hidden"), h.Name("consent"), h.Value(consent)),
				h.Button(h.Type("submit"), h.Name("decision"), h.Value("allow"), g.Text("Allow")),
				g.Text(" "),
				h.Button(h.Type("submit"), h.Name("decision"), h.Value("deny"), h.Class("secondary"), g.Text("Deny")),
			),
		)),
	).Render(w)
}

// oauthAuthorizeDecision handles the consent form. It runs behind
// authMiddleware; the session cookie is SameSite=Lax so cross-site posts
// arrive without it.
func (s *webservice) oauthAuthorizeDecision(w http.ResponseWriter, r *http.Request) {
	o := s.oauth
	var c oauthConsent
	if err := o.parse("consent", r.FormValue("consent"), &c); err != nil {
		http.Error(w, "consent expired; start again from your app", http.StatusBadRequest)
		return
	}
	if user := getUserInfo(r); user.Id != c.User || c.User != o.ownerID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if r.FormValue("decision") != "allow" {
		redirectWithError(w, r, c.RedirectURI, c.State, "access_denied")
		return
	}

	code := o.sign("code", oauthCode{
		ClientID:    c.ClientID,
		RedirectURI: c.RedirectURI,
		Challenge:   c.Challenge,
		User:        c.User,
		Nonce:       randomString(),
		Exp:         time.Now().Add(authCodeTTL).Unix(),
	})

	u, _ := url.Parse(c.RedirectURI)
	q := u.Query()
	q.Set("code", code)
	if c.State != "" {
		q.Set("state", c.State)
	}
	q.Set("iss", o.baseURL)
	u.RawQuery = q.Encode()
	http.Redirect(w, r, u.String(), http.StatusFound)
}

func redirectWithError(w http.ResponseWriter, r *http.Request, redirectURI, state, code string) {
	u, _ := url.Parse(redirectURI)
	q := u.Query()
	q.Set("error", code)
	if state != "" {
		q.Set("state", state)
	}
	u.RawQuery = q.Encode()
	http.Redirect(w, r, u.String(), http.StatusFound)
}

//
// token endpoint
//

func (o *oauthServer) token(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		oauthError(w, http.StatusBadRequest, "invalid_request", "bad form")
		return
	}

	var user, clientID string
	switch r.PostForm.Get("grant_type") {
	case "authorization_code":
		var c oauthCode
		if err := o.parse("code", r.PostForm.Get("code"), &c); err != nil {
			oauthError(w, http.StatusBadRequest, "invalid_grant", "invalid or expired code")
			return
		}
		if c.ClientID != r.PostForm.Get("client_id") || c.RedirectURI != r.PostForm.Get("redirect_uri") {
			oauthError(w, http.StatusBadRequest, "invalid_grant", "client_id or redirect_uri mismatch")
			return
		}
		sum := sha256.Sum256([]byte(r.PostForm.Get("code_verifier")))
		if base64.RawURLEncoding.EncodeToString(sum[:]) != c.Challenge {
			oauthError(w, http.StatusBadRequest, "invalid_grant", "pkce verification failed")
			return
		}
		if !o.redeemCode(c.Nonce, c.Exp) {
			oauthError(w, http.StatusBadRequest, "invalid_grant", "code already used")
			return
		}
		user, clientID = c.User, c.ClientID

	case "refresh_token":
		var grant oauthGrant
		if err := o.parse("refresh", r.PostForm.Get("refresh_token"), &grant); err != nil {
			oauthError(w, http.StatusBadRequest, "invalid_grant", "invalid or expired refresh token")
			return
		}
		if id := r.PostForm.Get("client_id"); id != "" && id != grant.ClientID {
			oauthError(w, http.StatusBadRequest, "invalid_grant", "client_id mismatch")
			return
		}
		user, clientID = grant.User, grant.ClientID

	default:
		oauthError(w, http.StatusBadRequest, "unsupported_grant_type", "")
		return
	}

	if user != o.ownerID {
		oauthError(w, http.StatusBadRequest, "invalid_grant", "user not authorized")
		return
	}

	now := time.Now()
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]any{
		"access_token":  o.sign("access", oauthGrant{User: user, Exp: now.Add(accessTokenTTL).Unix()}),
		"token_type":    "Bearer",
		"expires_in":    int(accessTokenTTL.Seconds()),
		"refresh_token": o.sign("refresh", oauthGrant{User: user, ClientID: clientID, Exp: now.Add(refreshTokenTTL).Unix()}),
		"scope":         oauthScope,
	})
}

// redeemCode marks a code nonce as used, returning false if it already was.
func (o *oauthServer) redeemCode(nonce string, exp int64) bool {
	now := time.Now()
	o.usedCodes.Range(func(k, v any) bool {
		if now.After(v.(time.Time)) {
			o.usedCodes.Delete(k)
		}
		return true
	})
	_, used := o.usedCodes.LoadOrStore(nonce, time.Unix(exp, 0))
	return !used
}

//
// helpers
//

// cors allows browser-based MCP clients to reach the public oauth endpoints.
func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Mcp-Protocol-Version, Mcp-Session-Id")
		w.Header().Set("Access-Control-Expose-Headers", "WWW-Authenticate, Mcp-Session-Id")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func oauthError(w http.ResponseWriter, status int, code, desc string) {
	body := map[string]string{"error": code}
	if desc != "" {
		body["error_description"] = desc
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, status, body)
}
