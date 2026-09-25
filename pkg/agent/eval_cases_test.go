package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"go-iot/pkg/agent/client"
	"go-iot/pkg/models"

	"github.com/stretchr/testify/require"
)

func TestEvalCasesAgainstHarness(t *testing.T) {
	dir := filepath.Join("testdata", "eval", "cases")
	ents, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.NotEmpty(t, ents)

	var ran int
	for _, ent := range ents {
		if ent.IsDir() || filepath.Ext(ent.Name()) != ".json" {
			continue
		}
		ran++
		name := ent.Name()
		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join(dir, name))
			require.NoError(t, err)
			var cas EvalCase
			require.NoError(t, json.Unmarshal(raw, &cas))
			require.NotEmpty(t, cas.ID)
			require.NotEmpty(t, cas.Expect.RunStatus)

			tr := runEvalScripted(t, cas)
			score := ScoreTrace(tr, cas.Expect)
			score.ID = cas.ID
			if !score.Pass {
				b, _ := json.MarshalIndent(score.Failures(), "", "  ")
				t.Fatalf("%s failed baseline checks:\n%s", cas.ID, b)
			}
		})
	}
	require.GreaterOrEqual(t, ran, 6)
}

func runEvalScripted(t *testing.T, c EvalCase) EvalTrace {
	t.Helper()
	origChat := DefaultChat
	origStream := DefaultChatStream
	origLookup := lookupProduct
	t.Cleanup(func() {
		DefaultChat = origChat
		DefaultChatStream = origStream
		lookupProduct = origLookup
	})
	DefaultChatStream = nil
	chat := &evalScript{turns: scriptToMessages(c.Script)}
	DefaultChat = chat.Chat

	var product *models.ProductModel
	if c.Product != nil {
		nt := c.Product.NetworkType
		if nt == "" {
			nt = "MQTT_BROKER"
		}
		product = &models.ProductModel{Product: models.Product{
			Id: c.Product.ID, CreateId: 1, NetworkType: nt, Script: c.Product.Script, State: c.Product.State,
		}}
	}
	lookupProduct = func(id string) (*models.ProductModel, error) {
		if product == nil || product.Id != id {
			return nil, errors.New("product not exist")
		}
		cp := *product
		return &cp, nil
	}
	origDev := lookupDevice
	origAdd := addDevice
	t.Cleanup(func() {
		lookupDevice = origDev
		addDevice = origAdd
	})
	lookupDevice = func(id string) (*models.DeviceModel, error) {
		return nil, errors.New("missing")
	}
	addDevice = func(ob *models.DeviceModel) error { return nil }

	store := NewMemoryStore()
	st := &models.AgentUserSettings{UserId: 1, BaseURL: "https://example.com/v1", Model: "m", ApiKey: "k"}
	conv := &models.AgentConversation{
		Id: "eval-" + c.ID, Title: "eval", Status: "active",
		RunStatus: RunIdle, CreateId: 1, CreateTime: models.NewDateTime(),
	}
	require.NoError(t, store.SaveConversation(conv))
	perms := writeCtx(1).Perms
	if c.NoPerms {
		perms = map[string]bool{}
	}
	_, err := RunTurnOpts(context.Background(), RunOpts{
		Store: store, Conv: conv, Settings: st, UserText: c.User, Perms: perms,
	})
	require.NoError(t, err)
	return ExtractTrace(store, conv)
}

type evalScript struct {
	turns []client.CompletionsMessage
	i     int
}

func (s *evalScript) Chat(ctx context.Context, baseURL, apiKey string, req client.CompletionsRequest) (*client.CompletionsResponse, error) {
	if s.i >= len(s.turns) {
		return &client.CompletionsResponse{}, nil
	}
	m := s.turns[s.i]
	s.i++
	return &client.CompletionsResponse{Choices: []struct {
		Message client.CompletionsMessage `json:"message"`
	}{{Message: m}}}, nil
}

func scriptToMessages(turns []evalTurn) []client.CompletionsMessage {
	var out []client.CompletionsMessage
	for i, tr := range turns {
		if tr.Tool == "" {
			out = append(out, client.CompletionsMessage{Role: "assistant", Content: tr.Content})
			continue
		}
		var tc client.ToolCall
		tc.Id = fmt.Sprintf("call_%d", i+1)
		tc.Type = "function"
		tc.Function.Name = tr.Tool
		tc.Function.Arguments = string(tr.Args)
		if tc.Function.Arguments == "" {
			tc.Function.Arguments = "{}"
		}
		out = append(out, client.CompletionsMessage{Role: "assistant", ToolCalls: []client.ToolCall{tc}})
	}
	return out
}
