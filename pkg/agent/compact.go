package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"go-iot/pkg/agent/client"
	"go-iot/pkg/models"
)

const (
	EventCompaction           = "compaction"
	RunCompacting             = "compacting"
	DefaultReserveTokens      = 16384
	DefaultKeepRecentTokens   = 20000
	CompactThresholdRatio     = 0.8
	ToolResultSummaryMaxChars = 2000

	SummarizationSystemPrompt = `You are a context summarization assistant. Your task is to read a conversation between a user and an IoT management AI assistant, then produce a structured summary following the exact format specified.

Do NOT continue the conversation. Do NOT respond to any questions in the conversation. ONLY output the structured summary.`

	InitialSummaryPrompt = `Produce a structured summary of the conversation for an IoT device/product management assistant. Follow this EXACT format:

## Goal
[What the user is trying to accomplish with IoT product/device setup]

## Product
- productId: [ID if mentioned]
- networkType: [network type, e.g. MQTT_BROKER, TCP_SERVER, MODBUS, etc.]
- status: [whether deployed/published]
- port: [port if applicable]

## Config
- [Whether username/password or other configurations are set (DO NOT include plain passwords)]

## TSL / Script / Collector
- TSL: [Key property/function/event IDs defined]
- Script: [Codec script responsibilities, e.g. OnConnect authentication, OnMessage decode, etc.]
- Collector: [Collector/point table configuration if applicable]

## Progress / Decisions / Blocked / Next
- Progress: [What has been completed]
- Decisions: [Key architectural or configuration choices made]
- Blocked: [Issues or errors encountered, if any]
- Next: [Next steps to complete the task]

Keep each section concise. Preserve exact IDs and names.`

	UpdateSummaryPrompt = `The conversation messages above are NEW conversation messages to incorporate into the existing summary provided in <previous-summary> tags.

Update the existing structured summary with new information. RULES:
- PRESERVE all existing information from the previous summary unless superseded
- ADD new progress, decisions, and context from the new messages
- UPDATE Product, Config, TSL/Script/Collector sections if changes were made
- UPDATE Progress / Decisions / Blocked / Next based on recent actions
- DO NOT include plain passwords

Use this EXACT format:

## Goal
[Preserve existing goals, add new ones if expanded]

## Product
- productId: [ID]
- networkType: [network type]
- status: [whether deployed]
- port: [port]

## Config
- [Whether username/password or other configs are set (DO NOT include plain passwords)]

## TSL / Script / Collector
- TSL: [Key property/function/event IDs]
- Script: [Codec script responsibilities]
- Collector: [Collector/point table if applicable]

## Progress / Decisions / Blocked / Next
- Progress: [What has been completed]
- Decisions: [Key architectural or configuration choices made]
- Blocked: [Issues or errors encountered, if any]
- Next: [Next steps to complete the task]

Keep each section concise.`
)

type CompactionPayload struct {
	Summary            string `json:"summary"`
	FirstKeptMessageId string `json:"firstKeptMessageId"`
	TokensBefore       int    `json:"tokensBefore"`
}

type CutPointResult struct {
	StartIndex         int
	CutIndex           int
	FirstKeptMessageId string
	PreviousSummary    string
	TurnPrefixMessages []models.AgentMessage
	IsSplitTurn        bool
}

func EstimateMessageTokens(m models.AgentMessage) int {
	if len(m.Payload) > 0 {
		return (len(m.Payload) + 3) / 4
	}
	return (len(m.Content) + 3) / 4
}

func EstimateMessagesTokens(msgs []models.AgentMessage) int {
	total := 0
	for _, m := range msgs {
		total += EstimateMessageTokens(m)
	}
	return total
}

// ShouldCompact checks if conversation context exceeds the 80% usable threshold.
// It directly uses the configured MaxPromptTokens (上下文长度).
func ShouldCompact(conv *models.AgentConversation, msgs []models.AgentMessage, cfg *models.AgentUserSettings) bool {
	if conv == nil || len(msgs) <= 1 || cfg == nil {
		return false
	}
	used := conv.LastPromptTokens
	if used <= 0 {
		used = EstimateMessagesTokens(msgs)
	}

	usable := cfg.MaxPromptTokens
	if usable <= 0 {
		usable = DefaultMaxPromptTokens
	}

	threshold := int(float64(usable) * CompactThresholdRatio)
	return used > threshold
}

// FindCutPoint locates the cut point to keep recent turns under keepRecentTokens budget.
// It prioritizes user message boundaries and falls back to assistant message boundaries for split turns.
func FindCutPoint(msgs []models.AgentMessage, cfg *models.AgentUserSettings) *CutPointResult {
	if len(msgs) <= 1 {
		return nil
	}
	startIndex := 0
	prevSummary := ""
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == RoleEvent && msgs[i].EventType == EventCompaction {
			var cp CompactionPayload
			_ = json.Unmarshal([]byte(msgs[i].Payload), &cp)
			prevSummary = cp.Summary
			if prevSummary == "" {
				prevSummary = msgs[i].Content
			}
			if cp.FirstKeptMessageId != "" {
				found := false
				for j, m := range msgs {
					if m.Id == cp.FirstKeptMessageId {
						startIndex = j
						found = true
						break
					}
				}
				if !found {
					startIndex = i + 1
				}
			} else {
				startIndex = i + 1
			}
			break
		}
	}
	if startIndex >= len(msgs) {
		return nil
	}

	keepBudget := DefaultKeepRecentTokens
	if cfg != nil && cfg.MaxPromptTokens > 0 && cfg.MaxPromptTokens/2 < keepBudget {
		keepBudget = cfg.MaxPromptTokens / 2
	}
	if keepBudget < 1024 {
		keepBudget = 1024
	}

	var userIndices []int
	for i := startIndex; i < len(msgs); i++ {
		if msgs[i].Role == RoleUser {
			userIndices = append(userIndices, i)
		}
	}

	if len(userIndices) == 0 {
		return nil
	}

	if len(userIndices) >= 2 {
		cutIndex := userIndices[len(userIndices)-1]
		if len(userIndices) >= 3 {
			prevUser := userIndices[len(userIndices)-2]
			if prevUser > startIndex {
				twoTurnsTokens := EstimateMessagesTokens(msgs[prevUser:])
				if twoTurnsTokens <= keepBudget {
					cutIndex = prevUser
				}
			}
		}
		if cutIndex <= startIndex {
			return nil
		}
		return &CutPointResult{
			StartIndex:         startIndex,
			CutIndex:           cutIndex,
			FirstKeptMessageId: msgs[cutIndex].Id,
			PreviousSummary:    prevSummary,
			IsSplitTurn:        false,
		}
	}

	// len(userIndices) == 1
	singleUserIdx := userIndices[0]
	if singleUserIdx > startIndex {
		return &CutPointResult{
			StartIndex:         startIndex,
			CutIndex:           singleUserIdx,
			FirstKeptMessageId: msgs[singleUserIdx].Id,
			PreviousSummary:    prevSummary,
			IsSplitTurn:        false,
		}
	}

	// Single turn exceeds budget -> split turn
	turnTokens := EstimateMessagesTokens(msgs[singleUserIdx:])
	if turnTokens > keepBudget {
		accum := 0
		asstCut := -1
		for i := len(msgs) - 1; i > singleUserIdx; i-- {
			accum += EstimateMessageTokens(msgs[i])
			if msgs[i].Role == RoleAssistant {
				asstCut = i
			}
			if accum >= keepBudget && asstCut > singleUserIdx {
				break
			}
		}
		if asstCut > singleUserIdx {
			return &CutPointResult{
				StartIndex:         startIndex,
				CutIndex:           asstCut,
				FirstKeptMessageId: msgs[asstCut].Id,
				PreviousSummary:    prevSummary,
				TurnPrefixMessages: msgs[singleUserIdx:asstCut],
				IsSplitTurn:        true,
			}
		}
	}

	return nil
}

// SerializeConversationForSummary transforms conversation messages into plain text
// with tool results truncated to ToolResultSummaryMaxChars.
func SerializeConversationForSummary(messages []models.AgentMessage) string {
	var parts []string
	for _, m := range messages {
		switch m.Role {
		case RoleUser:
			if text := strings.TrimSpace(m.Content); text != "" {
				parts = append(parts, fmt.Sprintf("[User]: %s", text))
			}
		case RoleAssistant:
			var obj map[string]any
			_ = json.Unmarshal([]byte(m.Payload), &obj)
			reasoning := ""
			if obj != nil {
				if r, ok := obj["reasoning"].(string); ok && strings.TrimSpace(r) != "" {
					reasoning = strings.TrimSpace(r)
				} else if r, ok := obj["reasoning_content"].(string); ok && strings.TrimSpace(r) != "" {
					reasoning = strings.TrimSpace(r)
				}
			}
			if reasoning != "" {
				parts = append(parts, fmt.Sprintf("[Assistant thinking]: %s", reasoning))
			}
			if text := strings.TrimSpace(m.Content); text != "" {
				parts = append(parts, fmt.Sprintf("[Assistant]: %s", text))
			}
			if obj != nil {
				calls := extractCalls(obj)
				if len(calls) > 0 {
					var callStrs []string
					for _, c := range calls {
						name := callName(c)
						fn, _ := c["function"].(map[string]any)
						args := ""
						if fn != nil {
							args, _ = fn["arguments"].(string)
						}
						callStrs = append(callStrs, fmt.Sprintf("%s(%s)", name, args))
					}
					parts = append(parts, fmt.Sprintf("[Assistant tool calls]: %s", strings.Join(callStrs, "; ")))
				}
			}
		case RoleTool:
			content := m.Content
			if len(content) > ToolResultSummaryMaxChars {
				truncated := len(content) - ToolResultSummaryMaxChars
				content = content[:ToolResultSummaryMaxChars] + fmt.Sprintf("\n\n[... %d more characters truncated]", truncated)
			}
			parts = append(parts, fmt.Sprintf("[Tool result]: %s", content))
		case RoleEvent:
			switch m.EventType {
			case EventConfirmRequired:
				parts = append(parts, fmt.Sprintf("[Event confirm_required]: %s", m.ToolName))
			case EventApplied:
				parts = append(parts, fmt.Sprintf("[Event applied]: %s", m.ToolName))
			case EventRejected:
				parts = append(parts, fmt.Sprintf("[Event rejected]: %s", m.ToolName))
			case EventUndone:
				parts = append(parts, fmt.Sprintf("[Event undone]: %s", m.ToolName))
			}
		}
	}
	return strings.Join(parts, "\n\n")
}

// GenerateSummary prompts the LLM to generate or update structured compaction summary.
func GenerateSummary(ctx context.Context, chat ChatFn, baseURL, apiKey, model string, conversationText, previousSummary string) (string, error) {
	if chat == nil {
		return "", fmt.Errorf("llm client not configured")
	}
	systemMsg := map[string]any{
		"role":    "system",
		"content": SummarizationSystemPrompt,
	}
	var userPrompt string
	if strings.TrimSpace(previousSummary) != "" {
		userPrompt = fmt.Sprintf("<previous-summary>\n%s\n</previous-summary>\n\nThe conversation messages above are NEW conversation messages to incorporate into the existing summary:\n\n%s\n\n%s",
			previousSummary, conversationText, UpdateSummaryPrompt)
	} else {
		userPrompt = fmt.Sprintf("Conversation messages to summarize:\n\n%s\n\n%s", conversationText, InitialSummaryPrompt)
	}
	userMsg := map[string]any{
		"role":    "user",
		"content": userPrompt,
	}
	t := 0.0
	req := client.CompletionsRequest{
		Model:       model,
		Messages:    []json.RawMessage{mustRaw(systemMsg), mustRaw(userMsg)},
		Temperature: &t,
		MaxTokens:   2048,
		User:        "compaction",
	}
	resp, err := chat(ctx, baseURL, apiKey, req)
	if err != nil {
		return "", err
	}
	if resp == nil || len(resp.Choices) == 0 {
		return "", fmt.Errorf("empty summarization response")
	}
	summary := strings.TrimSpace(resp.Choices[0].Message.Content)
	if summary == "" {
		return "", fmt.Errorf("empty summary content")
	}
	return summary, nil
}

// CompactConversation executes the compaction cycle: finds cut point, calls LLM for summary,
// appends compaction event message, and updates conversation prompt tokens.
func CompactConversation(ctx context.Context, store Store, conv *models.AgentConversation, cfg *models.AgentUserSettings, chat ChatFn, apiKey string, sink EventSink) (*models.AgentMessage, error) {
	msgs, err := store.ListMessages(conv.Id)
	if err != nil {
		return nil, err
	}
	cut := FindCutPoint(msgs, cfg)
	if cut == nil || cut.CutIndex <= cut.StartIndex {
		return nil, nil
	}
	msgsToSummarize := msgs[cut.StartIndex:cut.CutIndex]
	if len(msgsToSummarize) == 0 {
		return nil, nil
	}
	tokensBefore := conv.LastPromptTokens
	if tokensBefore <= 0 {
		tokensBefore = EstimateMessagesTokens(msgs)
	}

	convText := SerializeConversationForSummary(msgsToSummarize)
	if len(cut.TurnPrefixMessages) > 0 {
		convText += "\n\n[Current turn prefix]:\n\n" + SerializeConversationForSummary(cut.TurnPrefixMessages)
	}

	summary, err := GenerateSummary(ctx, chat, cfg.BaseURL, apiKey, cfg.Model, convText, cut.PreviousSummary)
	if err != nil {
		return nil, err
	}

	firstKeptId := msgs[cut.CutIndex].Id
	payloadRaw, _ := json.Marshal(CompactionPayload{
		Summary:            summary,
		FirstKeptMessageId: firstKeptId,
		TokensBefore:       tokensBefore,
	})

	compactMsg := &models.AgentMessage{
		Id:             newHexID(),
		ConversationId: conv.Id,
		Role:           RoleEvent,
		EventType:      EventCompaction,
		Payload:        string(payloadRaw),
		Content:        summary,
		CreateId:       conv.CreateId,
		CreateTime:     models.NewDateTime(),
	}
	if err := store.SaveMessage(compactMsg); err != nil {
		return nil, err
	}

	keptMsgs := msgs[cut.CutIndex:]
	conv.LastPromptTokens = (len(summary)+3)/4 + EstimateMessagesTokens(keptMsgs)
	_ = store.SaveConversation(conv)

	if sink != nil {
		sink.Emit("message", MessageView(compactMsg, ""))
	}
	return compactMsg, nil
}
