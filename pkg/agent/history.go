package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"go-iot/pkg/models"
	"go-iot/pkg/redis"
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

var msgSeqClock struct {
	mu   sync.Mutex
	last map[string]int64
}

// NextSeqNo 分配单会话下单调递增序号。优先通过 Redis INCR 获取；若 Redis 不可用则降级到内存发号器。
func NextSeqNo(convId string) int64 {
	if client := redis.GetRedisClient(); client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		key := ConvRedisKey(convId, "seq")
		seq, err := client.Incr(ctx, key).Result()
		if err == nil && seq > 0 {
			_ = client.Expire(ctx, key, ConvSeqTTL).Err()
			return seq
		}
	}

	msgSeqClock.mu.Lock()
	defer msgSeqClock.mu.Unlock()
	if msgSeqClock.last == nil {
		msgSeqClock.last = map[string]int64{}
	}
	next := msgSeqClock.last[convId] + 1
	msgSeqClock.last[convId] = next
	return next
}

// EnsureConvSeq 确保会话序号 key 存在。若不存在则从已有历史消息中获取最大序号并创建，设置 1 天过期。
func EnsureConvSeq(ctx context.Context, convId string, store Store) {
	if convId == "" {
		return
	}
	client := redis.GetRedisClient()
	if client == nil {
		return
	}
	key := ConvRedisKey(convId, "seq")
	n, err := client.Exists(ctx, key).Result()
	if err == nil && n > 0 {
		_ = client.Expire(ctx, key, ConvSeqTTL).Err()
		return
	}
	var maxSeq int64
	if store != nil {
		if s, err := store.GetMaxSeq(convId); err == nil {
			maxSeq = s
		}
	}
	_ = client.SetNX(ctx, key, maxSeq, ConvSeqTTL).Err()

	msgSeqClock.mu.Lock()
	if msgSeqClock.last == nil {
		msgSeqClock.last = map[string]int64{}
	}
	if msgSeqClock.last[convId] < maxSeq {
		msgSeqClock.last[convId] = maxSeq
	}
	msgSeqClock.mu.Unlock()
}

// StampMessage 给消息写入自增序号和创建时间。
func StampMessage(m *models.AgentMessage) {
	if m == nil {
		return
	}
	if m.SeqNo <= 0 {
		m.SeqNo = NextSeqNo(m.ConversationId)
	}
	if time.Time(m.CreateTime).IsZero() {
		m.CreateTime = models.NewDateTime()
	}
}

// SortMessages 按 SeqNo 升序排序。若缺少 SeqNo（历史数据）降级到按创建时间排序。
func SortMessages(msgs []models.AgentMessage) {
	sort.SliceStable(msgs, func(i, j int) bool {
		si, sj := msgs[i].SeqNo, msgs[j].SeqNo
		if si > 0 && sj > 0 && si != sj {
			return si < sj
		}
		if si != sj {
			return si < sj
		}
		ti, tj := msgs[i].CreateTime.UnixMilli(), msgs[j].CreateTime.UnixMilli()
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
// When a compaction event is present, messages prior to firstKeptMessageId are omitted
// and the latest summary is prepended.
func ToInput(msgs []models.AgentMessage) []json.RawMessage {
	var latestCompaction *models.AgentMessage
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == RoleEvent && msgs[i].EventType == EventCompaction {
			latestCompaction = &msgs[i]
			break
		}
	}

	firstKeptIndex := 0
	var summaryText string
	if latestCompaction != nil {
		var cp CompactionPayload
		_ = json.Unmarshal([]byte(latestCompaction.Payload), &cp)
		summaryText = cp.Summary
		if summaryText == "" {
			summaryText = latestCompaction.Content
		}
		if cp.FirstKeptMessageId != "" {
			found := false
			for i, m := range msgs {
				if m.Id == cp.FirstKeptMessageId {
					firstKeptIndex = i
					found = true
					break
				}
			}
			if !found {
				for i, m := range msgs {
					if m.Id == latestCompaction.Id {
						firstKeptIndex = i + 1
						break
					}
				}
			}
		} else {
			for i, m := range msgs {
				if m.Id == latestCompaction.Id {
					firstKeptIndex = i + 1
					break
				}
			}
		}
	}

	type item struct {
		msg models.AgentMessage
		obj map[string]any
	}
	var items []item
	for i := firstKeptIndex; i < len(msgs); i++ {
		m := msgs[i]
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
			stripReasoningFromInput(it.obj)
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
	if latestCompaction != nil && strings.TrimSpace(summaryText) != "" {
		summaryObj := map[string]any{
			"role":    RoleUser,
			"content": fmt.Sprintf("The conversation history before this point was compacted into the following summary:\n\n<summary>\n%s\n</summary>", summaryText),
		}
		out = append([]json.RawMessage{mustRaw(summaryObj)}, out...)
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

func DraftExpired(d models.AgentDraft) bool {
	if time.Time(d.ExpireTime).IsZero() {
		return false
	}
	return time.Time(d.ExpireTime).Before(time.Now())
}

func PendingDrafts(drafts []models.AgentDraft) []models.AgentDraft {
	var out []models.AgentDraft
	for _, d := range drafts {
		if d.Status == DraftPending && !DraftExpired(d) {
			out = append(out, d)
		}
	}
	return out
}

func MessageView(m *models.AgentMessage, draftStatus string) map[string]any {
	if m == nil {
		return map[string]any{}
	}
	dto := map[string]any{
		"id": m.Id, "role": m.Role, "content": m.Content,
		"eventType": m.EventType, "toolName": m.ToolName,
		"toolCallId": m.ToolCallId, "draftId": m.DraftId,
		"productId": m.ProductId, "createdTime": m.CreateTime, "seqNo": m.SeqNo,
	}
	if m.Role == RoleAssistant {
		if r := reasoningFromPayload(m.Payload); r != "" {
			dto["reasoning"] = r
		}
	}
	if m.EventType == EventConfirmRequired {
		dto["preview"] = json.RawMessage(orJSON(extractPreviewJSON(m.Payload)))
		if draftStatus != "" {
			dto["draftStatus"] = draftStatus
		}
	}
	return dto
}

func reasoningFromPayload(payload string) string {
	var m map[string]any
	if json.Unmarshal([]byte(payload), &m) != nil {
		return ""
	}
	if s, ok := m["reasoning"].(string); ok {
		return s
	}
	if s, ok := m["reasoning_content"].(string); ok {
		return s
	}
	return ""
}

func stripReasoningFromInput(obj map[string]any) {
	if obj == nil {
		return
	}
	delete(obj, "reasoning")
	delete(obj, "reasoning_content")
	delete(obj, "thinking")
}

func extractPreviewJSON(payload string) string {
	var m map[string]json.RawMessage
	if json.Unmarshal([]byte(payload), &m) != nil {
		return payload
	}
	if p, ok := m["preview"]; ok {
		return string(p)
	}
	return payload
}

func RunStatusForDrafts(pending int) string {
	if pending > 0 {
		return RunAwaitingConfirm
	}
	return RunIdle
}
