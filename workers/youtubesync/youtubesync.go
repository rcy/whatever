package youtubesync

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rcy/evoke"
	"github.com/rcy/whatever/catalog/notesmeta"
	"github.com/rcy/whatever/commands"
	"github.com/riverqueue/river"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	youtube "google.golang.org/api/youtube/v3"
)

const maxFirstSync = 100

type JobArgs struct {
	UserID string `json:"user_id"`
}

func (JobArgs) Kind() string { return "youtube_sync" }

type JobWorker struct {
	river.WorkerDefaults[JobArgs]
	cmdSender   evoke.CommandSender
	oauthConfig *oauth2.Config
	dataDir     string
}

func NewJobWorker(cmdSender evoke.CommandSender, oauthConfig *oauth2.Config, dataDir string) *JobWorker {
	return &JobWorker{
		cmdSender:   cmdSender,
		oauthConfig: oauthConfig,
		dataDir:     dataDir,
	}
}

type likedVideo struct {
	VideoID string
	LikedAt time.Time
}

func (w *JobWorker) Work(ctx context.Context, job *river.Job[JobArgs]) error {
	if job.Args.UserID == "" {
		return w.syncAll(ctx)
	}
	return w.syncUser(ctx, job.Args.UserID)
}

func (w *JobWorker) syncAll(ctx context.Context) error {
	dir := filepath.Join(w.dataDir, "youtube_tokens")
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
	token, err := loadToken(w.dataDir, userID)
	if err != nil {
		return fmt.Errorf("load youtube token: %w", err)
	}

	tokenSource := w.oauthConfig.TokenSource(ctx, token)
	svc, err := youtube.NewService(ctx, option.WithTokenSource(tokenSource))
	if err != nil {
		return fmt.Errorf("youtube service: %w", err)
	}

	seen, err := loadSeen(w.dataDir, userID)
	if err != nil {
		return fmt.Errorf("load seen: %w", err)
	}
	isFirstSync := len(seen) == 0

	var newVideos []likedVideo
	var pageToken string

	for {
		call := svc.PlaylistItems.List([]string{"snippet"}).PlaylistId("LL").MaxResults(50)
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		resp, err := call.Context(ctx).Do()
		if err != nil {
			return fmt.Errorf("youtube playlist items: %w", err)
		}

		for _, item := range resp.Items {
			videoID := item.Snippet.ResourceId.VideoId
			if seen[videoID] {
				if !isFirstSync {
					goto done
				}
				continue
			}

			likedAt, _ := time.Parse(time.RFC3339, item.Snippet.PublishedAt)

			newVideos = append(newVideos, likedVideo{
				VideoID: videoID,
				LikedAt: likedAt,
			})
			if isFirstSync && len(newVideos) >= maxFirstSync {
				goto done
			}
		}

		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}

done:
	// Create notes oldest-first so timestamps order correctly
	for i := len(newVideos) - 1; i >= 0; i-- {
		v := newVideos[i]
		noteID := uuid.New()
		url := "https://www.youtube.com/watch?v=" + v.VideoID

		err := w.cmdSender.Send(commands.CreateNote{
			Owner:       userID,
			NoteID:      noteID,
			Text:        url,
			Category:    notesmeta.Note.Slug,
			Subcategory: notesmeta.Note.Inbox().Slug,
			CreatedAt:   v.LikedAt,
		})
		if err != nil {
			return fmt.Errorf("create note for %s: %w", v.VideoID, err)
		}

		for _, tag := range []string{"ytsync", "like"} {
			err = w.cmdSender.Send(commands.AddNoteTag{NoteID: noteID, Tag: tag})
			if err != nil {
				return fmt.Errorf("tag note %s with %s: %w", v.VideoID, tag, err)
			}
		}

		seen[v.VideoID] = true
	}

	if len(newVideos) > 0 {
		if err := saveSeen(w.dataDir, userID, seen); err != nil {
			return fmt.Errorf("save seen: %w", err)
		}
	}

	// Save potentially refreshed token
	newToken, err := tokenSource.Token()
	if err == nil && newToken.AccessToken != token.AccessToken {
		saveToken(w.dataDir, userID, newToken)
	}

	return nil
}

func tokenPath(dataDir, userID string) string {
	return filepath.Join(dataDir, "youtube_tokens", userID+".json")
}

func loadToken(dataDir, userID string) (*oauth2.Token, error) {
	data, err := os.ReadFile(tokenPath(dataDir, userID))
	if err != nil {
		return nil, err
	}
	var token oauth2.Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, err
	}
	return &token, nil
}

func saveToken(dataDir, userID string, token *oauth2.Token) error {
	data, err := json.Marshal(token)
	if err != nil {
		return err
	}
	return os.WriteFile(tokenPath(dataDir, userID), data, 0600)
}

func seenPath(dataDir, userID string) string {
	return filepath.Join(dataDir, "youtube_sync", userID+".txt")
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
	dir := filepath.Join(dataDir, "youtube_sync")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	var lines []string
	for id := range seen {
		lines = append(lines, id)
	}
	return os.WriteFile(seenPath(dataDir, userID), []byte(strings.Join(lines, "\n")+"\n"), 0600)
}
