package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	DefaultGroqModel  = "llama-3.3-70b-versatile"
	DefaultBaseURL    = "https://api.groq.com/openai/v1/chat/completions"
	DefaultTimeout    = 60 * time.Second
	MaxRetries        = 3
	InitialBackoff    = 1 * time.Second
	MaxBackoff        = 30 * time.Second
	BackoffMultiplier = 2.0
)

// GroqClient handles communication with the Groq API
type GroqClient struct {
	apiKey     string
	httpClient *http.Client
	baseURL    string
}

// ChatMessage represents a message in the chat conversation
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest represents a request to the Groq chat API
type ChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
	MaxTokens   int           `json:"max_tokens"`
}

// ChatResponse represents a response from the Groq chat API
type ChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int         `json:"index"`
		Message      ChatMessage `json:"message"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// GroqError represents an error response from the Groq API
type GroqError struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

// NewGroqClient creates a new Groq API client
func NewGroqClient(apiKey string) *GroqClient {
	return &GroqClient{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
		baseURL: DefaultBaseURL,
	}
}

// NewGroqClientWithOptions creates a new Groq API client with custom options
func NewGroqClientWithOptions(apiKey string, baseURL string, timeout time.Duration) *GroqClient {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	if timeout == 0 {
		timeout = DefaultTimeout
	}

	return &GroqClient{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		baseURL: baseURL,
	}
}

// Chat sends a chat completion request to the Groq API with retry logic
func (c *GroqClient) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	// Set default model if not specified
	if req.Model == "" {
		req.Model = DefaultGroqModel
	}

	// Set default temperature if not specified
	if req.Temperature == 0 {
		req.Temperature = 0.7
	}

	// Set default max tokens if not specified
	if req.MaxTokens == 0 {
		req.MaxTokens = 1024
	}

	var lastErr error
	backoff := InitialBackoff

	for attempt := 0; attempt <= MaxRetries; attempt++ {
		if attempt > 0 {
			// Wait before retry with exponential backoff
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
			backoff = time.Duration(float64(backoff) * BackoffMultiplier)
			if backoff > MaxBackoff {
				backoff = MaxBackoff
			}
		}

		resp, err := c.doRequest(ctx, req)
		if err == nil {
			return resp, nil
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryableError(err) {
			return nil, err
		}
	}

	return nil, fmt.Errorf("failed after %d retries: %w", MaxRetries, lastErr)
}

// doRequest performs the actual HTTP request to the Groq API
func (c *GroqClient) doRequest(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var groqErr GroqError
		if err := json.Unmarshal(respBody, &groqErr); err == nil && groqErr.Error.Message != "" {
			return nil, &APIError{
				StatusCode: resp.StatusCode,
				Message:    groqErr.Error.Message,
				Type:       groqErr.Error.Type,
				Code:       groqErr.Error.Code,
			}
		}
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(respBody),
		}
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &chatResp, nil
}

// APIError represents an API error with status code and message
type APIError struct {
	StatusCode int
	Message    string
	Type       string
	Code       string
}

func (e *APIError) Error() string {
	if e.Type != "" {
		return fmt.Sprintf("Groq API error (%d): %s - %s", e.StatusCode, e.Type, e.Message)
	}
	return fmt.Sprintf("Groq API error (%d): %s", e.StatusCode, e.Message)
}

// IsRateLimitError checks if the error is a rate limit error
func (e *APIError) IsRateLimitError() bool {
	return e.StatusCode == http.StatusTooManyRequests
}

// IsServerError checks if the error is a server error
func (e *APIError) IsServerError() bool {
	return e.StatusCode >= 500
}

// isRetryableError checks if an error should be retried
func isRetryableError(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		// Retry on rate limits and server errors
		return apiErr.IsRateLimitError() || apiErr.IsServerError()
	}
	return false
}

// GetMessageContent extracts the content from the first choice in the response
func (r *ChatResponse) GetMessageContent() string {
	if len(r.Choices) == 0 {
		return ""
	}
	return r.Choices[0].Message.Content
}
