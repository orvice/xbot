package gemini

import (
	"context"
	"fmt"

	"butterfly.orx.me/core/log"
	"go.orx.me/xbot/internal/conf"
	"google.golang.org/genai"
)

var (
	client *Client
)

func GetClient() *Client {
	return client
}

func Init() error {
	ctx := context.Background()
	sdkClient, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  conf.Conf.PictureVendor.Key,
		Backend: genai.BackendVertexAI,
	})
	if err != nil {
		return fmt.Errorf("failed to create Gemini client: %w", err)
	}

	// If a custom endpoint is configured, set it
	if conf.Conf.PictureVendor.Endpoint != "" {
		genai.SetDefaultBaseURLs(genai.BaseURLParameters{
			GeminiURL: conf.Conf.PictureVendor.Endpoint,
			VertexURL: conf.Conf.PictureVendor.Endpoint,
		})
	}

	client = &Client{
		sdkClient: sdkClient,
	}
	return nil
}

// Client wraps the Gemini SDK client
type Client struct {
	sdkClient *genai.Client
}

// GenerateImageRequest represents parameters for image generation
type GenerateImageRequest struct {
	Prompt      string
	Temperature float64
	TopK        int
	TopP        float64
}

// GenerateImageResponse represents the result of image generation
type GenerateImageResponse struct {
	ImageData    string // Base64 encoded image
	MimeType     string
	FinishReason string
}

// GenerateImage generates an image using the Gemini image preview model
func (c *Client) GenerateImage(ctx context.Context, req GenerateImageRequest) (*GenerateImageResponse, error) {
	logger := log.FromContext(ctx).With("method", "GenerateImage")

	const maxRetries = 3
	var lastErr error

	// Set default values if not provided
	temperature := req.Temperature
	if temperature == 0 {
		temperature = 0.7
	}
	topK := float32(req.TopK)
	if topK == 0 {
		topK = 40
	}
	topP := req.TopP
	if topP == 0 {
		topP = 0.95
	}

	// Build content parts
	parts := []*genai.Part{
		{Text: req.Prompt},
	}
	contents := []*genai.Content{
		{Parts: parts},
	}

	// Configure generation parameters
	config := &genai.GenerateContentConfig{
		Temperature:    genai.Ptr(float32(temperature)),
		TopK:           genai.Ptr(topK),
		TopP:           genai.Ptr(float32(topP)),
		CandidateCount: 1,
	}

	// Retry loop
	for attempt := 1; attempt <= maxRetries; attempt++ {
		// Call the Gemini API
		resp, err := c.sdkClient.Models.GenerateContent(
			ctx,
			"gemini-3-pro-image-preview",
			contents,
			config,
		)
		if err != nil {
			lastErr = err
			logger.Error("GenerateContent failed",
				"error", err,
				"attempt", attempt,
				"maxRetries", maxRetries)
			if attempt < maxRetries {
				continue
			}
			return nil, fmt.Errorf("failed to generate content after %d attempts: %w", maxRetries, lastErr)
		}

		// Validate response
		if len(resp.Candidates) == 0 {
			lastErr = fmt.Errorf("no candidates in response")
			logger.Error("No candidates in response",
				"attempt", attempt,
				"maxRetries", maxRetries)
			if attempt < maxRetries {
				continue
			}
			return nil, lastErr
		}

		candidate := resp.Candidates[0]
		if candidate.Content == nil || len(candidate.Content.Parts) == 0 {
			lastErr = fmt.Errorf("no parts in candidate content")
			logger.Error("No parts in candidate content",
				"attempt", attempt,
				"maxRetries", maxRetries)
			if attempt < maxRetries {
				continue
			}
			return nil, lastErr
		}

		// Extract image data
		var imageData *genai.Blob
		for _, part := range candidate.Content.Parts {
			if part.InlineData != nil {
				imageData = part.InlineData
				break
			}
		}

		if imageData == nil {
			lastErr = fmt.Errorf("no image data found in response")
			logger.Error("No image data found in response",
				"attempt", attempt,
				"maxRetries", maxRetries)
			if attempt < maxRetries {
				continue
			}
			return nil, lastErr
		}

		logger.Info("Image generation successful",
			"mime_type", imageData.MIMEType,
			"finish_reason", candidate.FinishReason,
			"data_length", len(imageData.Data),
			"attempt", attempt)

		return &GenerateImageResponse{
			ImageData:    string(imageData.Data),
			MimeType:     imageData.MIMEType,
			FinishReason: string(candidate.FinishReason),
		}, nil
	}

	return nil, fmt.Errorf("failed to generate image after %d attempts: %w", maxRetries, lastErr)
}
