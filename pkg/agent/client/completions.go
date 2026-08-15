package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type StreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

type FunctionSpec struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters"`
}

type CompatTool struct {
	Type     string       `json:"type"`
	Function FunctionSpec `json:"function"`
}

type CompletionsRequest struct {
	Model             string            `json:"model"`
	Messages          []json.RawMessage `json:"messages"`
	Tools             []CompatTool      `json:"tools,omitempty"`
	ToolChoice        any               `json:"tool_choice,omitempty"`
	Temperature       *float64          `json:"temperature,omitempty"`
	Stream            bool              `json:"stream,omitempty"`
	StreamOptions     *StreamOptions    `json:"stream_options,omitempty"`
	MaxTokens         int               `json:"max_tokens,omitempty"`
	ParallelToolCalls *bool             `json:"parallel_tool_calls,omitempty"`
	ReasoningEffort   string            `json:"reasoning_effort,omitempty"`
	User              string            `json:"user,omitempty"`
}

type ToolCall struct {
	Id       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type CompletionsMessage struct {
	Role      string     `json:"role"`
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

type CompletionsResponse struct {
	Choices []struct {
		Message CompletionsMessage `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

type Client struct {
	HTTP *http.Client
}

func New() *Client {
	return &Client{
		HTTP: &http.Client{
			Timeout: 90 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

func JoinCompletionsURL(baseURL string) (string, error) {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return "", fmt.Errorf("baseUrl is empty")
	}
	if strings.HasSuffix(strings.TrimRight(baseURL, "/"), "/chat/completions") {
		return "", fmt.Errorf("baseUrl must not end with /chat/completions")
	}
	return url.JoinPath(baseURL, "chat/completions")
}

func (c *Client) Chat(ctx context.Context, baseURL, apiKey string, req CompletionsRequest) (*CompletionsResponse, error) {
	endpoint, err := JoinCompletionsURL(baseURL)
	if err != nil {
		return nil, err
	}
	if req.Stream {
		if req.StreamOptions == nil {
			req.StreamOptions = &StreamOptions{IncludeUsage: true}
		}
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	cli := c.HTTP
	if cli == nil {
		cli = New().HTTP
	}
	resp, err := cli.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, &HTTPError{Status: resp.StatusCode, Body: string(raw)}
	}
	var out CompletionsResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type HTTPError struct {
	Status int
	Body   string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("llm http %d: %s", e.Status, e.Body)
}

func IsToolsNotSupported(err error) bool {
	he, ok := err.(*HTTPError)
	if !ok || he.Status < 400 || he.Status >= 500 {
		return false
	}
	low := strings.ToLower(he.Body)
	// Only the provider saying the model cannot use tools. History/format
	// errors also mention tool/function and must not be mapped here.
	needles := []string{
		"tools are not supported",
		"tool use is not supported",
		"tools not supported",
		"does not support tool",
		"does not support function",
		"not support tools",
		"function calling is not supported",
		"function call is not supported",
		"tool_choice is not supported",
		"unknown field: tools",
		"unknown parameter: tools",
		"unknown parameter \"tools\"",
		"unexpected field: tools",
	}
	for _, n := range needles {
		if strings.Contains(low, n) {
			return true
		}
	}
	return false
}
