package agent

import (
	"encoding/json"
	"testing"

	"go-iot/pkg/core"
	"go-iot/pkg/models"
	_ "go-iot/pkg/network/servers/mqtt5"

	"github.com/stretchr/testify/require"
)

func TestGetProductConfigListsMQTTBrokerKeys(t *testing.T) {
	orig := lookupProduct
	t.Cleanup(func() { lookupProduct = orig })
	lookupProduct = func(id string) (*models.ProductModel, error) {
		return &models.ProductModel{Product: models.Product{Id: id, CreateId: 1, NetworkType: "MQTT_BROKER"}}, nil
	}
	out, err := ToolExec{Ctx: writeCtx(1)}.Execute(ToolGetProductConfig, json.RawMessage(`{"productId":"TEST-MQTT"}`))
	require.NoError(t, err)
	m := out.(map[string]any)
	require.Equal(t, true, m["ok"])
	items, ok := m["items"].([]core.MetaConfig)
	require.True(t, ok)
	var keys []string
	for _, it := range items {
		keys = append(keys, it.Property)
	}
	require.Contains(t, keys, "username")
	require.Contains(t, keys, "password")
}

func TestSaveProductConfigRequiresSet(t *testing.T) {
	orig := lookupProduct
	t.Cleanup(func() { lookupProduct = orig })
	lookupProduct = func(id string) (*models.ProductModel, error) {
		return &models.ProductModel{Product: models.Product{Id: id, CreateId: 1, NetworkType: "MQTT_BROKER"}}, nil
	}
	g := BeforeToolCall("confirm", writeCtx(1), ToolSaveProductConfig, json.RawMessage(`{"productId":"P1"}`))
	require.Equal(t, ToolBlock, g.Action)
	g = BeforeToolCall("confirm", writeCtx(1), ToolSaveProductConfig, json.RawMessage(`{"productId":"P1","set":{"username":"u"}}`))
	require.Equal(t, ToolConfirm, g.Action)
	require.True(t, g.Terminate)
}
