package agent

import (
	"encoding/json"
	"testing"

	"go-iot/pkg/agent/client"

	"github.com/stretchr/testify/require"
)

func TestBeforeToolCallInterceptsWrites(t *testing.T) {
	args := json.RawMessage(`{"id":"A","name":"n","networkType":"MQTT_BROKER"}`)
	g := BeforeToolCall("confirm", ToolCreateProduct, args)
	require.Equal(t, ToolConfirm, g.Action)
	require.True(t, g.Terminate)

	g = BeforeToolCall("auto", ToolCreateProduct, args)
	require.Equal(t, ToolAllow, g.Action)

	g = BeforeToolCall("confirm", ToolListProducts, json.RawMessage(`{}`))
	require.Equal(t, ToolAllow, g.Action)

	g = BeforeToolCall("confirm", ToolSaveTSL, json.RawMessage(`not-json`))
	require.Equal(t, ToolBlock, g.Action)
	require.True(t, g.Terminate)
}

func TestWillPauseForConfirm(t *testing.T) {
	var tc client.ToolCall
	tc.Function.Name = ToolCreateProduct
	tc.Function.Arguments = `{"id":"A","name":"n","networkType":"MQTT_BROKER"}`
	require.True(t, willPauseForConfirm("confirm", []client.ToolCall{tc}))
	require.False(t, willPauseForConfirm("auto", []client.ToolCall{tc}))

	var read client.ToolCall
	read.Function.Name = ToolListProducts
	read.Function.Arguments = `{}`
	require.False(t, willPauseForConfirm("confirm", []client.ToolCall{read}))
}
