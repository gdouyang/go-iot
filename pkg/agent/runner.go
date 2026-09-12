package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"go-iot/pkg/agent/client"
	logs "go-iot/pkg/logger"
	"go-iot/pkg/models"
	"go-iot/pkg/redis"
)

type ChatFn func(ctx context.Context, baseURL, apiKey string, req client.CompletionsRequest) (*client.CompletionsResponse, error)
type ChatStreamFn func(ctx context.Context, baseURL, apiKey string, req client.CompletionsRequest, onDelta func(client.StreamDelta)) (*client.CompletionsResponse, error)

var DefaultChat ChatFn
var DefaultChatStream ChatStreamFn

func init() {
	c := client.New()
	DefaultChat = c.Chat
	DefaultChatStream = c.ChatStream
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

var (
	locks      = &runLocks{}
	redisLocks sync.Map // convId -> *redis.Lock
)

// TryLock 尝试对指定会话加互斥锁。优先使用 Redis 分布式锁（自动续期），并协同本地内存锁。
func TryLock(id string) bool {
	if client := redis.GetRedisClient(); client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		lock, ok, err := redis.TryAcquireLock(ctx, ConvRedisKey(id, "lock"), 60*time.Second, redis.WithAutoRefresh(true))
		if err != nil || !ok {
			return false
		}
		if !locks.Try(id) {
			_ = lock.Unlock(context.Background())
			return false
		}
		redisLocks.Store(id, lock)
		return true
	}
	return locks.Try(id)
}

// Unlock 释放指定会话的互斥锁。
func Unlock(id string) {
	if v, ok := redisLocks.LoadAndDelete(id); ok {
		if lk, _ := v.(*redis.Lock); lk != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = lk.Unlock(ctx)
		}
	}
	locks.Unlock(id)
}

type RunResult struct {
	ConversationId string   `json:"conversationId"`
	RunStatus      string   `json:"runStatus"`
	PendingDrafts  []string `json:"pendingDrafts"`
	NeedsResume    bool     `json:"needsResume"`
}

type RunOpts struct {
	Store    Store
	Conv     *models.AgentConversation
	Settings *models.AgentUserSettings
	UserText string
	Continue bool
	Sink     EventSink
	Perms    map[string]bool
}

func RunTurn(ctx context.Context, store Store, conv *models.AgentConversation, settings *models.AgentUserSettings, userText string, continueRun bool) (*RunResult, error) {
	return RunTurnOpts(ctx, RunOpts{
		Store: store, Conv: conv, Settings: settings, UserText: userText, Continue: continueRun,
	})
}

func RunTurnOpts(ctx context.Context, opts RunOpts) (*RunResult, error) {
	store := opts.Store
	conv := opts.Conv
	settings := opts.Settings
	sink := opts.Sink
	if sink == nil {
		sink = NopSink{}
	}
	if !TryLock(conv.Id) {
		return nil, fmt.Errorf("%s", ReasonRunActive)
	}
	defer Unlock(conv.Id)

	cfg := Effective(settings)
	runTimeout := time.Duration(cfg.RunTimeoutSeconds) * time.Second
	if runTimeout <= 0 {
		runTimeout = time.Duration(DefaultRunTimeoutSeconds) * time.Second
	}
	runCtx, cancel := context.WithTimeout(ctx, runTimeout)
	defer cancel()
	slot := registerRun(conv.Id, cancel)
	defer finishRun(conv.Id, slot)

	EnsureConvSeq(runCtx, conv.Id, store)

	now := models.NewDateTime()
	if !opts.Continue {
		userPayload, _ := json.Marshal(map[string]any{"role": "user", "content": opts.UserText})
		um := &models.AgentMessage{
			Id: newHexID(), ConversationId: conv.Id, Role: RoleUser,
			Payload: string(userPayload), Content: opts.UserText,
			CreateId: conv.CreateId, CreateTime: now,
		}
		_ = store.SaveMessage(um)
		sink.Emit("message", MessageView(um, ""))
		if IsDefaultTitle(conv.Title) {
			go resolveTitleAsync(store, conv.Id, settings, opts.UserText, sink)
		}
	}
	conv.RunStatus = RunRunning
	conv.NeedsResume = false
	_ = store.SaveConversation(conv)
	sink.Emit("status", snapshotResult(store, conv))

	stopHB := startHeartbeat(store, conv, defaultHeartbeatInterval)
	defer stopHB()

	toolCtx := ToolContext{UserId: conv.CreateId, Perms: opts.Perms}
	exec := ToolExec{Ctx: toolCtx}
	chat := DefaultChat
	stream := DefaultChatStream
	apiKey := ""
	if settings != nil {
		apiKey = RevealAPIKey(settings.ApiKey)
	}

	callTimeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if callTimeout <= 0 {
		callTimeout = time.Duration(DefaultTimeoutSeconds) * time.Second
	}

	for turn := 0; turn < cfg.MaxTurns; turn++ {
		if err := runCtx.Err(); err != nil {
			break
		}
		msgs, err := store.ListMessages(conv.Id)
		if err != nil {
			return nil, err
		}

		if ShouldCompact(conv, msgs, cfg) && !pendingLeft(store, conv.Id) {
			sink.Emit("status", map[string]any{"runStatus": RunCompacting, "conversationId": conv.Id})
			if _, err := CompactConversation(runCtx, store, conv, cfg, chat, apiKey, sink); err != nil {
				logs.Warnf("agent compaction failed conv=%s: %v", conv.Id, err)
			} else {
				msgs, err = store.ListMessages(conv.Id)
				if err != nil {
					return nil, err
				}
			}
			sink.Emit("status", map[string]any{"runStatus": RunRunning, "conversationId": conv.Id})
		}

		input := append([]json.RawMessage{systemPromptMessage(conv.ProductId)}, CompactAndTrim(ToInput(msgs), cfg.MaxPromptTokens)...)
		temp := cfg.Temperature
		effort := cfg.ReasoningEffort
		parallel := false
		req := client.CompletionsRequest{
			Model:             cfg.Model,
			Messages:          input,
			Tools:             ToolCatalog(),
			Temperature:       temp,
			MaxTokens:         cfg.MaxTokens,
			ParallelToolCalls: &parallel,
			ReasoningEffort:   effort,
			User:              strconv.FormatInt(conv.CreateId, 10),
		}
		callCtx, cancelCall := context.WithTimeout(runCtx, callTimeout)
		resp, err := chatOnce(callCtx, stream, chat, cfg.BaseURL, apiKey, req, func(delta client.StreamDelta) {
			if delta.Reasoning != "" {
				sink.Emit("reasoning", map[string]any{"content": delta.Reasoning})
			}
			if delta.Content != "" {
				sink.Emit("delta", map[string]any{"content": delta.Content})
			}
		})
		cancelCall()
		if err != nil {
			if runCtx.Err() != nil {
				logs.Warnf("agent run turn canceled or timed out conv=%s turn=%d: %v", conv.Id, turn, runCtx.Err())
				break
			}
			logs.Errorf("agent run turn llm error conv=%s turn=%d model=%s: %v", conv.Id, turn, req.Model, err)
			if client.IsToolsNotSupported(err) {
				failRun(store, conv, sink)
				return nil, fmt.Errorf("%s: %v", ReasonToolsNotSupported, err)
			}
			failRun(store, conv, sink)
			return nil, err
		}
		if resp == nil || len(resp.Choices) == 0 {
			break
		}
		if resp.Usage.PromptTokens > 0 {
			conv.LastPromptTokens = resp.Usage.PromptTokens
			conv.LastCompletionTokens = resp.Usage.CompletionTokens
		}
		if n := resp.Usage.ReasoningTokenCount(); n > 0 {
			conv.LastReasoningTokens = n
		}
		msg := resp.Choices[0].Message
		msg.FillReasoning()
		msg.ToolCalls = ensureToolCalls(msg.ToolCalls)
		if willPauseForConfirm(cfg.WriteMode, toolCtx, msg.ToolCalls) {
			msg.Content = ""
		}
		asst := &models.AgentMessage{
			Id: newHexID(), ConversationId: conv.Id, Role: RoleAssistant,
			Payload: assistantPayload(msg.Content, msg.ToolCalls, msg.Reasoning), Content: msg.Content,
			CreateId: conv.CreateId, CreateTime: models.NewDateTime(),
		}
		_ = store.SaveMessage(asst)
		if msg.Content != "" || strings.TrimSpace(msg.Reasoning) != "" {
			sink.Emit("message", MessageView(asst, ""))
		}
		if len(msg.ToolCalls) == 0 {
			break
		}
		stop := false
		for _, tc := range msg.ToolCalls {
			name := tc.Function.Name
			args := json.RawMessage(tc.Function.Arguments)
			gate := BeforeToolCall(cfg.WriteMode, toolCtx, name, args)
			switch gate.Action {
			case ToolConfirm:
				preview, pid := BuildPreview(toolCtx, name, args)
				d, _ := QueueConfirmDraft(store, toolCtx.UserId, conv, tc.Id, name, string(args), preview, pid, cfg.DraftTTLHours)
				if d != nil {
					ev := &models.AgentMessage{}
					if msgs, err := store.ListMessages(conv.Id); err == nil {
						for i := len(msgs) - 1; i >= 0; i-- {
							if msgs[i].DraftId == d.Id && msgs[i].EventType == EventConfirmRequired {
								ev = &msgs[i]
								break
							}
						}
					}
					sink.Emit("message", MessageView(ev, d.Status))
				}
				if gate.Terminate {
					stop = true
				}
			case ToolBlock:
				content, _ := json.Marshal(map[string]any{"ok": false, "message": gate.Reason})
				tm := &models.AgentMessage{
					Id: newHexID(), ConversationId: conv.Id, Role: RoleTool,
					Payload: toolPayload(tc.Id, name, string(content)), Content: string(content), ToolName: name,
					ToolCallId: tc.Id, CreateId: conv.CreateId, CreateTime: models.NewDateTime(),
				}
				_ = store.SaveMessage(tm)
				sink.Emit("message", MessageView(tm, ""))
				if gate.Terminate {
					stop = true
				}
			default:
				if IsMutatingTool(name) {
					pid, err := ApplyMutating(toolCtx, name, string(args))
					content, _ := json.Marshal(map[string]any{"ok": err == nil, "productId": pid, "error": errString(err)})
					tm := &models.AgentMessage{
						Id: newHexID(), ConversationId: conv.Id, Role: RoleTool,
						Payload: toolPayload(tc.Id, name, string(content)), Content: string(content), ToolName: name,
						ToolCallId: tc.Id, ProductId: pid, CreateId: conv.CreateId, CreateTime: models.NewDateTime(),
					}
					_ = store.SaveMessage(tm)
					_ = store.SaveAudit(&models.AgentAudit{
						Id: newHexID(), ConversationId: conv.Id, ToolName: name, ProductId: pid,
						Action: "auto", Success: err == nil, Error: errString(err),
						CreateId: conv.CreateId, CreateTime: models.NewDateTime(),
					})
					if pid != "" && conv.ProductId == "" {
						conv.ProductId = pid
					}
					sink.Emit("message", MessageView(tm, ""))
					continue
				}
				out, err := exec.Execute(name, args)
				if err != nil {
					out = map[string]any{"ok": false, "message": err.Error()}
				}
				content, _ := json.Marshal(out)
				tm := &models.AgentMessage{
					Id: newHexID(), ConversationId: conv.Id, Role: RoleTool,
					Payload: toolPayload(tc.Id, name, string(content)), Content: string(content), ToolName: name,
					ToolCallId: tc.Id, CreateId: conv.CreateId, CreateTime: models.NewDateTime(),
				}
				_ = store.SaveMessage(tm)
				sink.Emit("message", MessageView(tm, ""))
			}
		}
		if stop {
			break
		}
	}

	if pendingLeft(store, conv.Id) {
		conv.RunStatus = RunAwaitingConfirm
	} else {
		conv.RunStatus = RunIdle
	}
	PurgeSettledDrafts(store, conv.Id)
	_ = store.SaveConversation(conv)
	res := snapshotResult(store, conv)
	sink.Emit("status", res)
	sink.Emit("done", res)
	return res, nil
}

func chatOnce(ctx context.Context, stream ChatStreamFn, chat ChatFn, baseURL, apiKey string, req client.CompletionsRequest, onDelta func(client.StreamDelta)) (*client.CompletionsResponse, error) {
	if stream != nil {
		resp, err := stream(ctx, baseURL, apiKey, req, onDelta)
		if err == nil {
			return resp, nil
		}
		if ctx.Err() != nil {
			return nil, err
		}
		if he, ok := err.(*client.HTTPError); ok && (he.Status == http.StatusUnauthorized || he.Status == http.StatusForbidden) {
			return nil, err
		}
		logs.Warnf("agent chat stream failed, fallback to non-stream: %v", err)
	}
	if chat == nil {
		return nil, fmt.Errorf("llm client not configured")
	}
	resp, err := chat(ctx, baseURL, apiKey, req)
	if err != nil {
		return nil, err
	}
	if resp != nil && len(resp.Choices) > 0 {
		resp.Choices[0].Message.FillReasoning()
		if onDelta != nil {
			if resp.Choices[0].Message.Reasoning != "" {
				onDelta(client.StreamDelta{Reasoning: resp.Choices[0].Message.Reasoning})
			}
			if resp.Choices[0].Message.Content != "" {
				onDelta(client.StreamDelta{Content: resp.Choices[0].Message.Content})
			}
		}
	}
	return resp, nil
}

func failRun(store Store, conv *models.AgentConversation, sink EventSink) {
	if pendingLeft(store, conv.Id) {
		conv.RunStatus = RunAwaitingConfirm
	} else {
		conv.RunStatus = RunIdle
	}
	_ = store.SaveConversation(conv)
	if sink != nil {
		sink.Emit("status", snapshotResult(store, conv))
	}
}

func startHeartbeat(store Store, conv *models.AgentConversation, interval time.Duration) func() {
	if interval <= 0 {
		interval = defaultHeartbeatInterval
	}
	done := make(chan struct{})
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-done:
				return
			case <-t.C:
				// 只刷 UpdateTime（活着的心跳），不整份回写，避免覆盖 run 刚写入的状态/计数
				_ = store.TouchConversation(conv.Id)
			}
		}
	}()
	return func() { close(done) }
}

func resolveTitleAsync(store Store, convId string, settings *models.AgentUserSettings, userText string, sink EventSink) {
	ctx, cancel := context.WithTimeout(context.Background(), titleTimeout)
	defer cancel()
	c, err := store.GetConversation(convId)
	if err != nil || c == nil || !IsDefaultTitle(c.Title) {
		return
	}
	title := ResolveTitle(ctx, store, c, settings, userText)
	c2, err := store.GetConversation(convId)
	if err != nil || c2 == nil || !IsDefaultTitle(c2.Title) {
		return
	}
	_ = store.UpdateTitle(convId, title)
	sink.Emit("title", map[string]any{"id": convId, "title": title})
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

func orJSON(s string) string {
	if strings.TrimSpace(s) == "" {
		return "null"
	}
	return s
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

func assistantPayload(content string, calls []client.ToolCall, reasoning string) string {
	m := map[string]any{"role": RoleAssistant, "content": content}
	if len(calls) > 0 {
		m["tool_calls"] = calls
	}
	if strings.TrimSpace(reasoning) != "" {
		m["reasoning"] = reasoning
	}
	raw, _ := json.Marshal(m)
	return string(raw)
}

func willPauseForConfirm(writeMode string, ctx ToolContext, calls []client.ToolCall) bool {
	for _, tc := range calls {
		if BeforeToolCall(writeMode, ctx, tc.Function.Name, json.RawMessage(tc.Function.Arguments)).Action == ToolConfirm {
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
