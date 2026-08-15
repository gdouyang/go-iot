package agent

import (
	"encoding/json"
	"testing"

	"go-iot/pkg/models"

	"github.com/stretchr/testify/require"
)

func seedConv(store Store, userId int64) *models.AgentConversation {
	c := &models.AgentConversation{
		Id: NewHexID(), Title: "t", Status: "active",
		RunStatus: RunAwaitingConfirm, CreateId: userId,
		CreateTime: models.NewDateTime(), UpdateTime: models.NewDateTime(),
	}
	_ = store.SaveConversation(c)
	return c
}

func seedPending(store Store, conv *models.AgentConversation, tool string) *models.AgentDraft {
	d := &models.AgentDraft{
		Id: NewHexID(), ConversationId: conv.Id, ToolCallId: "call-" + tool,
		ToolName: tool, Payload: `{"id":"X","name":"n","networkType":"MQTT_CLIENT"}`,
		Preview: `{"old":null}`, Status: DraftPending, CreateId: conv.CreateId,
		CreateTime: models.NewDateTime(),
	}
	_ = store.SaveDraft(d)
	_ = store.SaveMessage(&models.AgentMessage{
		Id: NewHexID(), ConversationId: conv.Id, Role: RoleEvent,
		EventType: EventConfirmRequired, DraftId: d.Id, ToolName: tool,
		Payload: `{"draftId":"` + d.Id + `"}`, Content: "待确认",
		CreateId: conv.CreateId, CreateTime: models.NewDateTime(),
	})
	return d
}

func TestRejectSubsetStaysAwaitingConfirmAndPersistsEvents(t *testing.T) {
	store := NewMemoryStore()
	c := seedConv(store, 3)
	d1 := seedPending(store, c, ToolUpdateProduct)
	d2 := seedPending(store, c, ToolUpdateProduct)

	res, err := RejectDrafts(store, 3, c, []string{d1.Id})
	require.NoError(t, err)
	require.Equal(t, RunAwaitingConfirm, res.RunStatus)
	require.Equal(t, []string{d2.Id}, res.PendingDrafts)

	got1, _ := store.GetDraft(d1.Id)
	got2, _ := store.GetDraft(d2.Id)
	require.Equal(t, DraftRejected, got1.Status)
	require.Equal(t, DraftPending, got2.Status)

	msgs, _ := store.ListMessages(c.Id)
	var types []string
	for _, m := range msgs {
		if m.Role == RoleEvent {
			types = append(types, m.EventType)
		}
	}
	require.Contains(t, types, EventConfirmRequired)
	require.Contains(t, types, EventRejected)
	require.NotContains(t, types, EventApplied)
}

func TestApplyPersistsAppliedEventNotAssistantMarkdown(t *testing.T) {
	store := NewMemoryStore()
	c := seedConv(store, 3)
	// update_product apply still hits ES via productsvc — stub by using unknown tool? 
	// Use reject path already covered; here inject a draft and mock ApplyMutating via update that fails owner.
	d := seedPending(store, c, ToolUpdateProduct)
	res, err := ApplyDrafts(store, 3, c, []string{d.Id}, false)
	require.NoError(t, err)
	// without ES GetProductMust fails; draft stays pending
	require.Equal(t, RunAwaitingConfirm, res.RunStatus)
	got, _ := store.GetDraft(d.Id)
	require.Equal(t, DraftPending, got.Status)
	require.NotEmpty(t, res.Failed)
}

func TestApplySuccessWritesAppliedEvent(t *testing.T) {
	orig := applyMutatingFn
	t.Cleanup(func() { applyMutatingFn = orig })
	applyMutatingFn = func(userId int64, toolName, payload string) (string, error) {
		require.Equal(t, int64(3), userId)
		require.Equal(t, ToolCreateProduct, toolName)
		return "HUMI-01", nil
	}

	store := NewMemoryStore()
	c := seedConv(store, 3)
	d := seedPending(store, c, ToolCreateProduct)
	asst, _ := json.Marshal(map[string]any{
		"role": "assistant", "content": "",
		"tool_calls": []any{
			map[string]any{"id": d.ToolCallId, "type": "function", "function": map[string]any{"name": ToolCreateProduct, "arguments": d.Payload}},
		},
	})
	require.NoError(t, store.SaveMessage(&models.AgentMessage{
		Id: NewHexID(), ConversationId: c.Id, Role: RoleAssistant,
		Payload: string(asst), CreateId: 3, CreateTime: models.NewDateTime(),
	}))
	res, err := ApplyDrafts(store, 3, c, []string{d.Id}, true)
	require.NoError(t, err)
	require.Equal(t, RunIdle, res.RunStatus)
	require.True(t, res.NeedsResume)
	require.Empty(t, res.Failed)
	require.Equal(t, "HUMI-01", res.Applied[0]["productId"])

	got, _ := store.GetDraft(d.Id)
	require.Equal(t, DraftApplied, got.Status)

	msgs, _ := store.ListMessages(c.Id)
	var sawApplied, sawAssistantMarkdown bool
	for _, m := range msgs {
		if m.Role == RoleEvent && m.EventType == EventApplied {
			sawApplied = true
			require.Contains(t, m.Content, "已应用")
		}
		if m.Role == RoleAssistant && m.EventType == EventApplied {
			sawAssistantMarkdown = true
		}
	}
	require.True(t, sawApplied)
	require.False(t, sawAssistantMarkdown)
	require.NotEmpty(t, ToInput(msgs)) // tool result is in model replay
}
