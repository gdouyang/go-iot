package client

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChatStreamParsesToolCallDeltas(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "text/event-stream", r.Header.Get("Accept"))
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call_1\",\"type\":\"function\",\"function\":{\"name\":\"list_products\",\"arguments\":\"\"}}]}}]}\n\n")
		_, _ = io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\"{}\"}}]}}]}\n\n")
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()
	cli := New()
	cli.HTTP = srv.Client()
	resp, err := cli.ChatStream(context.Background(), srv.URL+"/v1", "k", CompletionsRequest{Model: "m"}, nil)
	require.NoError(t, err)
	require.Len(t, resp.Choices[0].Message.ToolCalls, 1)
	require.Equal(t, "list_products", resp.Choices[0].Message.ToolCalls[0].Function.Name)
	require.Equal(t, "{}", resp.Choices[0].Message.ToolCalls[0].Function.Arguments)
}

func TestChatStreamFallsBackToJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"hi"}}]}`)
	}))
	defer srv.Close()
	cli := New()
	cli.HTTP = srv.Client()
	var got string
	resp, err := cli.ChatStream(context.Background(), srv.URL+"/v1", "k", CompletionsRequest{Model: "m"}, func(d StreamDelta) { got += d.Content })
	require.NoError(t, err)
	require.Equal(t, "hi", resp.Choices[0].Message.Content)
	require.Equal(t, "hi", got)
}

func TestChatStreamContentDeltas(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"hel\"}}]}\n\n")
		_, _ = io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"lo\"}}]}\n\n")
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()
	cli := New()
	cli.HTTP = srv.Client()
	var b strings.Builder
	resp, err := cli.ChatStream(context.Background(), srv.URL+"/v1", "k", CompletionsRequest{Model: "m"}, func(d StreamDelta) { b.WriteString(d.Content) })
	require.NoError(t, err)
	require.Equal(t, "hello", resp.Choices[0].Message.Content)
	require.Equal(t, "hello", b.String())
}

func TestChatStreamParsesReasoningContent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"reasoning_content\":\"想一\"}}]}\n\n")
		_, _ = io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"reasoning_content\":\"想\"}}]}\n\n")
		_, _ = io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"好的\"}}]}\n\n")
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()
	cli := New()
	cli.HTTP = srv.Client()
	var reason, content strings.Builder
	resp, err := cli.ChatStream(context.Background(), srv.URL+"/v1", "k", CompletionsRequest{Model: "m"}, func(d StreamDelta) {
		reason.WriteString(d.Reasoning)
		content.WriteString(d.Content)
	})
	require.NoError(t, err)
	require.Equal(t, "想一想", resp.Choices[0].Message.Reasoning)
	require.Equal(t, "好的", resp.Choices[0].Message.Content)
	require.Equal(t, "想一想", reason.String())
	require.Equal(t, "好的", content.String())
}

func TestChatStreamReturnsHTTPErrorOn401(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"Authentication Fails, Your api key is invalid"}}`))
	}))
	defer srv.Close()

	cli := New()
	cli.HTTP = srv.Client()
	_, err := cli.ChatStream(context.Background(), srv.URL+"/v1", "bad-key", CompletionsRequest{Model: "m"}, nil)
	require.Error(t, err)
	he, ok := err.(*HTTPError)
	require.True(t, ok)
	require.Equal(t, 401, he.Status)
	require.Contains(t, he.Body, "Authentication Fails")
	require.Contains(t, err.Error(), "llm http 401")
}

