package summarize

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/rcy/evoke"
	"github.com/rcy/whatever/commands"
	"github.com/rcy/whatever/events"
)

type Worker struct {
	cmdSender evoke.CommandSender
	client    openai.Client
	enabled   bool
}

func NewWorker(cmdSender evoke.CommandSender) *Worker {
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		fmt.Println("summarize: DEEPSEEK_API_KEY not set, summarization disabled")
		return &Worker{cmdSender: cmdSender, enabled: false}
	}

	c := openai.NewClient(
		option.WithBaseURL("https://api.deepseek.com/v1"),
		option.WithAPIKey(apiKey),
	)
	return &Worker{cmdSender: cmdSender, client: c, enabled: true}
}

func (w *Worker) Handle(e evoke.Event, replay bool) error {
	if !w.enabled {
		return nil
	}

	evt, ok := e.(events.NoteSummaryRequested)
	if !ok {
		return fmt.Errorf("not a NoteSummaryRequested event")
	}

	go func() {
		fmt.Println("summarize: starting...", evt.NoteID)
		summary, err := w.summarizeURL(evt.Text)
		if err != nil {
			fmt.Println("summarize: error:", err)
			return
		}
		if summary == "" {
			return
		}

		w.cmdSender.MustSend(commands.SummarizeNote{
			NoteID:  evt.NoteID,
			Summary: summary,
		})
		fmt.Println("summarize: done", evt.NoteID)
	}()

	return nil
}

func (w *Worker) summarizeURL(rawURL string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if !strings.HasPrefix(rawURL, "http") {
		return "", nil
	}

	content, err := fetchContent(rawURL)
	if err != nil {
		return "", fmt.Errorf("fetch: %w", err)
	}

	text := stripHTML(content)
	text = strings.TrimSpace(text)
	if len(text) > 8000 {
		text = text[:8000]
	}
	if len(text) < 50 {
		return "", nil
	}

	return w.summarize(text)
}

func (w *Worker) summarize(text string) (string, error) {
	prompt := fmt.Sprintf(
		"Summarize the following content in one sentence. Focus on the key point.\n\nContent:\n%s",
		text,
	)

	msg, err := w.client.Chat.Completions.New(context.Background(), openai.ChatCompletionNewParams{
		Model: openai.ChatModel("deepseek-chat"),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(prompt),
		},
	})
	if err != nil {
		return "", fmt.Errorf("deepseek: %w", err)
	}

	return strings.TrimSpace(msg.Choices[0].Message.Content), nil
}

var htmlTagRe = regexp.MustCompile(`<[^>]*>`)
var whitespaceRe = regexp.MustCompile(`\s+`)

func stripHTML(s string) string {
	s = htmlTagRe.ReplaceAllString(s, " ")
	s = whitespaceRe.ReplaceAllString(s, " ")
	return s
}

func fetchContent(rawURL string) (string, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "whatever/0.1 (note-summarizer; https://github.com/rcy/whatever)")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 100_000))
	if err != nil {
		return "", err
	}
	return string(body), nil
}
