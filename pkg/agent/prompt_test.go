package agent

import (
	"encoding/json"
	"fmt"
	"testing"

	"go-iot/pkg/tsl"

	"github.com/stretchr/testify/require"
)

func TestSystemPromptMessagePrependedShape(t *testing.T) {
	require.NotEmpty(t, SystemPrompt())
	raw := SystemPromptMessage()
	var m map[string]string
	require.NoError(t, json.Unmarshal(raw, &m))
	require.Equal(t, "system", m["role"])
	require.Equal(t, SystemPrompt(), m["content"])
}

func TestSystemPromptIncludesLiveTSLSchema(t *testing.T) {
	p := SystemPrompt()
	md := tsl.SchemaMarkdown()
	require.NotEmpty(t, md)
	require.Contains(t, p, md)
	require.Contains(t, p, "specs")
	require.Contains(t, p, `"unit"`)
	require.Contains(t, p, "scale")
}

func TestCreateProductToolSchemaHasNetworkEnum(t *testing.T) {
	var found json.RawMessage
	for _, t0 := range ToolCatalog() {
		if t0.Function.Name == ToolCreateProduct {
			found = t0.Function.Parameters
		}
	}
	require.NotEmpty(t, found)
	require.Contains(t, string(found), `"MQTT_BROKER"`)
	require.Contains(t, string(found), `"enum"`)
	require.NotContains(t, string(found), `"MQTT"`)
}

func TestGetTSLSchemaMatchesFromJson(t *testing.T) {
	out, err := ToolExec{}.Execute(ToolGetTSLSchema, json.RawMessage(`{}`))
	require.NoError(t, err)
	m, ok := out.(map[string]any)
	require.True(t, ok)
	require.Contains(t, fmt.Sprint(m["forbiddenKeys"]), "specs")
	require.Equal(t, tsl.Schema()["idPattern"], m["idPattern"])
}
