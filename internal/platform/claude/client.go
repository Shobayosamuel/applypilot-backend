// Package claude is a minimal client for the Claude Messages API.
package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"strings"
	"time"
)

// Minimal Claude Messages API client (https://docs.claude.com/en/api/messages).

var apiURL = "https://api.anthropic.com/v1/messages"

// SetBaseURL points the client at a different host (e.g. a gateway), keeping the /v1/messages path.
func SetBaseURL(base string) { apiURL = strings.TrimRight(base, "/") + "/v1/messages" }

type CacheControl struct {
	Type string `json:"type"` // "ephemeral"
}

type Source struct {
	Type      string `json:"type"` // "base64"
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

type Block struct {
	Type         string        `json:"type"` // text | document | tool_use
	Text         string        `json:"text,omitempty"`
	Source       *Source       `json:"source,omitempty"`
	CacheControl *CacheControl `json:"cache_control,omitempty"`
}

type Message struct {
	Role    string  `json:"role"`
	Content []Block `json:"content"`
}

type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
}

type ToolChoice struct {
	Type string `json:"type"` // auto | any | tool
	Name string `json:"name,omitempty"`
}

type Request struct {
	Model      string      `json:"model"`
	MaxTokens  int         `json:"max_tokens"`
	System     []Block     `json:"system,omitempty"`
	Messages   []Message   `json:"messages"`
	Tools      []Tool      `json:"tools,omitempty"`
	ToolChoice *ToolChoice `json:"tool_choice,omitempty"`
}

type Usage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
}

type ResponseBlock struct {
	Type  string          `json:"type"`
	Text  string          `json:"text"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

type Response struct {
	Model      string          `json:"model"`
	Content    []ResponseBlock `json:"content"`
	StopReason string          `json:"stop_reason"`
	Usage      Usage           `json:"usage"`
}

func (r *Response) Text() string {
	var b bytes.Buffer
	for _, c := range r.Content {
		if c.Type == "text" {
			b.WriteString(c.Text)
		}
	}
	return b.String()
}

type APIError struct {
	Status  int
	Type    string
	Message string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("claude api %d %s: %s", e.Status, e.Type, e.Message)
}

var ErrNoAPIKey = errors.New("ANTHROPIC_API_KEY is not configured")

type Client struct {
	apiKey string
	http   *http.Client
}

func NewClient(apiKey string) *Client {
	return &Client{apiKey: apiKey, http: &http.Client{Timeout: 180 * time.Second}}
}

// Create sends a request, retrying on 429/5xx/529 with jittered exponential backoff.
func (c *Client) Create(ctx context.Context, req Request) (*Response, error) {
	if c.apiKey == "" {
		return nil, ErrNoAPIKey
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	var lastErr error
	for attempt := 0; attempt < 4; attempt++ {
		if attempt > 0 {
			d := time.Duration(1<<attempt)*time.Second + time.Duration(rand.IntN(500))*time.Millisecond
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(d):
			}
		}
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		httpReq.Header.Set("content-type", "application/json")
		httpReq.Header.Set("x-api-key", c.apiKey)
		httpReq.Header.Set("anthropic-version", "2023-06-01")

		resp, err := c.http.Do(httpReq)
		if err != nil {
			lastErr = err
			continue
		}
		raw, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
		resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}
		if resp.StatusCode == http.StatusOK {
			var out Response
			if err := json.Unmarshal(raw, &out); err != nil {
				return nil, fmt.Errorf("decode claude response: %w", err)
			}
			return &out, nil
		}
		apiErr := &APIError{Status: resp.StatusCode}
		var e struct {
			Error struct {
				Type    string `json:"type"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal(raw, &e) == nil {
			apiErr.Type, apiErr.Message = e.Error.Type, e.Error.Message
		}
		lastErr = apiErr
		if resp.StatusCode == 429 || resp.StatusCode >= 500 {
			continue
		}
		return nil, apiErr
	}
	return nil, lastErr
}

// Structured forces Claude to call a single tool and decodes its input into out.
// This is the most reliable way to get schema-conforming JSON back.
func (c *Client) Structured(ctx context.Context, req Request, tool Tool, out any) (*Response, error) {
	req.Tools = []Tool{tool}
	req.ToolChoice = &ToolChoice{Type: "tool", Name: tool.Name}
	resp, err := c.Create(ctx, req)
	if err != nil {
		return nil, err
	}
	for _, b := range resp.Content {
		if b.Type == "tool_use" && b.Name == tool.Name {
			if err := json.Unmarshal(b.Input, out); err != nil {
				return resp, fmt.Errorf("decode tool input: %w", err)
			}
			return resp, nil
		}
	}
	return resp, errors.New("model did not return structured output")
}
