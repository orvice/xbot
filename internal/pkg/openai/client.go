package openai

import (
	"context"

	"butterfly.orx.me/core/log"
	openai "github.com/sashabaranov/go-openai"
	"go.orx.me/xbot/internal/conf"
)

var (
	client *openai.Client
)

func Init() error {
	var err error
	client, err = newClient()
	return err
}

func newClient() (*openai.Client, error) {
	config := openai.DefaultConfig(conf.Conf.OpenAI.Key)
	config.BaseURL = conf.Conf.OpenAI.Endpoint
	client := openai.NewClientWithConfig(config)
	return client, nil

}

func ChatCompletion(ctx context.Context, promptString, req string) (string, error) {
	logger := log.FromContext(ctx)

	prompt := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: promptString,
	}
	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: conf.Conf.OpenAI.Model,
			Messages: []openai.ChatCompletionMessage{
				prompt,
				{
					Role:    openai.ChatMessageRoleUser,
					Content: req,
				},
			},
		},
	)
	if err != nil {
		logger.Error("ChatCompletion error ",
			"error", err)
		return "", err
	}
	return resp.Choices[0].Message.Content, nil
}
