package agent

import (
	"fmt"
	"net"
	"net/url"
	"strings"

	"go-iot/pkg/models"
)

const (
	ReasonModelNotConfigured = "model_not_configured"
	ReasonInvalidArgs        = "invalid_args"
	ReasonNotFound           = "not_found"
	ReasonRunActive          = "run_active"
	ReasonPendingDrafts      = "pending_drafts"
	ReasonNotAwaiting        = "not_awaiting"
	ReasonDraftExpired       = "draft_expired"
	ReasonToolsNotSupported  = "tools_not_supported"
)

type SettingsView struct {
	BaseURL           string   `json:"baseUrl"`
	Model             string   `json:"model"`
	ReasoningEffort   string   `json:"reasoningEffort"`
	Temperature       *float64 `json:"temperature"`
	MaxTokens         int      `json:"maxTokens"`
	MaxTurns          int      `json:"maxTurns"`
	TimeoutSeconds    int      `json:"timeoutSeconds"`
	RunTimeoutSeconds int      `json:"runTimeoutSeconds"`
	MaxPromptTokens   int      `json:"maxPromptTokens"`
	DraftTTLHours     int      `json:"draftTtlHours"`
	AllowPrivateLLM   bool     `json:"allowPrivateLlm"`
	WriteMode         string   `json:"writeMode"`
	ApiKeySet         bool     `json:"apiKeySet"`
	ApiKeyMasked      string   `json:"apiKeyMasked"`
	ModelConfigured   bool     `json:"modelConfigured"`
	DefaultBaseURL    string   `json:"defaultBaseUrl"`
	DefaultModel      string   `json:"defaultModel"`
}

type SettingsPut struct {
	BaseURL           string   `json:"baseUrl"`
	Model             string   `json:"model"`
	ApiKey            string   `json:"apiKey"`
	ApiKeyClear       bool     `json:"apiKeyClear"`
	ReasoningEffort   string   `json:"reasoningEffort"`
	Temperature       *float64 `json:"temperature"`
	MaxTokens         *int     `json:"maxTokens"`
	MaxTurns          *int     `json:"maxTurns"`
	TimeoutSeconds    *int     `json:"timeoutSeconds"`
	RunTimeoutSeconds *int     `json:"runTimeoutSeconds"`
	MaxPromptTokens   *int     `json:"maxPromptTokens"`
	DraftTTLHours     *int     `json:"draftTtlHours"`
	AllowPrivateLLM   *bool    `json:"allowPrivateLlm"`
	WriteMode         string   `json:"writeMode"`
}

func ModelConfigured(st *models.AgentUserSettings) bool {
	if st == nil {
		return false
	}
	return strings.TrimSpace(st.ApiKey) != ""
}

func MaskAPIKey(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	if len(key) <= 4 {
		return "****"
	}
	return "****" + key[len(key)-4:]
}

func ViewSettings(st *models.AgentUserSettings) SettingsView {
	eff := Effective(st)
	v := SettingsView{
		BaseURL:           eff.BaseURL,
		Model:             eff.Model,
		ReasoningEffort:   eff.ReasoningEffort,
		Temperature:       eff.Temperature,
		MaxTokens:         eff.MaxTokens,
		MaxTurns:          eff.MaxTurns,
		TimeoutSeconds:    eff.TimeoutSeconds,
		RunTimeoutSeconds: eff.RunTimeoutSeconds,
		MaxPromptTokens:   eff.MaxPromptTokens,
		DraftTTLHours:     eff.DraftTTLHours,
		AllowPrivateLLM:   false,
		WriteMode:         eff.WriteMode,
		DefaultBaseURL:    DefaultBaseURL,
		DefaultModel:      DefaultModel,
	}
	if st != nil {
		v.AllowPrivateLLM = st.AllowPrivateLLM
		if strings.TrimSpace(st.BaseURL) != "" {
			v.BaseURL = st.BaseURL
		}
		if strings.TrimSpace(st.Model) != "" {
			v.Model = st.Model
		}
		v.ApiKeySet = strings.TrimSpace(st.ApiKey) != ""
		if v.ApiKeySet {
			v.ApiKeyMasked = MaskAPIKey(st.ApiKey)
		}
		v.ModelConfigured = ModelConfigured(st)
	}
	return v
}

func GetOrEmptySettings(store Store, userId int64) (*models.AgentUserSettings, error) {
	return store.GetSettings(userId)
}

func PutSettings(store Store, userId int64, put SettingsPut) (*models.AgentUserSettings, error) {
	if err := validateReasoningEffort(put.ReasoningEffort); err != nil {
		return nil, err
	}
	if put.WriteMode != "" && put.WriteMode != "confirm" && put.WriteMode != "auto" {
		return nil, fmt.Errorf("%s: invalid writeMode", ReasonInvalidArgs)
	}
	cur, err := store.GetSettings(userId)
	if err != nil {
		return nil, err
	}
	if cur == nil {
		cur = &models.AgentUserSettings{
			Id:     settingsID(userId),
			UserId: userId,
		}
	}
	allowPrivate := cur.AllowPrivateLLM
	if put.AllowPrivateLLM != nil {
		allowPrivate = *put.AllowPrivateLLM
	}
	if err := ValidateBaseURL(put.BaseURL, allowPrivate); err != nil {
		return nil, err
	}
	if put.BaseURL != "" {
		cur.BaseURL = strings.TrimRight(put.BaseURL, "/")
	}
	if put.Model != "" {
		cur.Model = put.Model
	}
	if strings.TrimSpace(put.ReasoningEffort) != "" {
		cur.ReasoningEffort = strings.TrimSpace(put.ReasoningEffort)
	}
	if put.Temperature != nil {
		cur.Temperature = put.Temperature
	}
	if put.MaxTokens != nil {
		cur.MaxTokens = *put.MaxTokens
	}
	if put.MaxTurns != nil {
		cur.MaxTurns = *put.MaxTurns
	}
	if put.TimeoutSeconds != nil {
		cur.TimeoutSeconds = *put.TimeoutSeconds
	}
	if put.RunTimeoutSeconds != nil {
		cur.RunTimeoutSeconds = *put.RunTimeoutSeconds
	}
	if put.MaxPromptTokens != nil {
		cur.MaxPromptTokens = *put.MaxPromptTokens
	}
	if put.DraftTTLHours != nil {
		cur.DraftTTLHours = *put.DraftTTLHours
	}
	if put.AllowPrivateLLM != nil {
		cur.AllowPrivateLLM = *put.AllowPrivateLLM
	}
	if put.WriteMode != "" {
		cur.WriteMode = put.WriteMode
	}
	if strings.TrimSpace(cur.BaseURL) == "" {
		return nil, fmt.Errorf("%s: baseUrl is required", ReasonInvalidArgs)
	}
	if strings.TrimSpace(cur.Model) == "" {
		return nil, fmt.Errorf("%s: model is required", ReasonInvalidArgs)
	}
	if put.ApiKeyClear {
		cur.ApiKey = ""
	} else if strings.TrimSpace(put.ApiKey) != "" {
		cur.ApiKey = strings.TrimSpace(put.ApiKey)
	}
	cur.Id = settingsID(userId)
	cur.UserId = userId
	cur.UpdateTime = models.NewDateTime()
	if err := store.SaveSettings(cur); err != nil {
		return nil, err
	}
	return cur, nil
}

func validateReasoningEffort(raw string) error {
	s := strings.TrimSpace(raw)
	if s == "" {
		return nil
	}
	if len(s) > 32 {
		return fmt.Errorf("%s: reasoningEffort too long", ReasonInvalidArgs)
	}
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.' {
			continue
		}
		return fmt.Errorf("%s: invalid reasoningEffort", ReasonInvalidArgs)
	}
	return nil
}

func ValidateBaseURL(raw string, allowPrivate bool) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("%s: invalid baseUrl", ReasonInvalidArgs)
	}
	if u.User != nil {
		return fmt.Errorf("%s: baseUrl must not contain userinfo", ReasonInvalidArgs)
	}
	path := strings.TrimRight(u.Path, "/")
	if strings.HasSuffix(path, "/chat/completions") {
		return fmt.Errorf("%s: baseUrl must not end with /chat/completions", ReasonInvalidArgs)
	}
	if u.Scheme != "https" {
		if !(allowPrivate && u.Scheme == "http") {
			return fmt.Errorf("%s: baseUrl must be https", ReasonInvalidArgs)
		}
	}
	host := u.Hostname()
	if strings.EqualFold(host, "localhost") {
		if !allowPrivate {
			return fmt.Errorf("%s: private llm host not allowed", ReasonInvalidArgs)
		}
		return nil
	}
	ip := net.ParseIP(host)
	if ip != nil {
		if isPrivateIP(ip) && !allowPrivate {
			return fmt.Errorf("%s: private llm host not allowed", ReasonInvalidArgs)
		}
		return nil
	}
	addrs, err := lookupIP(host)
	if err != nil {
		return fmt.Errorf("%s: resolve baseUrl host: %v", ReasonInvalidArgs, err)
	}
	for _, a := range addrs {
		if isPrivateIP(a) && !allowPrivate {
			return fmt.Errorf("%s: private llm host not allowed", ReasonInvalidArgs)
		}
	}
	return nil
}

var lookupIP = net.LookupIP

func isPrivateIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast()
}
