package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"butterfly.orx.me/core/log"
	"go.orx.me/xbot/internal/conf"
)

// ChatRequest represents the request structure
type ChatRequest struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// ChatMessage represents the message in the response
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatResponse represents the response structure
type ChatResponse struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

func Chat(ctx context.Context, prompt string) string {
	logger := log.FromContext(ctx).With("method", "Chat")
	endpoint := conf.Conf.ChatEndpoint

	// Create request payload
	reqData := []ChatRequest{
		{
			Message: prompt,
			Code:    1,
		},
	}

	// Marshal request to JSON
	jsonData, err := json.Marshal(reqData)
	if err != nil {
		logger.Error("Failed to marshal request", "error", err)
		return ""
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		logger.Error("Failed to create HTTP request", "error", err)
		return ""
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logger.Error("Failed to send HTTP request", "error", err)
		return ""
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("Failed to read response body", "error", err)
		return ""
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		logger.Error("HTTP request failed", "status", resp.StatusCode, "body", string(body))
		return ""
	}

	// Parse response
	var chatResp ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		logger.Error("Failed to unmarshal response", "error", err, "body", string(body))
		return ""
	}

	logger.Info("Chat request successful",
		"response_index", chatResp.Index,
		"finish_reason", chatResp.FinishReason,
		"content_length", len(chatResp.Message.Content))

	return chatResp.Message.Content
}
