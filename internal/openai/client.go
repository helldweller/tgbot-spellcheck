package openai

import (
	"context"
	"errors"
	"time"

	openai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

const (
	requestTimeout = 15 * time.Second
	maxRetries     = 3
)

type Client interface {
	CorrectText(ctx context.Context, text string) (string, error)
}

type client struct {
	api   openai.Client
	model string
}

func NewClient(apiKey, model string) Client {
	return &client{
		api: openai.NewClient(
			option.WithAPIKey(apiKey),
			option.WithRequestTimeout(requestTimeout),
			option.WithMaxRetries(maxRetries),
		),
		model: model,
	}
}

func (c *client) CorrectText(ctx context.Context, text string) (string, error) {
	prompt := buildPrompt(text)

	resp, err := c.api.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.DeveloperMessage("Ты — русскоязычный корректор. Исправь только орфографию, пунктуацию и грамматику, сохранив стиль автора. Отвечай только откорректированным текстом без пояснений."),
			openai.UserMessage(prompt),
		},
		Model: openai.ChatModel(c.model),
	})
	if err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", errors.New("openai: empty completion choices")
	}

	return resp.Choices[0].Message.Content, nil
}

func buildPrompt(text string) string {
	return "Проверь и исправь только орфографию и грамматику следующего текста. Сохрани стиль и форматирование. Верни только исправленный текст без комментариев:\n\n" + text
}
