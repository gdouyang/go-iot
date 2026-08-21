package agent

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"go-iot/pkg/agent/skills"
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

func TestSystemPromptKeepsToolsAndSkillsSeparate(t *testing.T) {
	p := SystemPrompt()
	tools := ToolsCatalogMarkdown()
	skill := skills.CatalogMarkdown()
	require.Contains(t, p, tools)
	require.Contains(t, p, skill)
	require.Less(t, strings.Index(p, tools), strings.Index(p, skill))
	require.NotContains(t, tools, "<available_skills>")
	require.NotContains(t, skill, "Available tools:")
	for _, t0 := range ToolCatalog() {
		require.Contains(t, tools, "- "+t0.Function.Name+": "+t0.Function.Description)
	}
}

func TestSystemPromptDoesNotDuplicateSkillNetworkTypes(t *testing.T) {
	p := SystemPrompt()
	require.Contains(t, p, "# 网络类型")
	require.NotContains(t, p, "每个产品单独起一套 Broker")
	require.Contains(t, skills.CatalogMarkdown(), "独占端口")
	require.Contains(t, skills.CatalogMarkdown(), "1883")
}

func TestSystemPromptIncludesSkillCatalog(t *testing.T) {
	p := SystemPrompt()
	require.Contains(t, p, "load_skill")
	require.NotContains(t, p, "get_codec_doc")
	require.Contains(t, p, "<available_skills>")
	require.Contains(t, p, "</available_skills>")
	require.Contains(t, p, skills.CatalogMarkdown())
	var found bool
	for _, t0 := range ToolCatalog() {
		if t0.Function.Name == ToolLoadSkill {
			found = true
			require.NotContains(t, string(t0.Function.Parameters), "MQTT_BROKER")
		}
		require.NotEqual(t, "get_codec_doc", t0.Function.Name)
	}
	require.True(t, found)
}

func TestLoadSkillReturnsCodecMarkdown(t *testing.T) {
	listed, err := ToolExec{}.Execute(ToolLoadSkill, json.RawMessage(`{}`))
	require.NoError(t, err)
	m, ok := listed.(map[string]any)
	require.True(t, ok)
	require.Equal(t, true, m["ok"])
	require.NotEmpty(t, m["skills"])

	out, err := ToolExec{}.Execute(ToolLoadSkill, json.RawMessage(`{"name":"MQTT_BROKER"}`))
	require.NoError(t, err)
	m, ok = out.(map[string]any)
	require.True(t, ok)
	require.Equal(t, true, m["ok"])
	require.Equal(t, "codec-mqtt-broker", m["name"])
	require.Contains(t, fmt.Sprint(m["markdown"]), "GetPassword")

	miss, err := ToolExec{}.Execute(ToolLoadSkill, json.RawMessage(`{"name":"MQTT"}`))
	require.NoError(t, err)
	m, ok = miss.(map[string]any)
	require.True(t, ok)
	require.Equal(t, false, m["ok"])
	require.NotEmpty(t, m["skills"])
}

func TestGetTSLSchemaMatchesFromJson(t *testing.T) {
	out, err := ToolExec{}.Execute(ToolGetTSLSchema, json.RawMessage(`{}`))
	require.NoError(t, err)
	m, ok := out.(map[string]any)
	require.True(t, ok)
	require.Contains(t, fmt.Sprint(m["forbiddenKeys"]), "specs")
	require.Equal(t, tsl.Schema()["idPattern"], m["idPattern"])
}
