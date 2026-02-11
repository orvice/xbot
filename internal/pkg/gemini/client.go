package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"butterfly.orx.me/core/log"
	"go.orx.me/xbot/internal/conf"
)

var (
	client *Client
)

func GetClient() *Client {
	return client
}

func Init() error {
	client = NewClient(conf.Conf.PictureVendor.Key, conf.Conf.PictureVendor.Endpoint)
	return nil
}

// Client represents a Gemini API client
type Client struct {
	APIKey     string
	BaseURL    string
	HTTPClient *http.Client
}

// NewClient creates a new Gemini API client
func NewClient(apiKey, baseURL string) *Client {
	return &Client{
		APIKey:     apiKey,
		BaseURL:    baseURL,
		HTTPClient: &http.Client{},
	}
}

// Part represents a content part in the request
type Part struct {
	Text string `json:"text,omitempty"`
}

// Content represents the content structure
type Content struct {
	Parts []Part `json:"parts"`
	Role  string `json:"role,omitempty"`
}

// GenerationConfig represents configuration for content generation
type GenerationConfig struct {
	Temperature     float64 `json:"temperature,omitempty"`
	TopK            int     `json:"topK,omitempty"`
	TopP            float64 `json:"topP,omitempty"`
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
	CandidateCount  int     `json:"candidateCount,omitempty"`
}

// GenerateContentRequest represents the request to generate content
type GenerateContentRequest struct {
	Contents         []Content        `json:"contents"`
	GenerationConfig GenerationConfig `json:"generationConfig,omitempty"`
}

// ImageData represents image data in the response
type ImageData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"` // Base64 encoded image data
}

// ResponsePart represents a part in the response
type ResponsePart struct {
	Text       string     `json:"text,omitempty"`
	InlineData *ImageData `json:"inlineData,omitempty"`
}

// ResponseContent represents content in the response
type ResponseContent struct {
	Parts []ResponsePart `json:"parts"`
	Role  string         `json:"role,omitempty"`
}

// Candidate represents a generation candidate
type Candidate struct {
	Content       ResponseContent `json:"content"`
	FinishReason  string          `json:"finishReason,omitempty"`
	SafetyRatings []SafetyRating  `json:"safetyRatings,omitempty"`
}

// SafetyRating represents safety rating information
type SafetyRating struct {
	Category    string `json:"category"`
	Probability string `json:"probability"`
}

// GenerateContentResponse represents the response from generate content API
type GenerateContentResponse struct {
	Candidates []Candidate `json:"candidates"`
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

	// Set default values if not provided
	if req.Temperature == 0 {
		req.Temperature = 0.7
	}
	if req.TopK == 0 {
		req.TopK = 40
	}
	if req.TopP == 0 {
		req.TopP = 0.95
	}

	// Build the API request
	apiReq := GenerateContentRequest{
		Contents: []Content{
			{
				Parts: []Part{
					{Text: req.Prompt},
				},
			},
		},
		GenerationConfig: GenerationConfig{
			Temperature:    req.Temperature,
			TopK:           req.TopK,
			TopP:           req.TopP,
			CandidateCount: 1,
		},
	}

	// Marshal request to JSON
	jsonData, err := json.Marshal(apiReq)
	if err != nil {
		logger.Error("Failed to marshal request", "error", err)
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Build the URL
	url := fmt.Sprintf("%s/v1beta/models/gemini-3-pro-image-preview:generateContent?key=%s",
		c.BaseURL, c.APIKey)

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		logger.Error("Failed to create HTTP request", "error", err)
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		logger.Error("Failed to send HTTP request", "error", err)
		return nil, fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("Failed to read response body", "error", err)
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		logger.Error("HTTP request failed", "status", resp.StatusCode, "body", string(body))
		return nil, fmt.Errorf("HTTP request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var apiResp GenerateContentResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		logger.Error("Failed to unmarshal response", "error", err, "body", string(body))
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Validate response
	if len(apiResp.Candidates) == 0 {
		logger.Error("No candidates in response")
		return nil, fmt.Errorf("no candidates in response")
	}

	candidate := apiResp.Candidates[0]
	if len(candidate.Content.Parts) == 0 {
		logger.Error("No parts in candidate content")
		return nil, fmt.Errorf("no parts in candidate content")
	}

	// Extract image data
	var imageData *ImageData
	for _, part := range candidate.Content.Parts {
		if part.InlineData != nil {
			imageData = part.InlineData
			break
		}
	}

	if imageData == nil {
		logger.Error("No image data found in response")
		return nil, fmt.Errorf("no image data found in response")
	}

	logger.Info("Image generation successful",
		"mime_type", imageData.MimeType,
		"finish_reason", candidate.FinishReason,
		"data_length", len(imageData.Data))

	return &GenerateImageResponse{
		ImageData:    imageData.Data,
		MimeType:     imageData.MimeType,
		FinishReason: candidate.FinishReason,
	}, nil
}
