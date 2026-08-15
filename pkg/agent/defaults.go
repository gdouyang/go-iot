package agent

import "go-iot/pkg/models"

const (
	DefaultBaseURL           = ""
	DefaultModel             = ""
	DefaultReasoningEffort   = "high"
	DefaultTemperature       = 0.2
	DefaultMaxTokens         = 16384
	DefaultMaxTurns          = 16
	DefaultTimeoutSeconds    = 90
	DefaultRunTimeoutSeconds = 600
	DefaultMaxPromptTokens   = 32000
	DefaultDraftTTLHours     = 24
	DefaultWriteMode         = "confirm"
)

// Effective fills empty user settings with in-code defaults (not yaml).
func Effective(st *models.AgentUserSettings) *models.AgentUserSettings {
	out := models.AgentUserSettings{}
	if st != nil {
		out = *st
	}
	if out.BaseURL == "" {
		out.BaseURL = DefaultBaseURL
	}
	if out.Model == "" {
		out.Model = DefaultModel
	}
	if out.ReasoningEffort == "" {
		out.ReasoningEffort = DefaultReasoningEffort
	}
	if out.Temperature == nil {
		t := DefaultTemperature
		out.Temperature = &t
	}
	if out.MaxTokens <= 0 {
		out.MaxTokens = DefaultMaxTokens
	}
	if out.MaxTurns <= 0 {
		out.MaxTurns = DefaultMaxTurns
	}
	if out.TimeoutSeconds <= 0 {
		out.TimeoutSeconds = DefaultTimeoutSeconds
	}
	if out.RunTimeoutSeconds <= 0 {
		out.RunTimeoutSeconds = DefaultRunTimeoutSeconds
	}
	if out.MaxPromptTokens <= 0 {
		out.MaxPromptTokens = DefaultMaxPromptTokens
	}
	if out.DraftTTLHours <= 0 {
		out.DraftTTLHours = DefaultDraftTTLHours
	}
	if out.WriteMode == "" {
		out.WriteMode = DefaultWriteMode
	}
	return &out
}
