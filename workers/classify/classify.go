package classify

import (
	"context"
	"fmt"
	"strings"

	"github.com/openai/openai-go"
	"github.com/rcy/evoke"
	"github.com/rcy/whatever/commands"
	"github.com/rcy/whatever/events"
)

type Worker struct {
	cmdSender evoke.CommandSender
}

func NewWorker(cmdSender evoke.CommandSender) *Worker {
	return &Worker{cmdSender: cmdSender}
}

func (w *Worker) Handle(e evoke.Event, replay bool) error {
	evt, ok := e.(events.NoteCreated)
	if !ok {
		return fmt.Errorf("not a NoteCreated event")
	}

	if evt.Category != "inbox" {
		return nil
	}

	go func() {
		category, err := Categorize(evt.Text)
		if err != nil {
			fmt.Println("classify error:", err)
			return
		}

		w.cmdSender.MustSend(commands.SetNoteCategory{
			NoteID:   evt.NoteID,
			Category: category,
			Actor:    "ai",
		})
	}()

	return nil
}

// Categorize calls gpt-4o-mini and returns "task" or "reference".
func Categorize(text string) (string, error) {
	client := openai.NewClient()

	prompt := fmt.Sprintf(
		"Classify the following as 'task' (something to do/action item) or 'reference' (something to remember/record). Reply with only: task or reference\n\nText: %s",
		text,
	)

	msg, err := client.Chat.Completions.New(context.Background(), openai.ChatCompletionNewParams{
		Model: openai.ChatModelGPT4oMini,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(prompt),
		},
	})
	if err != nil {
		return "", err
	}

	category := strings.TrimSpace(msg.Choices[0].Message.Content)
	if category != "task" && category != "reference" {
		return "", fmt.Errorf("unexpected response: %q", category)
	}

	return category, nil
}
