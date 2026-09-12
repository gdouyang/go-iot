package agent

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"go-iot/pkg/agent/client"
	"go-iot/pkg/models"

	"github.com/stretchr/testify/require"
)

func TestShouldCompactThreshold(t *testing.T) {
	cfg := Effective(&models.AgentUserSettings{
		Model:           "deepseek-flash",
		MaxPromptTokens: 32000,
		MaxTokens:       16384,
	})
	// MaxPromptTokens = 32000 -> usable = 32000 -> threshold = 32000 * 0.8 = 25600
	cfgDef := Effective(&models.AgentUserSettings{})
	require.Equal(t, 200000, cfgDef.MaxPromptTokens)

	msgs := []models.AgentMessage{
		{Id: "1", Role: RoleUser, Content: "hi", Payload: `{"role":"user","content":"hi"}`},
		{Id: "2", Role: RoleAssistant, Content: "hello", Payload: `{"role":"assistant","content":"hello"}`},
	}

	// Below 80%: 20000 tokens < 25600 threshold -> false
	convLow := &models.AgentConversation{
		Id:               "c1",
		LastPromptTokens: 20000,
	}
	require.False(t, ShouldCompact(convLow, msgs, cfg), "below 80% should not trigger compaction")

	// Above 80%: 26000 tokens > 25600 threshold -> true
	convHigh := &models.AgentConversation{
		Id:               "c1",
		LastPromptTokens: 26000,
	}
	require.True(t, ShouldCompact(convHigh, msgs, cfg), "above 80% should trigger compaction")

	// When LastPromptTokens is 0, estimate from msgs:
	// If message tokens exceed threshold -> true
	longContent := strings.Repeat("test content for token estimation ", 3000) // ~100k chars -> ~25k tokens
	bigMsgs := []models.AgentMessage{
		{Id: "1", Role: RoleUser, Content: longContent, Payload: longContent},
		{Id: "2", Role: RoleAssistant, Content: longContent, Payload: longContent},
	}
	convZero := &models.AgentConversation{
		Id:               "c1",
		LastPromptTokens: 0,
	}
	require.True(t, ShouldCompact(convZero, bigMsgs, cfg), "estimated tokens above 80% should trigger compaction")

	// len(msgs) <= 1 should never compact
	require.False(t, ShouldCompact(convHigh, msgs[:1], cfg))
}

func TestFindCutPointDoesNotSplitToolCallAndResult(t *testing.T) {
	cfg := Effective(&models.AgentUserSettings{
		MaxPromptTokens: 32000,
	})

	msgs := []models.AgentMessage{
		// Turn 1
		{Id: "u1", Role: RoleUser, Content: "查询产品列表", Payload: `{"role":"user","content":"查询产品列表"}`},
		{Id: "a1", Role: RoleAssistant, Content: "", Payload: `{"role":"assistant","content":"","tool_calls":[{"id":"call_1","type":"function","function":{"name":"list_products","arguments":"{}"}}]}`},
		{Id: "t1", Role: RoleTool, Content: `{"list":[]}`, ToolName: "list_products", ToolCallId: "call_1", Payload: `{"role":"tool","tool_call_id":"call_1","name":"list_products","content":"{\"list\":[]}"}`},
		// Turn 2
		{Id: "u2", Role: RoleUser, Content: "读取产品详情", Payload: `{"role":"user","content":"读取产品详情"}`},
		{Id: "a2", Role: RoleAssistant, Content: "", Payload: `{"role":"assistant","content":"","tool_calls":[{"id":"call_2","type":"function","function":{"name":"get_product","arguments":"{\"id\":\"DEV01\"}"}}]}`},
		{Id: "t2", Role: RoleTool, Content: `{"id":"DEV01"}`, ToolName: "get_product", ToolCallId: "call_2", Payload: `{"role":"tool","tool_call_id":"call_2","name":"get_product","content":"{\"id\":\"DEV01\"}"}`},
	}

	cut := FindCutPoint(msgs, cfg)
	require.NotNil(t, cut)
	// Cut point must land cleanly on a user message
	require.Equal(t, "u2", cut.FirstKeptMessageId)
	require.Equal(t, 3, cut.CutIndex) // index of u2
	require.False(t, cut.IsSplitTurn)

	// Verify messages to summarize contain full Turn 1 (User, Assistant with tool_call, Tool result)
	summarized := msgs[cut.StartIndex:cut.CutIndex]
	require.Len(t, summarized, 3)
	require.Equal(t, "u1", summarized[0].Id)
	require.Equal(t, "a1", summarized[1].Id)
	require.Equal(t, "t1", summarized[2].Id)

	// Verify kept messages contain full Turn 2 (User, Assistant with tool_call, Tool result)
	kept := msgs[cut.CutIndex:]
	require.Len(t, kept, 3)
	require.Equal(t, "u2", kept[0].Id)
	require.Equal(t, "a2", kept[1].Id)
	require.Equal(t, "t2", kept[2].Id)
}

func TestFindCutPointSplitTurn(t *testing.T) {
	// A single turn that exceeds budget should cut at an assistant message, not tool result
	cfg := Effective(&models.AgentUserSettings{
		MaxPromptTokens: 2048, // budget / 2 = 1024
	})

	bigPayload := strings.Repeat("x", 4000)
	msgs := []models.AgentMessage{
		{Id: "u1", Role: RoleUser, Content: "开始", Payload: `{"role":"user","content":"开始"}`},
		{Id: "a1", Role: RoleAssistant, Content: "", Payload: `{"role":"assistant","content":"","tool_calls":[{"id":"c1","type":"function","function":{"name":"step1","arguments":"{}"}}]}`},
		{Id: "t1", Role: RoleTool, Content: bigPayload, ToolName: "step1", ToolCallId: "c1", Payload: `{"role":"tool","tool_call_id":"c1","name":"step1","content":"` + bigPayload + `"}`},
		{Id: "a2", Role: RoleAssistant, Content: "继续中", Payload: `{"role":"assistant","content":"继续中","tool_calls":[{"id":"c2","type":"function","function":{"name":"step2","arguments":"{}"}}]}`},
		{Id: "t2", Role: RoleTool, Content: "ok", ToolName: "step2", ToolCallId: "c2", Payload: `{"role":"tool","tool_call_id":"c2","name":"step2","content":"ok"}`},
	}

	cut := FindCutPoint(msgs, cfg)
	require.NotNil(t, cut)
	require.True(t, cut.IsSplitTurn)
	// Must cut at assistant message a2, NEVER at tool result t1 or t2
	require.Equal(t, "a2", cut.FirstKeptMessageId)
	require.Equal(t, RoleAssistant, msgs[cut.CutIndex].Role)
}

func TestCompactConversationPreservesOriginalMessages(t *testing.T) {
	store := NewMemoryStore()
	conv := &models.AgentConversation{
		Id:               "test-conv-preserve",
		CreateId:         1,
		LastPromptTokens: 26000,
	}
	_ = store.SaveConversation(conv)

	origMsgs := []models.AgentMessage{
		{Id: "m1", ConversationId: conv.Id, Role: RoleUser, Content: "创建MQTT产品", Payload: `{"role":"user","content":"创建MQTT产品"}`},
		{Id: "m2", ConversationId: conv.Id, Role: RoleAssistant, Content: "好的", Payload: `{"role":"assistant","content":"好的"}`},
		{Id: "m3", ConversationId: conv.Id, Role: RoleUser, Content: "保存物模型", Payload: `{"role":"user","content":"保存物模型"}`},
		{Id: "m4", ConversationId: conv.Id, Role: RoleAssistant, Content: "物模型已保存", Payload: `{"role":"assistant","content":"物模型已保存"}`},
	}
	for i := range origMsgs {
		_ = store.SaveMessage(&origMsgs[i])
	}

	cfg := Effective(&models.AgentUserSettings{
		Model:           "deepseek-flash",
		MaxPromptTokens: 32000,
	})

	mockSummary := "## Goal\n创建MQTT产品\n## Product\n- productId: TEST\n## Config\n## TSL / Script / Collector\n## Progress / Decisions / Blocked / Next\n- Progress: 创建完成"
	mockChat := func(ctx context.Context, baseURL, apiKey string, req client.CompletionsRequest) (*client.CompletionsResponse, error) {
		resp := &client.CompletionsResponse{}
		resp.Choices = []struct {
			Message client.CompletionsMessage `json:"message"`
		}{
			{Message: client.CompletionsMessage{Role: RoleAssistant, Content: mockSummary}},
		}
		return resp, nil
	}

	compactMsg, err := CompactConversation(context.Background(), store, conv, cfg, mockChat, "", nil)
	require.NoError(t, err)
	require.NotNil(t, compactMsg)
	require.Equal(t, EventCompaction, compactMsg.EventType)
	require.Equal(t, RoleEvent, compactMsg.Role)
	require.Equal(t, mockSummary, compactMsg.Content)

	// Verify all original messages are STILL intact in store
	allMsgs, err := store.ListMessages(conv.Id)
	require.NoError(t, err)
	require.Len(t, allMsgs, 5) // 4 original + 1 compaction event

	foundCompaction := false
	for _, m := range allMsgs {
		if m.EventType == EventCompaction {
			foundCompaction = true
			var cp CompactionPayload
			require.NoError(t, json.Unmarshal([]byte(m.Payload), &cp))
			require.Equal(t, "m3", cp.FirstKeptMessageId)
			require.Equal(t, mockSummary, cp.Summary)
		}
	}
	require.True(t, foundCompaction)

	// Check original messages were not mutated
	m1, _ := store.ListMessages(conv.Id)
	require.Equal(t, "创建MQTT产品", m1[0].Content)
	require.Equal(t, `{"role":"user","content":"创建MQTT产品"}`, m1[0].Payload)
}

func TestSecondCompactionConsumesPreviousSummary(t *testing.T) {
	store := NewMemoryStore()
	conv := &models.AgentConversation{
		Id:               "test-conv-second",
		CreateId:         1,
		LastPromptTokens: 26000,
	}
	_ = store.SaveConversation(conv)

	msgs1 := []models.AgentMessage{
		{Id: "t1_u", ConversationId: conv.Id, Role: RoleUser, Content: "第一轮用户", Payload: `{"role":"user","content":"第一轮用户"}`},
		{Id: "t1_a", ConversationId: conv.Id, Role: RoleAssistant, Content: "第一轮回复", Payload: `{"role":"assistant","content":"第一轮回复"}`},
		{Id: "t2_u", ConversationId: conv.Id, Role: RoleUser, Content: "第二轮用户", Payload: `{"role":"user","content":"第二轮用户"}`},
		{Id: "t2_a", ConversationId: conv.Id, Role: RoleAssistant, Content: "第二轮回复", Payload: `{"role":"assistant","content":"第二轮回复"}`},
	}
	for i := range msgs1 {
		_ = store.SaveMessage(&msgs1[i])
	}

	cfg := Effective(&models.AgentUserSettings{
		Model:           "deepseek-flash",
		MaxPromptTokens: 32000,
	})

	summary1 := "## Goal\n目标1\n## Progress / Decisions / Blocked / Next\n- Progress: 第一阶段"
	mockChat1 := func(ctx context.Context, baseURL, apiKey string, req client.CompletionsRequest) (*client.CompletionsResponse, error) {
		resp := &client.CompletionsResponse{}
		resp.Choices = []struct {
			Message client.CompletionsMessage `json:"message"`
		}{
			{Message: client.CompletionsMessage{Role: RoleAssistant, Content: summary1}},
		}
		return resp, nil
	}

	// 1st Compaction
	cmp1, err := CompactConversation(context.Background(), store, conv, cfg, mockChat1, "", nil)
	require.NoError(t, err)
	require.NotNil(t, cmp1)

	// Add more turns
	msgs2 := []models.AgentMessage{
		{Id: "t3_u", ConversationId: conv.Id, Role: RoleUser, Content: "第三轮用户", Payload: `{"role":"user","content":"第三轮用户"}`},
		{Id: "t3_a", ConversationId: conv.Id, Role: RoleAssistant, Content: "第三轮回复", Payload: `{"role":"assistant","content":"第三轮回复"}`},
		{Id: "t4_u", ConversationId: conv.Id, Role: RoleUser, Content: "第四轮用户", Payload: `{"role":"user","content":"第四轮用户"}`},
		{Id: "t4_a", ConversationId: conv.Id, Role: RoleAssistant, Content: "第四轮回复", Payload: `{"role":"assistant","content":"第四轮回复"}`},
	}
	for i := range msgs2 {
		_ = store.SaveMessage(&msgs2[i])
	}

	// 2nd Compaction: verify previous summary is passed to LLM
	summary2 := "## Goal\n目标1与2\n## Progress / Decisions / Blocked / Next\n- Progress: 第二阶段"
	var interceptedPrompt string
	mockChat2 := func(ctx context.Context, baseURL, apiKey string, req client.CompletionsRequest) (*client.CompletionsResponse, error) {
		for _, raw := range req.Messages {
			var m map[string]any
			_ = json.Unmarshal(raw, &m)
			if m["role"] == "user" {
				interceptedPrompt = m["content"].(string)
			}
		}
		resp := &client.CompletionsResponse{}
		resp.Choices = []struct {
			Message client.CompletionsMessage `json:"message"`
		}{
			{Message: client.CompletionsMessage{Role: RoleAssistant, Content: summary2}},
		}
		return resp, nil
	}

	cmp2, err := CompactConversation(context.Background(), store, conv, cfg, mockChat2, "", nil)
	require.NoError(t, err)
	require.NotNil(t, cmp2)

	// Verify the 2nd compaction prompt includes the 1st summary inside <previous-summary> tags
	require.Contains(t, interceptedPrompt, "<previous-summary>")
	require.Contains(t, interceptedPrompt, summary1)
	require.Contains(t, interceptedPrompt, "</previous-summary>")

	// Test ToInput behavior with 2nd compaction
	allMsgs, err := store.ListMessages(conv.Id)
	require.NoError(t, err)
	input := ToInput(allMsgs)
	require.NotEmpty(t, input)

	// First item in input should be the summary message
	var firstMsg map[string]any
	require.NoError(t, json.Unmarshal(input[0], &firstMsg))
	require.Equal(t, RoleUser, firstMsg["role"])
	require.Contains(t, firstMsg["content"].(string), summary2)

	// Messages before firstKeptMessageId of cmp2 must not be in input
	inputContentStr := ""
	for _, raw := range input {
		inputContentStr += string(raw) + "\n"
	}
	require.NotContains(t, inputContentStr, "第一轮用户")
	require.NotContains(t, inputContentStr, "第一轮回复")
	// Kept tail (t4) should be present
	require.Contains(t, inputContentStr, "第四轮用户")
}

func TestSummarizationPromptContainsFormatConstraints(t *testing.T) {
	store := NewMemoryStore()
	conv := &models.AgentConversation{
		Id:               "test-format-prompt",
		CreateId:         1,
		LastPromptTokens: 26000,
	}
	_ = store.SaveConversation(conv)

	msgs := []models.AgentMessage{
		{Id: "u1", ConversationId: conv.Id, Role: RoleUser, Content: "需求说明", Payload: `{"role":"user","content":"需求说明"}`},
		{Id: "a1", ConversationId: conv.Id, Role: RoleAssistant, Content: "回复", Payload: `{"role":"assistant","content":"回复"}`},
		{Id: "u2", ConversationId: conv.Id, Role: RoleUser, Content: "后续", Payload: `{"role":"user","content":"后续"}`},
	}
	for i := range msgs {
		_ = store.SaveMessage(&msgs[i])
	}

	cfg := Effective(&models.AgentUserSettings{
		Model:           "deepseek-flash",
		MaxPromptTokens: 32000,
	})

	var capturedReq client.CompletionsRequest
	mockChat := func(ctx context.Context, baseURL, apiKey string, req client.CompletionsRequest) (*client.CompletionsResponse, error) {
		capturedReq = req
		resp := &client.CompletionsResponse{}
		resp.Choices = []struct {
			Message client.CompletionsMessage `json:"message"`
		}{
			{Message: client.CompletionsMessage{Role: RoleAssistant, Content: "## Goal\nDone"}},
		}
		return resp, nil
	}

	_, err := CompactConversation(context.Background(), store, conv, cfg, mockChat, "", nil)
	require.NoError(t, err)

	require.Len(t, capturedReq.Messages, 2)

	// Check system message constraints
	var sysMsg map[string]any
	require.NoError(t, json.Unmarshal(capturedReq.Messages[0], &sysMsg))
	require.Equal(t, "system", sysMsg["role"])
	sysContent := sysMsg["content"].(string)
	require.Contains(t, sysContent, "context summarization assistant")
	require.Contains(t, sysContent, "ONLY output the structured summary")

	// Check user prompt contains required markdown headings
	var userMsg map[string]any
	require.NoError(t, json.Unmarshal(capturedReq.Messages[1], &userMsg))
	require.Equal(t, "user", userMsg["role"])
	userContent := userMsg["content"].(string)
	require.Contains(t, userContent, "## Goal")
	require.Contains(t, userContent, "## Product")
	require.Contains(t, userContent, "## Config")
	require.Contains(t, userContent, "## TSL / Script / Collector")
	require.Contains(t, userContent, "## Progress / Decisions / Blocked / Next")
}

func TestToInputWithCompactionAndToolCalls(t *testing.T) {
	// Old messages before compaction
	u1 := models.AgentMessage{Id: "u1", Role: RoleUser, Content: "创建", Payload: `{"role":"user","content":"创建"}`}
	a1 := models.AgentMessage{Id: "a1", Role: RoleAssistant, Content: "好的", Payload: `{"role":"assistant","content":"好的"}`}

	// Compaction event
	cmpPayload, _ := json.Marshal(CompactionPayload{
		Summary:            "## Goal\n之前的创建操作已完成",
		FirstKeptMessageId: "u2",
		TokensBefore:       1000,
	})
	cmp := models.AgentMessage{
		Id:        "cmp1",
		Role:      RoleEvent,
		EventType: EventCompaction,
		Payload:   string(cmpPayload),
		Content:   "## Goal\n之前的创建操作已完成",
	}

	// Kept messages from u2 onwards, with assistant tool call and tool result
	u2 := models.AgentMessage{Id: "u2", Role: RoleUser, Content: "查设备", Payload: `{"role":"user","content":"查设备"}`}
	asstPayload, _ := json.Marshal(map[string]any{
		"role":    "assistant",
		"content": "",
		"tool_calls": []any{
			map[string]any{
				"id":   "call_list",
				"type": "function",
				"function": map[string]any{
					"name":      "list_products",
					"arguments": "{}",
				},
			},
		},
	})
	a2 := models.AgentMessage{Id: "a2", Role: RoleAssistant, Content: "", Payload: string(asstPayload)}
	toolPayload, _ := json.Marshal(map[string]any{
		"role":         "tool",
		"tool_call_id": "call_list",
		"name":         "list_products",
		"content":      `{"total":0}`,
	})
	t2 := models.AgentMessage{Id: "t2", Role: RoleTool, Content: `{"total":0}`, ToolName: "list_products", ToolCallId: "call_list", Payload: string(toolPayload)}

	msgs := []models.AgentMessage{u1, a1, cmp, u2, a2, t2}

	input := ToInput(msgs)
	require.Len(t, input, 4) // [summary, u2, a2, t2]

	// 1. Summary
	var m0 map[string]any
	require.NoError(t, json.Unmarshal(input[0], &m0))
	require.Equal(t, RoleUser, m0["role"])
	require.Contains(t, m0["content"].(string), "之前的创建操作已完成")

	// 2. User 2
	var m1 map[string]any
	require.NoError(t, json.Unmarshal(input[1], &m1))
	require.Equal(t, RoleUser, m1["role"])
	require.Equal(t, "查设备", m1["content"])

	// 3. Assistant 2 with tool_calls
	var m2 map[string]any
	require.NoError(t, json.Unmarshal(input[2], &m2))
	require.Equal(t, RoleAssistant, m2["role"])
	calls, ok := m2["tool_calls"].([]any)
	require.True(t, ok)
	require.Len(t, calls, 1)

	// 4. Tool 2 result
	var m3 map[string]any
	require.NoError(t, json.Unmarshal(input[3], &m3))
	require.Equal(t, RoleTool, m3["role"])
	require.Equal(t, "call_list", m3["tool_call_id"])
	require.Equal(t, "list_products", m3["name"])
}

func TestRunTurnExecutesCompactionWhenAboveThreshold(t *testing.T) {
	store := NewMemoryStore()
	conv := &models.AgentConversation{
		Id:               "test-runner-compact",
		CreateId:         1,
		Title:            "测试会话",
		LastPromptTokens: 26000, // triggers compaction with deepseek-flash threshold of 25600
	}
	_ = store.SaveConversation(conv)

	// Older messages that will be compacted
	_ = store.SaveMessage(&models.AgentMessage{
		Id: "u1", ConversationId: conv.Id, Role: RoleUser, Content: "创建产品A", Payload: `{"role":"user","content":"创建产品A"}`,
	})
	_ = store.SaveMessage(&models.AgentMessage{
		Id: "a1", ConversationId: conv.Id, Role: RoleAssistant, Content: "产品A已创建", Payload: `{"role":"assistant","content":"产品A已创建"}`,
	})

	cfg := &models.AgentUserSettings{
		Model:           "deepseek-flash",
		MaxPromptTokens: 32000,
		MaxTokens:       16384,
	}

	callCount := 0
	origChat := DefaultChat
	origStream := DefaultChatStream
	defer func() {
		DefaultChat = origChat
		DefaultChatStream = origStream
	}()

	DefaultStreamFn := func(ctx context.Context, baseURL, apiKey string, req client.CompletionsRequest, onDelta func(client.StreamDelta)) (*client.CompletionsResponse, error) {
		callCount++
		resp := &client.CompletionsResponse{}
		if req.User == "compaction" {
			// Compaction call
			resp.Choices = []struct {
				Message client.CompletionsMessage `json:"message"`
			}{
				{Message: client.CompletionsMessage{Role: RoleAssistant, Content: "## Goal\n创建产品A已完成\n## Product\n- productId: A"}},
			}
			return resp, nil
		}

		// Turn assistant call: verify it received the compacted summary message in its messages!
		foundSummary := false
		for _, raw := range req.Messages {
			var m map[string]any
			_ = json.Unmarshal(raw, &m)
			if m["role"] == RoleUser && strings.Contains(m["content"].(string), "<summary>") {
				foundSummary = true
			}
		}
		require.True(t, foundSummary, "Completions call must receive the compacted summary")

		resp.Choices = []struct {
			Message client.CompletionsMessage `json:"message"`
		}{
			{Message: client.CompletionsMessage{Role: RoleAssistant, Content: "正在为您处理新任务"}},
		}
		resp.Usage.PromptTokens = 500
		resp.Usage.CompletionTokens = 20
		return resp, nil
	}
	DefaultChatStream = DefaultStreamFn
	DefaultChat = func(ctx context.Context, baseURL, apiKey string, req client.CompletionsRequest) (*client.CompletionsResponse, error) {
		return DefaultStreamFn(ctx, baseURL, apiKey, req, nil)
	}

	res, err := RunTurn(context.Background(), store, conv, cfg, "开始新任务B", false)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, RunIdle, res.RunStatus)
	require.GreaterOrEqual(t, callCount, 2, "must call both compaction and turn completion")

	// Verify a compaction message was stored
	msgs, err := store.ListMessages(conv.Id)
	require.NoError(t, err)
	hasCompaction := false
	for _, m := range msgs {
		if m.EventType == EventCompaction {
			hasCompaction = true
			break
		}
	}
	require.True(t, hasCompaction, "compaction event must be saved in store")
}
