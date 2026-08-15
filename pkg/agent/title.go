package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"go-iot/pkg/agent/client"
	logs "go-iot/pkg/logger"
	"go-iot/pkg/models"
)

const (
	DefaultConvTitle    = "新会话"
	maxTitleRunes       = 16
	maxTitleSourceRunes = 500
	defaultTitleTimeout = 8 * time.Second
)

var titleTimeout = defaultTitleTimeout

var untitledSeq = regexp.MustCompile(`^新会话(\d+)$`)

func NormalizeUserTitle(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", fmt.Errorf("%s: title is required", ReasonInvalidArgs)
	}
	if utf8.RuneCountInString(s) > 128 {
		s = string([]rune(s)[:128])
	}
	return s, nil
}

func IsDefaultTitle(title string) bool {
	return strings.TrimSpace(title) == "" || strings.TrimSpace(title) == DefaultConvTitle
}

func NextUntitledTitle(titles []string) string {
	max := 0
	for _, t := range titles {
		m := untitledSeq.FindStringSubmatch(strings.TrimSpace(t))
		if len(m) != 2 {
			continue
		}
		n, err := strconv.Atoi(m[1])
		if err != nil || n <= max {
			continue
		}
		max = n
	}
	return fmt.Sprintf("%s%d", DefaultConvTitle, max+1)
}

func SanitizeTitle(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	s = strings.Trim(s, " \t\"'`“”‘’「」『』。.!！?？")
	s = strings.TrimPrefix(s, "**")
	s = strings.TrimSuffix(s, "**")
	s = strings.TrimSpace(s)
	if s == "" || IsDefaultTitle(s) || untitledSeq.MatchString(s) {
		return ""
	}
	if utf8.RuneCountInString(s) > maxTitleRunes {
		r := []rune(s)
		s = string(r[:maxTitleRunes])
	}
	if len(s) > 128 {
		s = s[:128]
	}
	return s
}

func ResolveTitle(ctx context.Context, store Store, conv *models.AgentConversation, settings *models.AgentUserSettings, userText string) string {
	if t := SanitizeTitle(summarizeTitle(ctx, settings, userText)); t != "" {
		return t
	}
	list, err := store.ListConversations(conv.CreateId)
	if err != nil {
		logs.Warnf("agent title list conversations: %v", err)
		return DefaultConvTitle + "1"
	}
	titles := make([]string, 0, len(list))
	for _, c := range list {
		if c.Id == conv.Id {
			continue
		}
		titles = append(titles, c.Title)
	}
	fb := NextUntitledTitle(titles)
	logs.Warnf("agent title fallback conv=%s title=%s", conv.Id, fb)
	return fb
}

func summarizeTitle(ctx context.Context, settings *models.AgentUserSettings, userText string) string {
	chat := DefaultChat
	if chat == nil || settings == nil {
		return ""
	}
	src := strings.TrimSpace(userText)
	if src == "" {
		return ""
	}
	if utf8.RuneCountInString(src) > maxTitleSourceRunes {
		src = string([]rune(src)[:maxTitleSourceRunes])
	}
	cfg := Effective(settings)
	sys, _ := json.Marshal(map[string]string{
		"role":    "system",
		"content": "根据用户这句话写一个不超过16个字的会话标题。只输出标题，不要引号、标点收尾或解释。",
	})
	usr, _ := json.Marshal(map[string]string{"role": "user", "content": src})
	temp := 0.2
	ctx, cancel := context.WithTimeout(ctx, titleTimeout)
	defer cancel()
	resp, err := chat(ctx, cfg.BaseURL, settings.ApiKey, client.CompletionsRequest{
		Model:           cfg.Model,
		Messages:        []json.RawMessage{sys, usr},
		Temperature:     &temp,
		MaxTokens:       256,
		ReasoningEffort: "none",
		User:            strconv.FormatInt(settings.UserId, 10),
	})
	if err != nil {
		logs.Warnf("agent title summarize llm error: %v", err)
		return ""
	}
	if resp == nil || len(resp.Choices) == 0 {
		logs.Warnf("agent title summarize empty choices")
		return ""
	}
	return resp.Choices[0].Message.Content
}
