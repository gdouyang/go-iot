package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-iot/pkg/agent"
	"go-iot/pkg/api/web/session"
	"go-iot/pkg/models"
	"go-iot/pkg/redis"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-chi/chi/v5"
	goredis "github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/require"
)

func setupAgentHTTP(t *testing.T) (*chi.Mux, func(user *models.User) string) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(func() { mr.Close() })
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	restore := redis.SetClientForTest(client)
	t.Cleanup(func() {
		_ = client.Close()
		restore()
	})

	store := agent.NewMemoryStore()
	agent.DefaultStore = store
	t.Cleanup(func() { agent.DefaultStore = agent.NewMemoryStore() })

	api := &agentApi{}
	r := chi.NewRouter()
	r.Get("/agent/status", api.status)
	r.Get("/agent/settings", api.getSettings)
	r.Put("/agent/settings", api.putSettings)
	r.Post("/agent/conversations", api.createConv)
	r.Post("/agent/conversations/page", api.pageConv)
	r.Get("/agent/conversations/{id}", api.getConv)
	r.Put("/agent/conversations/{id}", api.putConv)
	r.Get("/agent/conversations/{id}/messages", api.listMessages)
	r.Post("/agent/conversations/{id}/messages", api.postMessage)
	r.Post("/agent/conversations/{id}/apply", api.apply)
	r.Post("/agent/conversations/{id}/reject", api.reject)

	login := func(user *models.User) string {
		s := session.NewSession(3600)
		s.SetAttribute("user", user)
		s.SetPermission(map[string]bool{
			"agent-mgr:query": true, "agent-mgr:add": true,
			"agent-mgr:save": true, "agent-mgr:delete": true,
			"product-mgr:query": true, "product-mgr:add": true, "product-mgr:save": true,
		})
		return s.Sessionid
	}
	return r, login
}

func doJSON(t *testing.T, mux http.Handler, method, path, token string, body any) (int, map[string]any) {
	t.Helper()
	var rdr io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, rdr)
	if token != "" {
		req.Header.Set("Authorization", token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w.Code, out
}

func TestAgentMenuAndKeyGateNoYamlSwitch(t *testing.T) {
	mux, login := setupAgentHTTP(t)
	tok := login(&models.User{Id: 1, Username: "a"})

	code, body := doJSON(t, mux, "GET", "/agent/status", tok, nil)
	require.Equal(t, 200, code)
	res := body["result"].(map[string]any)
	require.Equal(t, false, res["modelConfigured"])
	require.Equal(t, agent.ReasonModelNotConfigured, res["reason"])
	_, hasEnabled := res["enabled"]
	require.False(t, hasEnabled)

	code, body = doJSON(t, mux, "POST", "/agent/conversations", tok, map[string]any{"title": "t"})
	require.Equal(t, 200, code)
	require.Equal(t, true, body["success"])

	priv := true
	code, body = doJSON(t, mux, "PUT", "/agent/settings", tok, map[string]any{
		"baseUrl": "http://127.0.0.1:11434/v1", "model": "llama", "apiKey": "k-1234",
		"allowPrivateLlm": priv, "writeMode": "confirm", "maxTurns": 8,
	})
	require.Equal(t, 200, code)
	st := body["result"].(map[string]any)
	require.Equal(t, true, st["modelConfigured"])
	require.Equal(t, true, st["allowPrivateLlm"])
	require.Equal(t, float64(8), st["maxTurns"])
}

func TestAgentIsolationAndMessagesModelGate(t *testing.T) {
	mux, login := setupAgentHTTP(t)
	tokA := login(&models.User{Id: 1, Username: "a"})
	tokB := login(&models.User{Id: 2, Username: "b"})

	code, body := doJSON(t, mux, "POST", "/agent/conversations", tokA, map[string]any{"title": "alice"})
	require.Equal(t, 200, code)
	conv := body["result"].(map[string]any)
	id := conv["id"].(string)

	code, body = doJSON(t, mux, "GET", "/agent/conversations/"+id, tokB, nil)
	require.Equal(t, 404, code)
	require.Equal(t, agent.ReasonNotFound, body["result"].(map[string]any)["reason"])

	code, body = doJSON(t, mux, "GET", "/agent/conversations/"+id+"/messages", tokB, nil)
	require.Equal(t, 404, code)

	code, body = doJSON(t, mux, "POST", "/agent/conversations/"+id+"/messages", tokA, map[string]any{"content": "hi"})
	require.Equal(t, 409, code)
	require.Equal(t, agent.ReasonModelNotConfigured, body["result"].(map[string]any)["reason"])
}

func TestSettingsNeverReturnFullKey(t *testing.T) {
	mux, login := setupAgentHTTP(t)
	tok := login(&models.User{Id: 4, Username: "u"})
	code, body := doJSON(t, mux, "PUT", "/agent/settings", tok, map[string]any{
		"baseUrl": "http://127.0.0.1:11434/v1", "model": "grok-4.5", "apiKey": "sk-abcdef9999",
		"allowPrivateLlm": true,
	})
	require.Equal(t, 200, code)
	res := body["result"].(map[string]any)
	require.Equal(t, true, res["apiKeySet"])
	require.Equal(t, "****9999", res["apiKeyMasked"])
	require.Equal(t, true, res["modelConfigured"])
	_, has := res["apiKey"]
	require.False(t, has)

	raw, _ := json.Marshal(body)
	require.NotContains(t, string(raw), "sk-abcdef9999")
}

func TestApplyRejectReplayEventsOverHTTP(t *testing.T) {
	mux, login := setupAgentHTTP(t)
	tok := login(&models.User{Id: 5, Username: "u"})

	code, body := doJSON(t, mux, "POST", "/agent/conversations", tok, map[string]any{"title": "c"})
	require.Equal(t, 200, code)
	id := body["result"].(map[string]any)["id"].(string)
	c, err := agent.DefaultStore.GetConversation(id)
	require.NoError(t, err)
	c.RunStatus = agent.RunAwaitingConfirm
	require.NoError(t, agent.DefaultStore.SaveConversation(c))

	d1 := &models.AgentDraft{
		Id: agent.NewHexID(), ConversationId: id, ToolCallId: "t1",
		ToolName: agent.ToolCreateProduct, Payload: `{}`, Status: agent.DraftPending,
		CreateId: 5, CreateTime: models.NewDateTime(),
	}
	d2 := &models.AgentDraft{
		Id: agent.NewHexID(), ConversationId: id, ToolCallId: "t2",
		ToolName: agent.ToolCreateProduct, Payload: `{}`, Status: agent.DraftPending,
		CreateId: 5, CreateTime: models.NewDateTime(),
	}
	require.NoError(t, agent.DefaultStore.SaveDraft(d1))
	require.NoError(t, agent.DefaultStore.SaveDraft(d2))

	code, body = doJSON(t, mux, "POST", "/agent/conversations/"+id+"/reject", tok, map[string]any{
		"draftIds": []string{d1.Id},
	})
	require.Equal(t, 200, code)
	res := body["result"].(map[string]any)
	require.Equal(t, agent.RunAwaitingConfirm, res["runStatus"])

	code, body = doJSON(t, mux, "GET", "/agent/conversations/"+id+"/messages", tok, nil)
	require.Equal(t, 200, code)
	list := body["result"].(map[string]any)["list"].([]any)
	var sawRejected bool
	for _, it := range list {
		m := it.(map[string]any)
		if m["eventType"] == agent.EventRejected {
			sawRejected = true
			require.Equal(t, agent.RoleEvent, m["role"])
		}
		require.NotEqual(t, agent.EventRejected, m["role"])
	}
	require.True(t, sawRejected)
}

func TestRenameConversationOverHTTP(t *testing.T) {
	mux, login := setupAgentHTTP(t)
	tok := login(&models.User{Id: 6, Username: "u"})
	tokB := login(&models.User{Id: 7, Username: "b"})

	code, body := doJSON(t, mux, "POST", "/agent/conversations", tok, map[string]any{"title": "新会话"})
	require.Equal(t, 200, code)
	id := body["result"].(map[string]any)["id"].(string)

	code, body = doJSON(t, mux, "PUT", "/agent/conversations/"+id, tok, map[string]any{"title": "  MQTT温湿度  "})
	require.Equal(t, 200, code)
	require.Equal(t, "MQTT温湿度", body["result"].(map[string]any)["title"])

	code, _ = doJSON(t, mux, "PUT", "/agent/conversations/"+id, tokB, map[string]any{"title": "别人的"})
	require.Equal(t, 404, code)

	code, body = doJSON(t, mux, "PUT", "/agent/conversations/"+id, tok, map[string]any{"title": "   "})
	require.Equal(t, 400, code)
	require.Equal(t, agent.ReasonInvalidArgs, body["result"].(map[string]any)["reason"])
}
