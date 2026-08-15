package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChatSendsNestedToolsAndReasoningEffort(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/chat/completions", r.URL.Path)
		require.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		raw, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(raw, &got))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"ok"}}]}`))
	}))
	defer srv.Close()

	temp := 0.0
	parallel := false
	tools := []CompatTool{{
		Type: "function",
		Function: FunctionSpec{
			Name:        "list_products",
			Description: "list",
			Parameters:  json.RawMessage(`{"type":"object"}`),
		},
	}}
	cli := New()
	cli.HTTP = srv.Client()
	_, err := cli.Chat(context.Background(), srv.URL+"/v1", "test-key", CompletionsRequest{
		Model:             "grok-4.5",
		Messages:          []json.RawMessage{json.RawMessage(`{"role":"user","content":"hi"}`)},
		Tools:             tools,
		Temperature:       &temp,
		MaxTokens:         16,
		ParallelToolCalls: &parallel,
		ReasoningEffort:   "low",
		User:              "7",
	})
	require.NoError(t, err)

	raw, _ := json.Marshal(got)
	require.Contains(t, string(raw), `"reasoning_effort"`)
	require.NotContains(t, string(raw), `"reasoning":{"effort"`)
	require.Equal(t, "low", got["reasoning_effort"])
	require.Equal(t, float64(0), got["temperature"])

	tls, ok := got["tools"].([]any)
	require.True(t, ok)
	require.Len(t, tls, 1)
	tool0 := tls[0].(map[string]any)
	fn := tool0["function"].(map[string]any)
	require.Equal(t, "function", tool0["type"])
	require.Equal(t, "list_products", fn["name"])
	require.NotContains(t, strings.ToLower(string(raw)), `"type":"function","name":"list_products"`)
}

func TestIsToolsNotSupportedDoesNotMatchHistoryErrors(t *testing.T) {
	require.True(t, IsToolsNotSupported(&HTTPError{Status: 400, Body: `{"error":"tools are not supported"}`}))
	require.True(t, IsToolsNotSupported(&HTTPError{Status: 400, Body: `function calling is not supported`}))
	require.False(t, IsToolsNotSupported(&HTTPError{Status: 400, Body: `An assistant message with 'tool_calls' must be followed by tool messages responding to each tool_call_id`}))
	require.False(t, IsToolsNotSupported(&HTTPError{Status: 400, Body: `messages with role 'tool' must be a response to a preceding message with tool_calls`}))
	require.False(t, IsToolsNotSupported(&HTTPError{Status: 500, Body: `tools are not supported`}))
}

func TestJoinCompletionsURLRejectsDoubledPath(t *testing.T) {
	_, err := JoinCompletionsURL("https://api.x.ai/v1/chat/completions")
	require.Error(t, err)
	u, err := JoinCompletionsURL("https://api.x.ai/v1")
	require.NoError(t, err)
	require.Equal(t, "https://api.x.ai/v1/chat/completions", u)
}
