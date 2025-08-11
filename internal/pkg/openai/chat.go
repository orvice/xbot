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
	ChatID  int64  `json:"chat_id"`
}

// ChatResponse represents the response structure
type ChatResponse struct {
	Output string `json:"output"`
}

func Chat(ctx context.Context, chatID int64, prompt string) string {
	logger := log.FromContext(ctx).With("method", "Chat")
	endpoint := conf.Conf.ChatEndpoint

	// Create request payload
	reqData := []ChatRequest{
		{
			Message: prompt,
			Code:    1,
			ChatID:  chatID,
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

	// Parse response - try array format first, then single object format
	var output string

	// Try to parse as array format: [{"output": "..."}]
	var chatRespArray []ChatResponse
	if err := json.Unmarshal(body, &chatRespArray); err == nil && len(chatRespArray) > 0 {
		output = chatRespArray[0].Output
		logger.Info("Chat request successful (array format)",
			"output_length", len(output))
	} else {
		// Try to parse as single object format: {"output": "..."}
		var chatRespSingle ChatResponse
		if err := json.Unmarshal(body, &chatRespSingle); err != nil {
			logger.Error("Failed to unmarshal response in both formats", "error", err, "body", string(body))
			return ""
		}
		output = chatRespSingle.Output
		logger.Info("Chat request successful (single object format)",
			"output_length", len(output))
	}

	return output
}
