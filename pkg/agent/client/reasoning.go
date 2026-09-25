package client

import (
	"encoding/json"
	"strings"
)

type StreamDelta struct {
	Content   string
	Reasoning string
}

func reasoningDelta(reasoningContent string, reasoning json.RawMessage, thinking string) string {
	if reasoningContent != "" {
		return reasoningContent
	}
	if len(reasoning) > 0 && string(reasoning) != "null" {
		var s string
		if json.Unmarshal(reasoning, &s) == nil && s != "" {
			return s
		}
		if t := ReasoningText("", reasoning); t != "" {
			return t
		}
	}
	return thinking
}

func ReasoningText(reasoningContent string, reasoning json.RawMessage) string {
	if s := strings.TrimSpace(reasoningContent); s != "" {
		return s
	}
	if len(reasoning) == 0 || string(reasoning) == "null" {
		return ""
	}
	var s string
	if json.Unmarshal(reasoning, &s) == nil {
		return s
	}
	var obj map[string]any
	if json.Unmarshal(reasoning, &obj) != nil {
		return ""
	}
	for _, k := range []string{"content", "text", "summary"} {
		if v, ok := obj[k].(string); ok && strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
