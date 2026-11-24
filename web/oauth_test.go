package web

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/rcy/whatever/app"
	googleoauth "google.golang.org/api/oauth2/v2"
)

const testSecret = "0123456789abcdef0123456789abcdef"

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	a, err := app.New(filepath.Join(t.TempDir(), "events.db"))
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewUnstartedServer(nil)
	mux, err := Server(a, Config{
		BaseURL:            "http://" + ts.Listener.Addr().String(),
		GoogleClientID:     "x",
		GoogleClientSecret: "x",
		SessionSecret:      testSecret,
		WebhookToken:       "wh",
		OwnerID:            "owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	ts.Config.Handler = mux
	ts.Start()
	t.Cleanup(ts.Close)
	return ts
}

func sessionCookie(t *testing.T, userID string) *http.Cookie {
	t.Helper()
	sm, _ := newSessionManager(testSecret)
	rec := httptest.NewRecorder()
	if err := sm.issue(rec, httptest.NewRequest("GET", "/", nil), googleoauth.Userinfo{Id: userID, Email: userID + "@example.com"}, false); err != nil {
		t.Fatal(err)
	}
	return rec.Result().Cookies()[0]
}

var noRedirect = &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}

func decode(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	defer resp.Body.Close()
	var m map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		t.Fatal(err)
	}
	return m
}

func authorizeURL(base, clientID, redirectURI, challenge string) string {
	q := url.Values{
		"response_type":         {"code"},
		"client_id":             {clientID},
		"redirect_uri":          {redirectURI},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
		"state":                 {"xyz"},
	}
	return base + "/oauth/authorize?" + q.Encode()
}

func TestOAuthFlow(t *testing.T) {
	ts := newTestServer(t)
	base := ts.URL

	// unauthenticated MCP request points at resource metadata
	resp, _ := http.Post(base+"/mcp", "application/json", strings.NewReader(`{}`))
	if resp.StatusCode != 401 || !strings.Contains(resp.Header.Get("WWW-Authenticate"), "resource_metadata=") {
		t.Fatalf("mcp unauth: %d %q", resp.StatusCode, resp.Header.Get("WWW-Authenticate"))
	}

	resp, _ = http.Get(base + "/.well-known/oauth-protected-resource/mcp")
	if prm := decode(t, resp); prm["authorization_servers"].([]any)[0] != base {
		t.Fatalf("prm: %v", prm)
	}
	resp, _ = http.Get(base + "/.well-known/oauth-authorization-server")
	if asm := decode(t, resp); asm["registration_endpoint"] != base+"/oauth/register" {
		t.Fatalf("asm: %v", asm)
	}

	// register
	resp, _ = http.Post(base+"/oauth/register", "application/json", strings.NewReader(`{"redirect_uri":"x","redirect_uris":["http://evil.com/cb"]}`))
	if resp.StatusCode != 400 {
		t.Fatalf("non-loopback http redirect accepted: %d", resp.StatusCode)
	}
	redirectURI := "http://localhost:5555/callback"
	resp, _ = http.Post(base+"/oauth/register", "application/json", strings.NewReader(`{"client_name":"Test","redirect_uris":["`+redirectURI+`"]}`))
	reg := decode(t, resp)
	clientID, _ := reg["client_id"].(string)
	if clientID == "" {
		t.Fatalf("register: %v", reg)
	}

	verifier := "a-very-long-random-code-verifier-string-0123456789"
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])
	authURL := authorizeURL(base, clientID, redirectURI, challenge)

	// not logged in -> google login
	resp, _ = noRedirect.Get(authURL)
	if resp.StatusCode != 303 || !strings.HasPrefix(resp.Header.Get("Location"), "/auth?next=") {
		t.Fatalf("authorize no session: %d %s", resp.StatusCode, resp.Header.Get("Location"))
	}

	// wrong redirect uri is rejected without redirecting
	req, _ := http.NewRequest("GET", authorizeURL(base, clientID, "http://localhost:6666/other", challenge), nil)
	req.AddCookie(sessionCookie(t, "owner"))
	resp, _ = noRedirect.Do(req)
	if resp.StatusCode != 400 {
		t.Fatalf("mismatched redirect: %d", resp.StatusCode)
	}

	// non-owner is denied
	req, _ = http.NewRequest("GET", authURL, nil)
	req.AddCookie(sessionCookie(t, "stranger"))
	resp, _ = noRedirect.Do(req)
	if loc := resp.Header.Get("Location"); !strings.Contains(loc, "error=access_denied") {
		t.Fatalf("non-owner: %d %s", resp.StatusCode, loc)
	}

	// owner sees consent page
	owner := sessionCookie(t, "owner")
	req, _ = http.NewRequest("GET", authURL, nil)
	req.AddCookie(owner)
	resp, _ = noRedirect.Do(req)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	m := regexp.MustCompile(`name="consent" value="([^"]+)"`).FindSubmatch(body)
	if resp.StatusCode != 200 || m == nil {
		t.Fatalf("consent page: %d %s", resp.StatusCode, body)
	}

	// approve
	req, _ = http.NewRequest("POST", base+"/oauth/authorize", strings.NewReader(url.Values{"consent": {string(m[1])}, "decision": {"allow"}}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(owner)
	resp, _ = noRedirect.Do(req)
	loc, _ := url.Parse(resp.Header.Get("Location"))
	code := loc.Query().Get("code")
	if resp.StatusCode != 302 || code == "" || loc.Query().Get("state") != "xyz" || loc.Host != "localhost:5555" {
		t.Fatalf("approve: %d %s", resp.StatusCode, loc)
	}

	tokenReq := func(form url.Values) (int, map[string]any) {
		resp, _ := http.PostForm(base+"/oauth/token", form)
		return resp.StatusCode, decode(t, resp)
	}
	codeForm := func(v string) url.Values {
		return url.Values{"grant_type": {"authorization_code"}, "code": {code}, "client_id": {clientID}, "redirect_uri": {redirectURI}, "code_verifier": {v}}
	}

	if status, _ := tokenReq(codeForm("wrong")); status != 400 {
		t.Fatalf("bad verifier accepted")
	}
	status, tok := tokenReq(codeForm(verifier))
	if status != 200 || tok["access_token"] == nil {
		t.Fatalf("token: %d %v", status, tok)
	}
	if status, _ := tokenReq(codeForm(verifier)); status != 400 {
		t.Fatalf("code reuse accepted")
	}

	callMCP := func(access string) int {
		req, _ := http.NewRequest("POST", base+"/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_notes","arguments":{}}}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		req.Header.Set("Authorization", "Bearer "+access)
		resp, _ := http.DefaultClient.Do(req)
		resp.Body.Close()
		return resp.StatusCode
	}
	if s := callMCP(tok["access_token"].(string)); s != 200 {
		t.Fatalf("mcp with token: %d", s)
	}
	if s := callMCP(tok["refresh_token"].(string)); s != 401 {
		t.Fatalf("refresh token accepted as access token: %d", s)
	}

	status, tok2 := tokenReq(url.Values{"grant_type": {"refresh_token"}, "refresh_token": {tok["refresh_token"].(string)}, "client_id": {clientID}})
	if status != 200 || callMCP(tok2["access_token"].(string)) != 200 {
		t.Fatalf("refresh: %d %v", status, tok2)
	}
}
