package agent

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"go-iot/pkg/eventbus"
	"go-iot/pkg/models"

	"github.com/stretchr/testify/require"
)

func TestGetDebugLogsListensOnEventBus(t *testing.T) {
	orig := lookupProduct
	t.Cleanup(func() { lookupProduct = orig })
	lookupProduct = func(id string) (*models.ProductModel, error) {
		return &models.ProductModel{Product: models.Product{Id: id, CreateId: 1}}, nil
	}
	go func() {
		time.Sleep(40 * time.Millisecond)
		eventbus.PublishDebug(eventbus.NewDebugMessage("info", "d1", "P-DBG", "from-bus"))
	}()
	out, err := ToolExec{Ctx: writeCtx(1)}.Execute(ToolGetDebugLogs, json.RawMessage(`{"productId":"P-DBG","waitMs":400,"limit":10}`))
	require.NoError(t, err)
	m := out.(map[string]any)
	require.Equal(t, true, m["ok"])
	logs, _ := m["logs"].([]map[string]any)
	require.NotEmpty(t, logs)
	require.Equal(t, "from-bus", logs[0]["data"])
}

func TestRunScriptUsesSavedScript(t *testing.T) {
	orig := lookupProduct
	t.Cleanup(func() { lookupProduct = orig })
	lookupProduct = func(id string) (*models.ProductModel, error) {
		return &models.ProductModel{Product: models.Product{
			Id: id, CreateId: 1,
			Script: `function OnMessage(c){ console.log("trial "+c.MsgToString()); c.SaveProperties({n:1}) }`,
		}}, nil
	}
	out, err := ToolExec{Ctx: writeCtx(1)}.Execute(ToolRunScript, json.RawMessage(`{"productId":"P-RUN","payload":"hi"}`))
	require.NoError(t, err)
	m := out.(map[string]any)
	require.Equal(t, true, m["ok"], m["message"])
	logs, _ := m["logs"].([]map[string]any)
	require.NotEmpty(t, logs)
	require.Contains(t, logs[0]["data"], "trial")
	props, _ := m["savedProperties"].([]map[string]any)
	require.Len(t, props, 1)
}

func TestRunScriptRejectsUnknownHook(t *testing.T) {
	orig := lookupProduct
	t.Cleanup(func() { lookupProduct = orig })
	lookupProduct = func(id string) (*models.ProductModel, error) {
		return &models.ProductModel{Product: models.Product{Id: id, CreateId: 1, Script: "function OnMessage(c){}"}}, nil
	}
	g := BeforeToolCall("confirm", writeCtx(1), ToolRunScript, json.RawMessage(`{"productId":"P1","hook":"OnFoo"}`))
	require.Equal(t, ToolBlock, g.Action)
}

func TestGetDebugLogsRequiresOwner(t *testing.T) {
	orig := lookupProduct
	t.Cleanup(func() { lookupProduct = orig })
	lookupProduct = func(id string) (*models.ProductModel, error) {
		return nil, errors.New("product not exist")
	}
	g := BeforeToolCall("confirm", writeCtx(1), ToolGetDebugLogs, json.RawMessage(`{"productId":"NOPE"}`))
	require.Equal(t, ToolBlock, g.Action)
}
