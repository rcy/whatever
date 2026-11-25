package classify

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/openai/openai-go"
	"github.com/rcy/evoke"
	"github.com/rcy/whatever/commands"
	"github.com/riverqueue/river"
)

// JobArgs are the arguments for a classify River job.
type JobArgs struct {
	NoteID uuid.UUID `json:"note_id"`
	Text   string    `json:"text"`
}

func (JobArgs) Kind() string { return "classify" }

// JobWorker is a River worker that classifies notes using an LLM.
type JobWorker struct {
	river.WorkerDefaults[JobArgs]
	cmdSender evoke.CommandSender
}

func NewJobWorker(cmdSender evoke.CommandSender) *JobWorker {
	return &JobWorker{cmdSender: cmdSender}
}

func (w *JobWorker) Work(ctx context.Context, job *river.Job[JobArgs]) error {
	category, err := Categorize(job.Args.Text)
	if err != nil {
		return err
	}

	return w.cmdSender.Send(commands.SetNoteCategory{
		NoteID:   job.Args.NoteID,
		Category: category,
		Actor:    "ai",
	})
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
