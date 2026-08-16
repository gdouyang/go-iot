package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"go-iot/pkg/agent/client"
	"go-iot/pkg/models"
)

type ChatFn func(ctx context.Context, baseURL, apiKey string, req client.CompletionsRequest) (*client.CompletionsResponse, error)

var DefaultChat ChatFn

func init() {
	c := client.New()
	DefaultChat = c.Chat
}

type runLocks struct {
	mu sync.Mutex
	m  map[string]struct{}
}

func (l *runLocks) Try(id string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.m == nil {
		l.m = map[string]struct{}{}
	}
	if _, ok := l.m[id]; ok {
		return false
	}
	l.m[id] = struct{}{}
	return true
}

func (l *runLocks) Unlock(id string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.m, id)
}

var locks = &runLocks{}

type RunResult struct {
	ConversationId string   `json:"conversationId"`
	RunStatus      string   `json:"runStatus"`
	PendingDrafts  []string `json:"pendingDrafts"`
	NeedsResume    bool     `json:"needsResume"`
}

func TryLock(convId string) bool { return locks.Try(convId) }
func Unlock(convId string)       { locks.Unlock(convId) }

func RunTurn(ctx context.Context, store Store, conv *models.AgentConversation, settings *models.AgentUserSettings, userText string, continueRun bool) (*RunResult, error) {
	if !locks.Try(conv.Id) {
		return nil, fmt.Errorf("%s", ReasonRunActive)
	}
	defer locks.Unlock(conv.Id)

	cfg := Effective(settings)
	now := models.NewDateTime()
	if !continueRun {
		userPayload, _ := json.Marshal(map[string]any{
			"role": "user", "content": userText,
		})
		_ = store.SaveMessage(&models.AgentMessage{
			Id: newHexID(), ConversationId: conv.Id, Role: RoleUser,
			Payload: string(userPayload), Content: userText,
			CreateId: conv.CreateId, CreateTime: now,
		})
		if IsDefaultTitle(conv.Title) {
			conv.Title = ResolveTitle(ctx, store, conv, settings, userText)
		}
	}
	conv.RunStatus = RunRunning
	conv.NeedsResume = false
	_ = store.SaveConversation(conv)

	exec := ToolExec{UserId: conv.CreateId}
	chat := DefaultChat
	if chat == nil {
		return nil, fmt.Errorf("llm client not configured")
	}

	for turn := 0; turn < cfg.MaxTurns; turn++ {
		msgs, err := store.ListMessages(conv.Id)
		if err != nil {
			return nil, err
		}
		input := append([]json.RawMessage{SystemPromptMessage()}, ToInput(msgs)...)
		temp := cfg.Temperature
		effort := cfg.ReasoningEffort
		parallel := false
		resp, err := chat(ctx, cfg.BaseURL, settings.ApiKey, client.CompletionsRequest{
			Model:             cfg.Model,
			Messages:          input,
			Tools:             ToolCatalog(),
			Temperature:       temp,
			MaxTokens:         cfg.MaxTokens,
			ParallelToolCalls: &parallel,
			ReasoningEffort:   effort,
			User:              strconv.FormatInt(conv.CreateId, 10),
		})
		if err != nil {
			if client.IsToolsNotSupported(err) {
				return nil, fmt.Errorf("%s: %v", ReasonToolsNotSupported, err)
			}
			conv.RunStatus = RunIdle
			if pendingLeft(store, conv.Id) {
				conv.RunStatus = RunAwaitingConfirm
			}
			_ = store.SaveConversation(conv)
			return nil, err
		}
		if len(resp.Choices) == 0 {
			break
		}
		msg := resp.Choices[0].Message
		msg.ToolCalls = ensureToolCalls(msg.ToolCalls)
		if willPauseForConfirm(cfg.WriteMode, conv.CreateId, msg.ToolCalls) {
			msg.Content = ""
		}
		_ = store.SaveMessage(&models.AgentMessage{
			Id: newHexID(), ConversationId: conv.Id, Role: RoleAssistant,
			Payload: assistantPayload(msg.Content, msg.ToolCalls), Content: msg.Content,
			CreateId: conv.CreateId, CreateTime: models.NewDateTime(),
		})
		if len(msg.ToolCalls) == 0 {
			break
		}
		stop := false
		for _, tc := range msg.ToolCalls {
			name := tc.Function.Name
			args := json.RawMessage(tc.Function.Arguments)
			gate := BeforeToolCall(cfg.WriteMode, conv.CreateId, name, args)
			switch gate.Action {
			case ToolConfirm:
				preview, _ := json.Marshal(map[string]any{"old": nil, "new": json.RawMessage(args), "truncated": false})
				_, _ = QueueConfirmDraft(store, conv.CreateId, conv, tc.Id, name, string(args), string(preview), "")
				if gate.Terminate {
					stop = true
				}
			case ToolBlock:
				content, _ := json.Marshal(map[string]any{"ok": false, "message": gate.Reason})
				_ = store.SaveMessage(&models.AgentMessage{
					Id: newHexID(), ConversationId: conv.Id, Role: RoleTool,
					Payload: toolPayload(tc.Id, name, string(content)), Content: string(content), ToolName: name,
					ToolCallId: tc.Id, CreateId: conv.CreateId, CreateTime: models.NewDateTime(),
				})
				if gate.Terminate {
					stop = true
				}
			default:
				if IsMutatingTool(name) {
					pid, err := ApplyMutating(conv.CreateId, name, string(args))
					content, _ := json.Marshal(map[string]any{"ok": err == nil, "productId": pid, "error": errString(err)})
					_ = store.SaveMessage(&models.AgentMessage{
						Id: newHexID(), ConversationId: conv.Id, Role: RoleTool,
						Payload: toolPayload(tc.Id, name, string(content)), Content: string(content), ToolName: name,
						ToolCallId: tc.Id, ProductId: pid, CreateId: conv.CreateId, CreateTime: models.NewDateTime(),
					})
					continue
				}
				out, err := exec.Execute(name, args)
				if err != nil {
					out = map[string]any{"ok": false, "message": err.Error()}
				}
				content, _ := json.Marshal(out)
				_ = store.SaveMessage(&models.AgentMessage{
					Id: newHexID(), ConversationId: conv.Id, Role: RoleTool,
					Payload: toolPayload(tc.Id, name, string(content)), Content: string(content), ToolName: name,
					ToolCallId: tc.Id, CreateId: conv.CreateId, CreateTime: models.NewDateTime(),
				})
			}
		}
		if stop {
			break
		}
	}

	if pendingLeft(store, conv.Id) {
		conv.RunStatus = RunAwaitingConfirm
	} else if conv.RunStatus == RunRunning {
		conv.RunStatus = RunIdle
	}
	_ = store.SaveConversation(conv)
	return snapshotResult(store, conv), nil
}

func PendingLeft(store Store, convId string) bool {
	return pendingLeft(store, convId)
}

func pendingLeft(store Store, convId string) bool {
	all, _ := store.ListDrafts(convId)
	return len(PendingDrafts(all)) > 0
}

func snapshotResult(store Store, conv *models.AgentConversation) *RunResult {
	all, _ := store.ListDrafts(conv.Id)
	var ids []string
	for _, d := range PendingDrafts(all) {
		ids = append(ids, d.Id)
	}
	return &RunResult{
		ConversationId: conv.Id,
		RunStatus:      conv.RunStatus,
		PendingDrafts:  ids,
		NeedsResume:    conv.NeedsResume,
	}
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func ensureToolCalls(calls []client.ToolCall) []client.ToolCall {
	for i := range calls {
		if strings.TrimSpace(calls[i].Id) == "" {
			calls[i].Id = "call_" + newHexID()
		}
		if calls[i].Type == "" {
			calls[i].Type = "function"
		}
	}
	return calls
}

func assistantPayload(content string, calls []client.ToolCall) string {
	m := map[string]any{"role": RoleAssistant, "content": content}
	if len(calls) > 0 {
		m["tool_calls"] = calls
	}
	raw, _ := json.Marshal(m)
	return string(raw)
}

func willPauseForConfirm(writeMode string, userId int64, calls []client.ToolCall) bool {
	for _, tc := range calls {
		if BeforeToolCall(writeMode, userId, tc.Function.Name, json.RawMessage(tc.Function.Arguments)).Action == ToolConfirm {
			return true
		}
	}
	return false
}

func toolPayload(callId, name, content string) string {
	m := map[string]any{"role": RoleTool, "tool_call_id": callId, "content": content}
	if name != "" {
		m["name"] = name
	}
	raw, _ := json.Marshal(m)
	return string(raw)
}
