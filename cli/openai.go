package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/openai/openai-go"
)

// categorize calls gpt-4o-mini and returns "task" or "reference".
func categorize(text string) (string, error) {
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
