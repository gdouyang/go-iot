package agent

import (
	"encoding/json"
)

const compactBodyLimit = 8192

// CompactAndTrim shrinks tool payloads then drops oldest turns until under maxTokens.
// maxTokens<=0 keeps everything after compact. System prompt is not in msgs.
func CompactAndTrim(msgs []json.RawMessage, maxTokens int) []json.RawMessage {
	out := compactToolPayloads(msgs)
	if maxTokens <= 0 {
		return out
	}
	sysCost := estimateTokens(SystemPromptMessage())
	budget := maxTokens - sysCost
	if budget < 1024 {
		budget = 1024
	}
	return trimToBudget(out, budget)
}

func compactToolPayloads(msgs []json.RawMessage) []json.RawMessage {
	lastSkill := -1
	lastByTool := map[string]int{}
	parsed := make([]map[string]any, len(msgs))
	for i, raw := range msgs {
		var obj map[string]any
		if json.Unmarshal(raw, &obj) != nil {
			continue
		}
		parsed[i] = obj
		if roleOf(obj) != RoleTool {
			continue
		}
		name := asString(obj["name"])
		if name == ToolLoadSkill {
			lastSkill = i
		}
		lastByTool[name] = i
	}
	out := make([]json.RawMessage, 0, len(msgs))
	for i, raw := range msgs {
		obj := parsed[i]
		if obj == nil {
			out = append(out, raw)
			continue
		}
		if roleOf(obj) != RoleTool {
			out = append(out, raw)
			continue
		}
		name := asString(obj["name"])
		content := asString(obj["content"])
		keepFull := false
		switch name {
		case ToolLoadSkill:
			// Keep the last skill body unchanged so later turns only append (prefix cache).
			keepFull = i == lastSkill
		case ToolGetTSLSchema:
			keepFull = false
		case ToolGetTSL, ToolGetScript, ToolGetNetwork, ToolGetCollector, ToolGetProductConfig, ToolGetDebugLogs, ToolRunScript:
			keepFull = lastByTool[name] == i
		default:
			keepFull = len(content) <= compactBodyLimit || lastByTool[name] == i
		}
		if keepFull {
			out = append(out, raw)
			continue
		}
		obj["content"] = compactContent(name, content)
		out = append(out, mustRaw(obj))
	}
	return out
}

func compactContent(name, content string) string {
	var body map[string]any
	_ = json.Unmarshal([]byte(content), &body)
	skill := ""
	if body != nil {
		skill, _ = body["name"].(string)
	}
	msg := map[string]any{
		"ok": true, "omitted": true, "tool": name,
		"message": "full result omitted; call " + name + " again if you need the body",
	}
	if skill != "" {
		msg["name"] = skill
	}
	raw, _ := json.Marshal(msg)
	return string(raw)
}

func trimToBudget(msgs []json.RawMessage, budget int) []json.RawMessage {
	if len(msgs) == 0 {
		return msgs
	}
	cost := make([]int, len(msgs))
	total := 0
	for i, m := range msgs {
		cost[i] = estimateTokens(m)
		total += cost[i]
	}
	if total <= budget {
		return msgs
	}
	drop := make([]bool, len(msgs))
	for i := 0; i < len(msgs) && total > budget; i++ {
		if isProtectedTail(msgs, i) {
			continue
		}
		drop[i] = true
		total -= cost[i]
		// also drop following tool rows of this assistant so the transcript stays valid
		if roleRaw(msgs[i]) == RoleAssistant {
			for j := i + 1; j < len(msgs) && roleRaw(msgs[j]) == RoleTool; j++ {
				if !drop[j] {
					drop[j] = true
					total -= cost[j]
				}
			}
		}
	}
	out := make([]json.RawMessage, 0, len(msgs))
	for i, m := range msgs {
		if !drop[i] {
			out = append(out, m)
		}
	}
	if len(out) == 0 {
		return msgs[len(msgs)-1:]
	}
	return out
}

func isProtectedTail(msgs []json.RawMessage, i int) bool {
	// keep the last user turn and everything after it
	lastUser := -1
	for j := len(msgs) - 1; j >= 0; j-- {
		if roleRaw(msgs[j]) == RoleUser {
			lastUser = j
			break
		}
	}
	if lastUser < 0 {
		return i >= len(msgs)-4
	}
	return i >= lastUser
}

func roleRaw(raw json.RawMessage) string {
	var obj map[string]any
	if json.Unmarshal(raw, &obj) != nil {
		return ""
	}
	return roleOf(obj)
}

func estimateTokens(raw json.RawMessage) int {
	n := len(raw)
	if n == 0 {
		return 0
	}
	return (n + 3) / 4
}
