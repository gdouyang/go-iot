package agent

import (
	"encoding/json"
	"errors"
	"testing"

	"go-iot/pkg/agent/client"
	"go-iot/pkg/models"

	"github.com/stretchr/testify/require"
)

func stubNoProduct(t *testing.T) {
	t.Helper()
	orig := lookupProduct
	t.Cleanup(func() { lookupProduct = orig })
	lookupProduct = func(id string) (*models.ProductModel, error) {
		return nil, errors.New("product not exist")
	}
}

func TestBeforeToolCallInterceptsWrites(t *testing.T) {
	stubNoProduct(t)
	args := json.RawMessage(`{"id":"A","name":"n","networkType":"MQTT_BROKER"}`)
	g := BeforeToolCall("confirm", 1, ToolCreateProduct, args)
	require.Equal(t, ToolConfirm, g.Action)
	require.True(t, g.Terminate)

	g = BeforeToolCall("auto", 1, ToolCreateProduct, args)
	require.Equal(t, ToolAllow, g.Action)

	g = BeforeToolCall("confirm", 1, ToolListProducts, json.RawMessage(`{}`))
	require.Equal(t, ToolAllow, g.Action)

	g = BeforeToolCall("confirm", 1, ToolSaveTSL, json.RawMessage(`not-json`))
	require.Equal(t, ToolBlock, g.Action)
	require.False(t, g.Terminate)
}

func TestBeforeToolCallRejectsDirtyTSLBeforeConfirm(t *testing.T) {
	orig := lookupProduct
	t.Cleanup(func() { lookupProduct = orig })
	lookupProduct = func(id string) (*models.ProductModel, error) {
		return &models.ProductModel{Product: models.Product{Id: id, CreateId: 7}}, nil
	}

	dirty := json.RawMessage(`{"productId":"P1","tsl":{"properties":[{"id":"temperature","name":"温度","type":"float","specs":{"unit":"℃"}}]}}`)
	g := BeforeToolCall("confirm", 7, ToolSaveTSL, dirty)
	require.Equal(t, ToolBlock, g.Action)
	require.False(t, g.Terminate)
	require.Contains(t, g.Reason, "specs")

	clean := json.RawMessage(`{"productId":"P1","tsl":{"properties":[{"id":"temperature","name":"温度","type":"float","unit":"℃","scale":1}]}}`)
	g = BeforeToolCall("confirm", 7, ToolSaveTSL, clean)
	require.Equal(t, ToolConfirm, g.Action)
}

func TestBeforeToolCallRequiresProductBeforeTSL(t *testing.T) {
	orig := lookupProduct
	t.Cleanup(func() { lookupProduct = orig })
	lookupProduct = func(id string) (*models.ProductModel, error) {
		return nil, errors.New("product [P1] not exist")
	}
	args := json.RawMessage(`{"productId":"P1","tsl":{"properties":[{"id":"t","name":"t","type":"int"}]}}`)
	g := BeforeToolCall("confirm", 7, ToolSaveTSL, args)
	require.Equal(t, ToolBlock, g.Action)
	require.Contains(t, g.Reason, "create_product")
}

func TestBeforeToolCallRejectsBrokenScriptBeforeConfirm(t *testing.T) {
	orig := lookupProduct
	t.Cleanup(func() { lookupProduct = orig })
	lookupProduct = func(id string) (*models.ProductModel, error) {
		return &models.ProductModel{Product: models.Product{Id: id, CreateId: 7}}, nil
	}
	g := BeforeToolCall("confirm", 7, ToolSaveScript, json.RawMessage(`{"productId":"P1","script":"function OnMessage("}`))
	require.Equal(t, ToolBlock, g.Action)
	require.False(t, g.Terminate)

	g = BeforeToolCall("confirm", 7, ToolSaveScript, json.RawMessage(`{"productId":"P1","script":"function OnMessage(c){}"}`))
	require.Equal(t, ToolConfirm, g.Action)
}

func TestBeforeToolCallRejectsBadNetworkType(t *testing.T) {
	g := BeforeToolCall("confirm", 1, ToolCreateProduct, json.RawMessage(`{"id":"A","name":"n","networkType":"MQTT"}`))
	require.Equal(t, ToolBlock, g.Action)
	require.Contains(t, g.Reason, "网络类型")
}

func TestEachToolHasValidateAndExecute(t *testing.T) {
	require.NotEmpty(t, ToolCatalog())
	for _, spec := range ToolCatalog() {
		t0, ok := LookupTool(spec.Function.Name)
		require.True(t, ok, spec.Function.Name)
		require.Equal(t, spec.Function.Name, t0.Name())
		require.NotEmpty(t, t0.Spec().Function.Name)
	}
	_, ok := LookupTool("deploy_product")
	require.False(t, ok)
}

func TestWillPauseForConfirm(t *testing.T) {
	stubNoProduct(t)
	var tc client.ToolCall
	tc.Function.Name = ToolCreateProduct
	tc.Function.Arguments = `{"id":"A","name":"n","networkType":"MQTT_BROKER"}`
	require.True(t, willPauseForConfirm("confirm", 1, []client.ToolCall{tc}))
	require.False(t, willPauseForConfirm("auto", 1, []client.ToolCall{tc}))

	var read client.ToolCall
	read.Function.Name = ToolListProducts
	read.Function.Arguments = `{}`
	require.False(t, willPauseForConfirm("confirm", 1, []client.ToolCall{read}))

	var dirty client.ToolCall
	dirty.Function.Name = ToolSaveTSL
	dirty.Function.Arguments = `{"productId":"P1","tsl":{"properties":[{"id":"t","name":"t","type":"float","specs":{}}]}}`
	require.False(t, willPauseForConfirm("confirm", 1, []client.ToolCall{dirty}))
}
