package agent

import (
	"context"
	"errors"
	"sync"
	"testing"

	"go-iot/pkg/agent/client"
	"go-iot/pkg/models"

	"github.com/stretchr/testify/require"
)

func TestRunTurnAssistantOnlyJSON(t *testing.T) {
	origChat := DefaultChat
	origStream := DefaultChatStream
	t.Cleanup(func() {
		DefaultChat = origChat
		DefaultChatStream = origStream
	})
	DefaultChatStream = nil
	DefaultChat = func(ctx context.Context, baseURL, apiKey string, req client.CompletionsRequest) (*client.CompletionsResponse, error) {
		content := "你好"
		if len(req.Tools) == 0 {
			content = "温湿度接入"
		}
		return &client.CompletionsResponse{Choices: []struct {
			Message client.CompletionsMessage `json:"message"`
		}{{Message: client.CompletionsMessage{Role: "assistant", Content: content}}}}, nil
	}

	store := NewMemoryStore()
	st, err := PutSettings(store, 1, SettingsPut{
		BaseURL: "https://example.com/v1", Model: "m", ApiKey: "k",
	})
	require.NoError(t, err)
	c := &models.AgentConversation{
		Id: NewHexID(), Title: DefaultConvTitle, Status: "active",
		RunStatus: RunIdle, CreateId: 1, CreateTime: models.NewDateTime(),
	}
	require.NoError(t, store.SaveConversation(c))

	var mu sync.Mutex
	var events []string
	sink := eventLog{fn: func(ev string, _ any) {
		mu.Lock()
		defer mu.Unlock()
		events = append(events, ev)
	}}
	res, err := RunTurnOpts(context.Background(), RunOpts{
		Store: store, Conv: c, Settings: st, UserText: "帮我看看",
		Sink: sink, Perms: writeCtx(1).Perms,
	})
	require.NoError(t, err)
	require.Equal(t, RunIdle, res.RunStatus)
	msgs, _ := store.ListMessages(c.Id)
	require.GreaterOrEqual(t, len(msgs), 2)
	mu.Lock()
	defer mu.Unlock()
	require.Contains(t, events, "message")
	require.Contains(t, events, "done")
}

type eventLog struct {
	fn func(string, any)
}

func (e eventLog) Emit(event string, payload any) {
	if e.fn != nil {
		e.fn(event, payload)
	}
}

func TestRunTurnPausesOnCreateProduct(t *testing.T) {
	origChat := DefaultChat
	origStream := DefaultChatStream
	origLookup := lookupProduct
	t.Cleanup(func() {
		DefaultChat = origChat
		DefaultChatStream = origStream
		lookupProduct = origLookup
	})
	lookupProduct = func(id string) (*models.ProductModel, error) {
		return nil, errors.New("missing")
	}
	DefaultChatStream = nil
	DefaultChat = func(ctx context.Context, baseURL, apiKey string, req client.CompletionsRequest) (*client.CompletionsResponse, error) {
		var tc client.ToolCall
		tc.Id = "call_1"
		tc.Type = "function"
		tc.Function.Name = ToolCreateProduct
		tc.Function.Arguments = `{"id":"A1","name":"温湿度","networkType":"MQTT_BROKER"}`
		return &client.CompletionsResponse{Choices: []struct {
			Message client.CompletionsMessage `json:"message"`
		}{{Message: client.CompletionsMessage{ToolCalls: []client.ToolCall{tc}}}}}, nil
	}
	store := NewMemoryStore()
	st, err := PutSettings(store, 1, SettingsPut{BaseURL: "https://example.com/v1", Model: "m", ApiKey: "k"})
	require.NoError(t, err)
	c := &models.AgentConversation{Id: NewHexID(), Title: "t", Status: "active", RunStatus: RunIdle, CreateId: 1}
	require.NoError(t, store.SaveConversation(c))
	res, err := RunTurnOpts(context.Background(), RunOpts{
		Store: store, Conv: c, Settings: st, UserText: "建产品",
		Perms: writeCtx(1).Perms,
	})
	require.NoError(t, err)
	require.Equal(t, RunAwaitingConfirm, res.RunStatus)
	require.Len(t, res.PendingDrafts, 1)
}

func TestRunTurnLLMErrorReturnsError(t *testing.T) {
	origChat := DefaultChat
	origStream := DefaultChatStream
	t.Cleanup(func() {
		DefaultChat = origChat
		DefaultChatStream = origStream
	})
	DefaultChatStream = nil
	DefaultChat = func(ctx context.Context, baseURL, apiKey string, req client.CompletionsRequest) (*client.CompletionsResponse, error) {
		return nil, &client.HTTPError{Status: 401, Body: `{"error":{"message":"Authentication Fails, Your api key is invalid"}}`}
	}
	store := NewMemoryStore()
	st, err := PutSettings(store, 1, SettingsPut{BaseURL: "https://example.com/v1", Model: "m", ApiKey: "bad-key"})
	require.NoError(t, err)
	c := &models.AgentConversation{Id: NewHexID(), Title: "t", Status: "active", RunStatus: RunIdle, CreateId: 1}
	require.NoError(t, store.SaveConversation(c))

	res, err := RunTurnOpts(context.Background(), RunOpts{
		Store: store, Conv: c, Settings: st, UserText: "测试",
		Perms: writeCtx(1).Perms,
	})
	require.Error(t, err)
	require.Nil(t, res)
	require.Contains(t, err.Error(), "llm http 401")
	// 验证错误发生后会话状态被重置为 RunIdle
	latest, err := store.GetConversation(c.Id)
	require.NoError(t, err)
	require.Equal(t, RunIdle, latest.RunStatus)
}

func TestChatOnceSkipsFallbackOn401(t *testing.T) {
	streamCalled := false
	chatCalled := false
	streamFn := func(ctx context.Context, baseURL, apiKey string, req client.CompletionsRequest, onDelta func(client.StreamDelta)) (*client.CompletionsResponse, error) {
		streamCalled = true
		return nil, &client.HTTPError{Status: 401, Body: `{"error":"unauthorized"}`}
	}
	chatFn := func(ctx context.Context, baseURL, apiKey string, req client.CompletionsRequest) (*client.CompletionsResponse, error) {
		chatCalled = true
		return nil, nil
	}

	_, err := chatOnce(context.Background(), streamFn, chatFn, "https://api.test", "key", client.CompletionsRequest{Model: "m"}, nil)
	require.Error(t, err)
	require.True(t, streamCalled)
	require.False(t, chatCalled, "chatOnce should not fallback to chat on 401")
}

