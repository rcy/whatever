package bskysync

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rcy/evoke"
	"github.com/rcy/whatever/catalog/notesmeta"
	"github.com/rcy/whatever/commands"
	"github.com/riverqueue/river"
)

const maxFirstSync = 100

type JobArgs struct {
	UserID string `json:"user_id"`
}

func (JobArgs) Kind() string { return "bsky_sync" }

type JobWorker struct {
	river.WorkerDefaults[JobArgs]
	cmdSender evoke.CommandSender
	dataDir   string
}

func NewJobWorker(cmdSender evoke.CommandSender, dataDir string) *JobWorker {
	return &JobWorker{cmdSender: cmdSender, dataDir: dataDir}
}

type credentials struct {
	Handle      string `json:"handle"`
	AppPassword string `json:"app_password"`
}

type sessionResponse struct {
	AccessJwt string `json:"accessJwt"`
	DID       string `json:"did"`
}

type likedPost struct {
	URI     string
	LikedAt time.Time
}

func (w *JobWorker) Work(ctx context.Context, job *river.Job[JobArgs]) error {
	if job.Args.UserID == "" {
		return w.syncAll(ctx)
	}
	return w.syncUser(ctx, job.Args.UserID)
}

func (w *JobWorker) syncAll(ctx context.Context) error {
	dir := filepath.Join(w.dataDir, "bsky_tokens")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		name := e.Name()
		if filepath.Ext(name) == ".json" {
			userID := strings.TrimSuffix(name, ".json")
			if err := w.syncUser(ctx, userID); err != nil {
				return fmt.Errorf("sync user %s: %w", userID, err)
			}
		}
	}
	return nil
}

func (w *JobWorker) syncUser(ctx context.Context, userID string) error {
	creds, err := loadCredentials(w.dataDir, userID)
	if err != nil {
		return fmt.Errorf("load bsky credentials: %w", err)
	}

	session, err := createSession(ctx, creds)
	if err != nil {
		return fmt.Errorf("bsky auth: %w", err)
	}

	seen, err := loadSeen(w.dataDir, userID)
	if err != nil {
		return fmt.Errorf("load seen: %w", err)
	}
	isFirstSync := len(seen) == 0

	var newPosts []likedPost
	var cursor string

	for {
		likes, nextCursor, err := fetchLikes(ctx, session, creds.Handle, cursor)
		if err != nil {
			return fmt.Errorf("fetch likes: %w", err)
		}

		for _, lp := range likes {
			if seen[lp.URI] {
				if !isFirstSync {
					goto done
				}
				continue
			}
			newPosts = append(newPosts, lp)
			if isFirstSync && len(newPosts) >= maxFirstSync {
				goto done
			}
		}

		if nextCursor == "" {
			break
		}
		cursor = nextCursor
	}

done:
	// Create oldest first so timestamps order correctly
	for i := len(newPosts) - 1; i >= 0; i-- {
		p := newPosts[i]
		noteID := uuid.New()

		// Convert AT URI to web URL
		noteText := atURIToURL(p.URI)

		err := w.cmdSender.Send(commands.CreateNote{
			Owner:       userID,
			NoteID:      noteID,
			Text:        noteText,
			Category:    notesmeta.Note.Slug,
			Subcategory: notesmeta.Note.Inbox().Slug,
			CreatedAt:   p.LikedAt,
		})
		if err != nil {
			return fmt.Errorf("create note for %s: %w", p.URI, err)
		}

		for _, tag := range []string{"bsync", "like"} {
			err = w.cmdSender.Send(commands.AddNoteTag{NoteID: noteID, Tag: tag})
			if err != nil {
				return fmt.Errorf("tag note %s with %s: %w", p.URI, tag, err)
			}
		}

		seen[p.URI] = true
	}

	if len(newPosts) > 0 {
		if err := saveSeen(w.dataDir, userID, seen); err != nil {
			return fmt.Errorf("save seen: %w", err)
		}
	}

	return nil
}

func createSession(ctx context.Context, creds *credentials) (*sessionResponse, error) {
	body := fmt.Sprintf(`{"identifier":%q,"password":%q}`, creds.Handle, creds.AppPassword)
	req, err := http.NewRequestWithContext(ctx, "POST", "https://bsky.social/xrpc/com.atproto.server.createSession", strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("createSession %d: %s", resp.StatusCode, b)
	}

	var session sessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		return nil, err
	}
	return &session, nil
}

type likesResponse struct {
	Feed   []feedItem `json:"feed"`
	Cursor string     `json:"cursor"`
}

type feedItem struct {
	Post struct {
		URI       string `json:"uri"`
		IndexedAt string `json:"indexedAt"`
	} `json:"post"`
}

func fetchLikes(ctx context.Context, session *sessionResponse, actor, cursor string) ([]likedPost, string, error) {
	u, _ := url.Parse("https://bsky.social/xrpc/app.bsky.feed.getActorLikes")
	q := u.Query()
	q.Set("actor", actor)
	q.Set("limit", "100")
	if cursor != "" {
		q.Set("cursor", cursor)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Authorization", "Bearer "+session.AccessJwt)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return nil, "", fmt.Errorf("getActorLikes %d: %s", resp.StatusCode, b)
	}

	var result likesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, "", err
	}

	var posts []likedPost
	for _, item := range result.Feed {
		likedAt, _ := time.Parse(time.RFC3339, item.Post.IndexedAt)
		posts = append(posts, likedPost{
			URI:     item.Post.URI,
			LikedAt: likedAt,
		})
	}

	return posts, result.Cursor, nil
}

func atURIToURL(uri string) string {
	// at://did:plc:xxx/app.bsky.feed.post/yyy -> https://bsky.app/profile/did:plc:xxx/post/yyy
	trimmed := strings.TrimPrefix(uri, "at://")
	parts := strings.Split(trimmed, "/")
	if len(parts) < 3 {
		return uri
	}
	did := parts[0]
	rkey := parts[len(parts)-1]
	return fmt.Sprintf("https://bsky.app/profile/%s/post/%s", did, rkey)
}

func loadCredentials(dataDir, userID string) (*credentials, error) {
	data, err := os.ReadFile(filepath.Join(dataDir, "bsky_tokens", userID+".json"))
	if err != nil {
		return nil, err
	}
	var creds credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, err
	}
	return &creds, nil
}

func seenPath(dataDir, userID string) string {
	return filepath.Join(dataDir, "bsky_sync", userID+".txt")
}

func loadSeen(dataDir, userID string) (map[string]bool, error) {
	data, err := os.ReadFile(seenPath(dataDir, userID))
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]bool{}, nil
		}
		return nil, err
	}
	seen := map[string]bool{}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line != "" {
			seen[line] = true
		}
	}
	return seen, nil
}

func saveSeen(dataDir, userID string, seen map[string]bool) error {
	dir := filepath.Join(dataDir, "bsky_sync")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	var lines []string
	for uri := range seen {
		lines = append(lines, uri)
	}
	return os.WriteFile(seenPath(dataDir, userID), []byte(strings.Join(lines, "\n")+"\n"), 0600)
}
