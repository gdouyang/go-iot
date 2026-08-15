package agent

import (
	"encoding/json"
	"testing"

	"go-iot/pkg/models"

	"github.com/stretchr/testify/require"
)

func TestToInputOmitsEventRows(t *testing.T) {
	user, _ := json.Marshal(map[string]any{"role": "user", "content": "hi"})
	asst, _ := json.Marshal(map[string]any{
		"role": "assistant", "content": "ok",
		"tool_calls": []any{
			map[string]any{"id": "c1", "type": "function", "function": map[string]any{"name": "list_products", "arguments": "{}"}},
		},
	})
	ev, _ := json.Marshal(map[string]any{"draftId": "d1", "tool": "create_product"})
	tool, _ := json.Marshal(map[string]any{"role": "tool", "tool_call_id": "c1", "content": "{}"})
	msgs := []models.AgentMessage{
		{Role: RoleUser, Payload: string(user)},
		{Role: RoleAssistant, Payload: string(asst)},
		{Role: RoleEvent, EventType: EventConfirmRequired, Payload: string(ev)},
		{Role: RoleEvent, EventType: EventApplied, Payload: string(ev)},
		{Role: RoleTool, Payload: string(tool), ToolName: "list_products"},
	}
	in := ToInput(msgs)
	require.Len(t, in, 3)
	roles := make([]string, 0, len(in))
	for _, raw := range in {
		var m map[string]any
		require.NoError(t, json.Unmarshal(raw, &m))
		require.NotEqual(t, RoleEvent, m["role"])
		_, hasDraft := m["draftId"]
		require.False(t, hasDraft)
		roles = append(roles, m["role"].(string))
	}
	require.Equal(t, []string{RoleUser, RoleAssistant, RoleTool}, roles)
}

func TestToInputAttachesOutOfOrderToolResults(t *testing.T) {
	user, _ := json.Marshal(map[string]any{"role": "user", "content": "继续"})
	asst, _ := json.Marshal(map[string]any{
		"role": "assistant", "content": "请点击应用",
		"tool_calls": []any{
			map[string]any{"id": "call_create", "type": "function", "function": map[string]any{"name": "create_product", "arguments": "{}"}},
		},
	})
	tool, _ := json.Marshal(map[string]any{"role": "tool", "tool_call_id": "call_create", "content": `{"ok":true}`})
	// Same-second ES order can put the tool row before the assistant.
	msgs := []models.AgentMessage{
		{Role: RoleTool, Payload: string(tool), ToolName: "create_product"},
		{Role: RoleUser, Payload: string(user)},
		{Role: RoleAssistant, Payload: string(asst)},
	}
	in := ToInput(msgs)
	require.Len(t, in, 3)
	var roles []string
	for _, raw := range in {
		var m map[string]any
		require.NoError(t, json.Unmarshal(raw, &m))
		roles = append(roles, m["role"].(string))
		if m["role"] == RoleTool {
			require.Equal(t, "call_create", m["tool_call_id"])
			require.Equal(t, "create_product", m["name"])
		}
		if m["role"] == RoleAssistant {
			_, ok := m["tool_calls"]
			require.True(t, ok)
		}
	}
	require.Equal(t, []string{RoleUser, RoleAssistant, RoleTool}, roles)
}

func TestToInputDropsEmptyToolCallsAndFillsMissingIDs(t *testing.T) {
	asst, _ := json.Marshal(map[string]any{
		"role": "assistant", "content": "",
		"tool_calls": []any{
			map[string]any{"type": "function", "function": map[string]any{"name": "get_tsl_schema", "arguments": "{}"}},
		},
	})
	tool, _ := json.Marshal(map[string]any{"role": "tool", "content": `{"ok":true}`})
	plain, _ := json.Marshal(map[string]any{"role": "assistant", "content": "好", "tool_calls": []any{}})
	in := ToInput([]models.AgentMessage{
		{Role: RoleAssistant, Payload: string(asst)},
		{Role: RoleTool, Payload: string(tool), ToolName: "get_tsl_schema"},
		{Role: RoleAssistant, Payload: string(plain)},
	})
	require.Len(t, in, 3)
	var first map[string]any
	require.NoError(t, json.Unmarshal(in[0], &first))
	calls := first["tool_calls"].([]any)
	require.Len(t, calls, 1)
	id := calls[0].(map[string]any)["id"].(string)
	require.NotEmpty(t, id)

	var mid map[string]any
	require.NoError(t, json.Unmarshal(in[1], &mid))
	require.Equal(t, RoleTool, mid["role"])
	require.Equal(t, id, mid["tool_call_id"])
	require.Equal(t, "get_tsl_schema", mid["name"])

	var last map[string]any
	require.NoError(t, json.Unmarshal(in[2], &last))
	_, has := last["tool_calls"]
	require.False(t, has)
}

func TestSortMessagesByCreateTimeMs(t *testing.T) {
	msgs := []models.AgentMessage{
		{Id: "t", Role: RoleTool, CreateTimeMs: 5},
		{Id: "e", Role: RoleEvent, EventType: EventApplied, CreateTimeMs: 4},
		{Id: "a", Role: RoleAssistant, CreateTimeMs: 2},
		{Id: "c", Role: RoleEvent, EventType: EventConfirmRequired, CreateTimeMs: 3},
		{Id: "u", Role: RoleUser, CreateTimeMs: 1},
	}
	SortMessages(msgs)
	require.Equal(t, []string{"u", "a", "c", "e", "t"}, []string{msgs[0].Id, msgs[1].Id, msgs[2].Id, msgs[3].Id, msgs[4].Id})
}

func TestSaveMessageAssignsIncreasingTimestamp(t *testing.T) {
	store := NewMemoryStore()
	conv := "conv-ts"
	var last int64
	for i := 0; i < 8; i++ {
		m := &models.AgentMessage{Id: NewHexID(), ConversationId: conv, Role: RoleUser, Payload: `{"role":"user","content":"x"}`}
		require.NoError(t, store.SaveMessage(m))
		require.Greater(t, m.CreateTimeMs, last)
		last = m.CreateTimeMs
	}
	list, err := store.ListMessages(conv)
	require.NoError(t, err)
	require.Len(t, list, 8)
	for i := 1; i < len(list); i++ {
		require.Greater(t, list[i].CreateTimeMs, list[i-1].CreateTimeMs)
	}
}

func TestHasNoDeployTool(t *testing.T) {
	require.False(t, HasDeployTool())
	for _, t0 := range ToolCatalog() {
		require.NotEqual(t, "deploy_product", t0.Function.Name)
		require.NotEqual(t, "start_network", t0.Function.Name)
		require.Equal(t, "function", t0.Type)
		require.NotEmpty(t, t0.Function.Name)
	}
}
