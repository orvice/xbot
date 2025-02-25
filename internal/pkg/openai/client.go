package openai

import (
	"context"

	"butterfly.orx.me/core/log"
	openai "github.com/sashabaranov/go-openai"
	"go.orx.me/xbot/internal/conf"
)

var (
	client        *openai.Client
	pictureClient *openai.Client
)

func Init() error {
	var err error
	client, err = newClient()
	pictureClient, err = newPictureClient()
	return err
}

func newClient() (*openai.Client, error) {
	config := openai.DefaultConfig(conf.Conf.OpenAI.Key)
	config.BaseURL = conf.Conf.OpenAI.Endpoint
	client := openai.NewClientWithConfig(config)
	return client, nil
}

func newPictureClient() (*openai.Client, error) {
	config := openai.DefaultConfig(conf.Conf.PictureVendor.Key)
	config.BaseURL = conf.Conf.PictureVendor.Endpoint
	client := openai.NewClientWithConfig(config)
	return client, nil
}

func ChatCompletion(ctx context.Context, model string, promptString, req string) (string, error) {
	logger := log.FromContext(ctx)

	prompt := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: promptString,
	}
	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: model,
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

func GenImage(ctx context.Context, promptString string) (string, error) {
	logger := log.FromContext(ctx)
	reqBase64 := openai.ImageRequest{
		Prompt:         promptString,
		ResponseFormat: openai.CreateImageResponseFormatB64JSON,
		N:              1,
		Model:          conf.Conf.PictureVendor.Model,
		Size:           openai.CreateImageSize1792x1024,
		Quality:        openai.CreateImageQualityHD,
		Style:          openai.CreateImageStyleNatural,
	}
	resp, err := pictureClient.CreateImage(ctx, reqBase64)
	if err != nil {
		logger.Error("CreateImage error ",
			"error", err)
		return "", err
	}
	return resp.Data[0].URL, nil
}
