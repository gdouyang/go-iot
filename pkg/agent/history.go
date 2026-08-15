package agent

import (
	"encoding/json"
	"sort"
	"strings"
	"sync"
	"time"

	"go-iot/pkg/models"
)

const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleTool      = "tool"
	RoleEvent     = "event"

	EventConfirmRequired = "confirm_required"
	EventApplied         = "applied"
	EventRejected        = "rejected"
	EventUndone          = "undone"
	EventToolCall        = "tool_call"
	EventToolResult      = "tool_result"

	RunIdle            = "idle"
	RunRunning         = "running"
	RunAwaitingConfirm = "awaiting_confirm"
	RunApplying        = "applying"

	DraftPending  = "pending"
	DraftApplied  = "applied"
	DraftRejected = "rejected"
	DraftArtifact = "artifact"
)

func messageOrder(m models.AgentMessage) int {
	switch m.Role {
	case RoleUser:
		return 0
	case RoleAssistant:
		return 1
	case RoleEvent:
		switch m.EventType {
		case EventConfirmRequired:
			return 2
		case EventApplied, EventRejected:
			return 3
		default:
			return 4
		}
	case RoleTool:
		return 5
	default:
		return 6
	}
}

var msgClock struct {
	mu   sync.Mutex
	last map[string]int64
}

// StampMessage writes a strictly increasing millisecond timestamp on the message.
func StampMessage(m *models.AgentMessage) {
	if m == nil {
		return
	}
	now := time.Now().UnixMilli()
	msgClock.mu.Lock()
	defer msgClock.mu.Unlock()
	if msgClock.last == nil {
		msgClock.last = map[string]int64{}
	}
	if last := msgClock.last[m.ConversationId]; now <= last {
		now = last + 1
	}
	msgClock.last[m.ConversationId] = now
	m.CreateTimeMs = now
	if time.Time(m.CreateTime).IsZero() {
		m.CreateTime = models.DateTime(time.UnixMilli(now))
	}
}

func messageTime(m models.AgentMessage) int64 {
	if m.CreateTimeMs > 0 {
		return m.CreateTimeMs
	}
	return m.CreateTime.UnixMilli()
}

// SortMessages orders a conversation by CreateTimeMs (write order).
func SortMessages(msgs []models.AgentMessage) {
	sort.SliceStable(msgs, func(i, j int) bool {
		ti, tj := messageTime(msgs[i]), messageTime(msgs[j])
		if ti != tj {
			return ti < tj
		}
		oi, oj := messageOrder(msgs[i]), messageOrder(msgs[j])
		if oi != oj {
			return oi < oj
		}
		return msgs[i].Id < msgs[j].Id
	})
}

// ToInput rebuilds Completions messages. Event rows are never sent to the model.
// Tool results are attached to the assistant tool_calls they belong to, even if
// ES returned them out of order (same-second timestamps).
func ToInput(msgs []models.AgentMessage) []json.RawMessage {
	type item struct {
		msg models.AgentMessage
		obj map[string]any
	}
	var items []item
	for _, m := range msgs {
		if m.Role == RoleEvent || stringsEmpty(m.Payload) {
			continue
		}
		var obj map[string]any
		if json.Unmarshal([]byte(m.Payload), &obj) != nil {
			continue
		}
		items = append(items, item{msg: m, obj: obj})
	}

	type asstInfo struct {
		calls []callRef
	}
	var assts []asstInfo
	toolsByID := map[string][]item{}
	var unnamed []item
	for _, it := range items {
		switch roleOf(it.obj) {
		case RoleAssistant:
			var refs []callRef
			for _, c := range extractCalls(it.obj) {
				refs = append(refs, callRef{id: strings.TrimSpace(asString(c["id"])), name: callName(c)})
			}
			assts = append(assts, asstInfo{calls: refs})
		case RoleTool:
			id := strings.TrimSpace(asString(it.obj["tool_call_id"]))
			if id == "" {
				unnamed = append(unnamed, it)
			} else {
				toolsByID[id] = append(toolsByID[id], it)
			}
		}
	}
	ui := 0
	for ai := range assts {
		for ci := range assts[ai].calls {
			if assts[ai].calls[ci].id != "" {
				continue
			}
			id := "call_" + newHexID()
			if ui < len(unnamed) {
				unnamed[ui].obj["tool_call_id"] = id
				toolsByID[id] = append(toolsByID[id], unnamed[ui])
				ui++
			}
			assts[ai].calls[ci].id = id
		}
	}

	used := map[string]int{}
	var out []json.RawMessage
	ai := 0
	for _, it := range items {
		switch roleOf(it.obj) {
		case RoleTool:
			continue
		case RoleAssistant:
			info := assts[ai]
			ai++
			writeCallIDs(it.obj, info.calls)
			if len(info.calls) == 0 {
				delete(it.obj, "tool_calls")
			}
			out = append(out, mustRaw(it.obj))
			for _, ref := range info.calls {
				start := used[ref.id]
				list := toolsByID[ref.id]
				for _, t := range list[start:] {
					if name := firstNonEmpty(asString(t.obj["name"]), t.msg.ToolName, ref.name); name != "" {
						t.obj["name"] = name
					}
					out = append(out, mustRaw(t.obj))
				}
				used[ref.id] = len(list)
			}
		default:
			out = append(out, mustRaw(it.obj))
		}
	}
	return out
}

func roleOf(obj map[string]any) string {
	s, _ := obj["role"].(string)
	return s
}

func asString(v any) string {
	s, _ := v.(string)
	return s
}

func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}

func extractCalls(obj map[string]any) []map[string]any {
	raw, ok := obj["tool_calls"]
	if !ok || raw == nil {
		return nil
	}
	arr, ok := raw.([]any)
	if !ok || len(arr) == 0 {
		return nil
	}
	var out []map[string]any
	for _, it := range arr {
		m, ok := it.(map[string]any)
		if ok {
			out = append(out, m)
		}
	}
	return out
}

func callName(c map[string]any) string {
	fn, _ := c["function"].(map[string]any)
	if fn == nil {
		return ""
	}
	return asString(fn["name"])
}

type callRef struct {
	id   string
	name string
}

func writeCallIDs(obj map[string]any, refs []callRef) {
	calls := extractCalls(obj)
	if len(calls) == 0 || len(refs) == 0 {
		return
	}
	for i, ref := range refs {
		if i >= len(calls) {
			break
		}
		if ref.id != "" {
			calls[i]["id"] = ref.id
		}
		if asString(calls[i]["type"]) == "" {
			calls[i]["type"] = "function"
		}
	}
	arr := make([]any, len(calls))
	for i, c := range calls {
		arr[i] = c
	}
	obj["tool_calls"] = arr
}

func mustRaw(obj map[string]any) json.RawMessage {
	raw, _ := json.Marshal(obj)
	return raw
}

func stringsEmpty(s string) bool {
	return len(s) == 0
}

func TailMessages(msgs []models.AgentMessage, pageSize int) []models.AgentMessage {
	if pageSize <= 0 {
		pageSize = 50
	}
	if len(msgs) <= pageSize {
		return msgs
	}
	return msgs[len(msgs)-pageSize:]
}

func PendingDrafts(drafts []models.AgentDraft) []models.AgentDraft {
	var out []models.AgentDraft
	for _, d := range drafts {
		if d.Status == DraftPending {
			out = append(out, d)
		}
	}
	return out
}

func RunStatusForDrafts(pending int) string {
	if pending > 0 {
		return RunAwaitingConfirm
	}
	return RunIdle
}
