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

func ChatCompletionWithModels(ctx context.Context, models []string, promptString, req string) (string, string, error) {
	logger := log.FromContext(ctx).With("method", "ChatCompletionWithModels")
	prompt := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: promptString,
	}

	// Try each model in the provided order until one succeeds
	var lastError error
	for _, currentModel := range models {
		logger.Info("Attempting to use model", "model", currentModel)

		// Try to call the API with current model
		resp, err := client.CreateChatCompletion(
			ctx, // Use the passed context instead of creating a new one
			openai.ChatCompletionRequest{
				Model: currentModel,
				Messages: []openai.ChatCompletionMessage{
					prompt,
					{
						Role:    openai.ChatMessageRoleUser,
						Content: req,
					},
				},
			},
		)

		// If successful, return the result
		if err == nil {
			logger.Info("Successfully used model",
				"model", currentModel,
				"responseLength", len(resp.Choices[0].Message.Content),
			)
			return resp.Choices[0].Message.Content, currentModel, nil
		}

		// Log the error and continue to next model
		lastError = err
		logger.Error("ChatCompletion failed with model",
			"model", currentModel,
			"error", err)
	}

	// If we've tried all models and all failed, return the last error
	logger.Error("All models failed in ChatCompletionWithModels",
		"attemptedModels", models,
		"lastError", lastError)

	return "", "", lastError
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
		// Size:           openai.CreateImageSize1792x1024,
		// Quality:        openai.CreateImageQualityHD,
		// Style: openai.CreateImageStyleNatural,
	}
	resp, err := pictureClient.CreateImage(ctx, reqBase64)
	if err != nil {
		logger.Error("CreateImage error ",
			"error", err)
		return "", err
	}
	logger.Info("CreateImage success",
		"data.url.len", len(resp.Data[0].URL))
	return resp.Data[0].B64JSON, nil
}
