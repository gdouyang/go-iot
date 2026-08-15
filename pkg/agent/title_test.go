package agent

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"go-iot/pkg/agent/client"
	"go-iot/pkg/models"

	"github.com/stretchr/testify/require"
)

func TestNormalizeUserTitle(t *testing.T) {
	got, err := NormalizeUserTitle("  MQTT温湿度  ")
	require.NoError(t, err)
	require.Equal(t, "MQTT温湿度", got)
	_, err = NormalizeUserTitle("   ")
	require.Error(t, err)
	long := strings.Repeat("标", 130)
	got, err = NormalizeUserTitle(long)
	require.NoError(t, err)
	require.Equal(t, 128, len([]rune(got)))
}

func TestIsDefaultTitle(t *testing.T) {
	require.True(t, IsDefaultTitle(""))
	require.True(t, IsDefaultTitle("新会话"))
	require.True(t, IsDefaultTitle("  新会话  "))
	require.False(t, IsDefaultTitle("新会话1"))
	require.False(t, IsDefaultTitle("MQTT温湿度"))
}

func TestNextUntitledTitleIncrements(t *testing.T) {
	require.Equal(t, "新会话1", NextUntitledTitle(nil))
	require.Equal(t, "新会话1", NextUntitledTitle([]string{"新会话", "MQTT产品"}))
	require.Equal(t, "新会话2", NextUntitledTitle([]string{"新会话1"}))
	require.Equal(t, "新会话4", NextUntitledTitle([]string{"新会话1", "新会话3", "温湿度"}))
}

func TestSanitizeTitle(t *testing.T) {
	require.Equal(t, "MQTT温湿度", SanitizeTitle("  「MQTT温湿度」。\n第二行"))
	require.Equal(t, "", SanitizeTitle("新会话"))
	require.Equal(t, "", SanitizeTitle("新会话2"))
	long := "一二三四五六七八九十一二三四五六七八"
	got := SanitizeTitle(long)
	require.Equal(t, 16, len([]rune(got)))
}

func TestSummarizeTitleRequestOmitsParallelToolCalls(t *testing.T) {
	var got map[string]any
	orig := DefaultChat
	t.Cleanup(func() { DefaultChat = orig })
	DefaultChat = func(ctx context.Context, baseURL, apiKey string, req client.CompletionsRequest) (*client.CompletionsResponse, error) {
		raw, err := json.Marshal(req)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(raw, &got))
		return &client.CompletionsResponse{Choices: []struct {
			Message client.CompletionsMessage `json:"message"`
		}{{Message: client.CompletionsMessage{Content: "温湿度接入"}}}}, nil
	}
	gotTitle := ResolveTitle(context.Background(), NewMemoryStore(), &models.AgentConversation{Id: "c", CreateId: 1}, &models.AgentUserSettings{UserId: 1, ApiKey: "k", BaseURL: "https://example.com/v1", Model: "m"}, "新增温湿度")
	require.Equal(t, "温湿度接入", gotTitle)
	require.NotContains(t, got, "parallel_tool_calls")
	require.Equal(t, "none", got["reasoning_effort"])
	require.GreaterOrEqual(t, got["max_tokens"], float64(256))
}

func TestSummarizeTitleTimesOutAndFallsBack(t *testing.T) {
	origTO := titleTimeout
	titleTimeout = 20 * time.Millisecond
	t.Cleanup(func() { titleTimeout = origTO })

	orig := DefaultChat
	t.Cleanup(func() { DefaultChat = orig })
	DefaultChat = func(ctx context.Context, baseURL, apiKey string, req client.CompletionsRequest) (*client.CompletionsResponse, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	start := time.Now()
	got := ResolveTitle(context.Background(), NewMemoryStore(), &models.AgentConversation{Id: "c", Title: DefaultConvTitle, CreateId: 1}, &models.AgentUserSettings{UserId: 1, ApiKey: "k", BaseURL: "https://example.com/v1", Model: "m"}, "新增产品")
	require.Less(t, time.Since(start), time.Second)
	require.Equal(t, "新会话1", got)
}

func TestResolveTitleUsesModelThenFallback(t *testing.T) {
	store := NewMemoryStore()
	user := int64(9)
	exist := &models.AgentConversation{Id: "a", Title: "新会话1", CreateId: user}
	require.NoError(t, store.SaveConversation(exist))
	cur := &models.AgentConversation{Id: "b", Title: DefaultConvTitle, CreateId: user}
	require.NoError(t, store.SaveConversation(cur))

	orig := DefaultChat
	t.Cleanup(func() { DefaultChat = orig })
	DefaultChat = func(ctx context.Context, baseURL, apiKey string, req client.CompletionsRequest) (*client.CompletionsResponse, error) {
		return &client.CompletionsResponse{Choices: []struct {
			Message client.CompletionsMessage `json:"message"`
		}{{Message: client.CompletionsMessage{Content: "MQTT温湿度传感器"}}}}, nil
	}
	got := ResolveTitle(context.Background(), store, cur, &models.AgentUserSettings{UserId: user, ApiKey: "k", BaseURL: "https://example.com/v1", Model: "m"}, "新增一个MQTT温湿度")
	require.Equal(t, "MQTT温湿度传感器", got)

	DefaultChat = func(ctx context.Context, baseURL, apiKey string, req client.CompletionsRequest) (*client.CompletionsResponse, error) {
		return nil, context.DeadlineExceeded
	}
	got = ResolveTitle(context.Background(), store, cur, &models.AgentUserSettings{UserId: user, ApiKey: "k"}, "hi")
	require.Equal(t, "新会话2", got)
}
