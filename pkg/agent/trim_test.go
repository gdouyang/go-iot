package agent

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompactOmitsOldLoadSkill(t *testing.T) {
	skill, _ := json.Marshal(map[string]any{
		"role": RoleTool, "name": ToolLoadSkill, "tool_call_id": "c1",
		"content": `{"ok":true,"name":"codec-mqtt-broker","markdown":"` + strings.Repeat("M", 200) + `"}`,
	})
	skill2, _ := json.Marshal(map[string]any{
		"role": RoleTool, "name": ToolLoadSkill, "tool_call_id": "c2",
		"content": `{"ok":true,"name":"codec-modbus","markdown":"` + strings.Repeat("N", 200) + `"}`,
	})
	user, _ := json.Marshal(map[string]any{"role": RoleUser, "content": "hi"})
	out := CompactAndTrim([]json.RawMessage{user, skill, skill2}, 0)
	require.Len(t, out, 3)
	var first, last map[string]any
	require.NoError(t, json.Unmarshal(out[1], &first))
	require.NoError(t, json.Unmarshal(out[2], &last))
	require.Contains(t, first["content"].(string), `"omitted":true`)
	require.Contains(t, last["content"].(string), "markdown")
}

func TestCompactKeepsSingleLoadSkillAcrossUserTurns(t *testing.T) {
	skill, _ := json.Marshal(map[string]any{
		"role": RoleTool, "name": ToolLoadSkill, "tool_call_id": "c1",
		"content": `{"ok":true,"name":"codec-mqtt-broker","markdown":"` + strings.Repeat("M", 200) + `"}`,
	})
	user1, _ := json.Marshal(map[string]any{"role": RoleUser, "content": "first"})
	user2, _ := json.Marshal(map[string]any{"role": RoleUser, "content": "second"})
	out := CompactAndTrim([]json.RawMessage{user1, skill, user2}, 0)
	var mid map[string]any
	require.NoError(t, json.Unmarshal(out[1], &mid))
	require.Contains(t, mid["content"].(string), "markdown")
	require.NotContains(t, mid["content"].(string), `"omitted":true`)
}

func TestTrimKeepsLastUserTurn(t *testing.T) {
	var msgs []json.RawMessage
	for i := 0; i < 8; i++ {
		u, _ := json.Marshal(map[string]any{"role": RoleUser, "content": strings.Repeat("x", 200)})
		a, _ := json.Marshal(map[string]any{"role": RoleAssistant, "content": strings.Repeat("y", 200)})
		msgs = append(msgs, u, a)
	}
	out := CompactAndTrim(msgs, 200)
	require.NotEmpty(t, out)
	var last map[string]any
	require.NoError(t, json.Unmarshal(out[len(out)-1], &last))
	require.Equal(t, RoleAssistant, last["role"])
	var firstKeep map[string]any
	require.NoError(t, json.Unmarshal(out[0], &firstKeep))
	require.Equal(t, RoleUser, firstKeep["role"])
}
