package models

// AgentConversation 会话。索引 goiot-agentconversation。
type AgentConversation struct {
	Id                   string   `json:"id" orm:"pk;column(id_);size(32)"`
	Title                string   `json:"title" orm:"column(title_);size(128)"`
	ProductId            string   `json:"productId" orm:"column(product_id_);size(32);null"`
	Status               string   `json:"status" orm:"column(status_);size(16)"`
	RunStatus            string   `json:"runStatus" orm:"column(run_status_);size(24)"`
	NeedsResume          bool     `json:"needsResume" orm:"column(needs_resume_)"`
	ResumeExpireTime     DateTime `json:"resumeExpireTime" orm:"column(resume_expire_time_);null"`
	LastPromptTokens     int      `json:"lastPromptTokens" orm:"column(last_prompt_tokens_);type(integer);null"`
	LastCompletionTokens int      `json:"lastCompletionTokens" orm:"column(last_completion_tokens_);type(integer);null"`
	LastReasoningTokens  int      `json:"lastReasoningTokens" orm:"column(last_reasoning_tokens_);type(integer);null"`
	CreateId             int64    `json:"createId" orm:"column(create_id_)"`
	CreateTime           DateTime `json:"createTime" orm:"column(create_time_)"`
	UpdateTime           DateTime `json:"updateTime" orm:"column(update_time_)"`
}

// AgentMessage Completions 线格式或 FE 事件。
type AgentMessage struct {
	Id             string   `json:"id" orm:"pk;column(id_);size(32)"`
	ConversationId string   `json:"conversationId" orm:"column(conversation_id_);size(32)"`
	Role           string   `json:"role" orm:"column(role_);size(16)"`
	EventType      string   `json:"eventType" orm:"column(event_type_);size(32);null"`
	Payload        string   `json:"payload" orm:"column(payload_);type(text)"`
	Content        string   `json:"content" orm:"column(content_);type(text)"`
	ToolName       string   `json:"toolName" orm:"column(tool_name_);size(64);null"`
	ToolCallId     string   `json:"toolCallId" orm:"column(tool_call_id_);size(64);null"`
	DraftId        string   `json:"draftId" orm:"column(draft_id_);size(32);null"`
	ProductId      string   `json:"productId" orm:"column(product_id_);size(32);null"`
	CreateId       int64    `json:"createId" orm:"column(create_id_)"`
	CreateTime     DateTime `json:"createTime" orm:"column(create_time_)"`
	CreateTimeMs   int64    `json:"createTimeMs" orm:"column(create_time_ms_)"`
}

// AgentDraft 确认草稿。apply 失败保持 pending。
type AgentDraft struct {
	Id             string   `json:"id" orm:"pk;column(id_);size(32)"`
	ConversationId string   `json:"conversationId" orm:"column(conversation_id_);size(32)"`
	ToolCallId     string   `json:"toolCallId" orm:"column(tool_call_id_);size(64)"`
	ToolName       string   `json:"toolName" orm:"column(tool_name_);size(64)"`
	ProductId      string   `json:"productId" orm:"column(product_id_);size(32);null"`
	Payload        string   `json:"payload" orm:"column(payload_);type(text)"`
	Preview        string   `json:"preview" orm:"column(preview_);type(text)"`
	Status         string   `json:"status" orm:"column(status_);size(16)"`
	ExpireTime     DateTime `json:"expireTime" orm:"column(expire_time_)"`
	CreateId       int64    `json:"createId" orm:"column(create_id_)"`
	CreateTime     DateTime `json:"createTime" orm:"column(create_time_)"`
}

// AgentAudit 审计。
type AgentAudit struct {
	Id             string   `json:"id" orm:"pk;column(id_);size(32)"`
	ConversationId string   `json:"conversationId" orm:"column(conversation_id_);size(32)"`
	DraftId        string   `json:"draftId" orm:"column(draft_id_);size(32);null"`
	ToolName       string   `json:"toolName" orm:"column(tool_name_);size(64)"`
	ProductId      string   `json:"productId" orm:"column(product_id_);size(32);null"`
	Action         string   `json:"action" orm:"column(action_);size(16)"`
	BeforeSnapshot string   `json:"beforeSnapshot" orm:"column(before_snapshot_);type(text);null"`
	AfterSummary   string   `json:"afterSummary" orm:"column(after_summary_);type(text);null"`
	Success        bool     `json:"success" orm:"column(success_)"`
	Undone         bool     `json:"undone" orm:"column(undone_)"`
	Error          string   `json:"error" orm:"column(error_);type(text);null"`
	CreateId       int64    `json:"createId" orm:"column(create_id_)"`
	CreateTime     DateTime `json:"createTime" orm:"column(create_time_)"`
}

// AgentUserSettings 每用户模型设置。Id=agentset_{userId}。ApiKey 明文存 ES，JSON 不输出。
type AgentUserSettings struct {
	Id              string   `json:"id" orm:"pk;column(id_);size(32)"`
	UserId          int64    `json:"userId" orm:"column(user_id_)"`
	BaseURL         string   `json:"baseUrl" orm:"column(base_url_);size(512);null"`
	Model           string   `json:"model" orm:"column(model_);size(128);null"`
	ApiKey            string   `json:"apiKey" orm:"column(api_key_);type(text);null"`
	ReasoningEffort   string   `json:"reasoningEffort" orm:"column(reasoning_effort_);size(32);null"`
	Temperature       *float64 `json:"temperature" orm:"column(temperature_);type(double);null"`
	MaxTokens         int      `json:"maxTokens" orm:"column(max_tokens_);type(integer);null"`
	MaxTurns          int      `json:"maxTurns" orm:"column(max_turns_);type(integer);null"`
	TimeoutSeconds    int      `json:"timeoutSeconds" orm:"column(timeout_seconds_);type(integer);null"`
	RunTimeoutSeconds int      `json:"runTimeoutSeconds" orm:"column(run_timeout_seconds_);type(integer);null"`
	MaxPromptTokens   int      `json:"maxPromptTokens" orm:"column(max_prompt_tokens_);type(integer);null"`
	DraftTTLHours     int      `json:"draftTtlHours" orm:"column(draft_ttl_hours_);type(integer);null"`
	AllowPrivateLLM   bool     `json:"allowPrivateLlm" orm:"column(allow_private_llm_)"`
	WriteMode         string   `json:"writeMode" orm:"column(write_mode_);size(16);null"`
	UpdateTime        DateTime `json:"updateTime" orm:"column(update_time_)"`
}
