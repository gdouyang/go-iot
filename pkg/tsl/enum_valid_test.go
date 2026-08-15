package tsl_test

import (
	"fmt"
	"testing"

	"go-iot/pkg/tsl"

	"github.com/stretchr/testify/require"
)

func TestFromJsonRejectsEmptyEnumElements(t *testing.T) {
	text := `{
  "properties": [
    {"id": "mode", "name": "mode", "type": "enum", "elements": []}
  ]
}`
	err := tsl.NewTslData().FromJson(text)
	require.Error(t, err)
	require.Contains(t, err.Error(), "enum elements is empty")
}

func TestFromJsonRejectsSpecsWrapper(t *testing.T) {
	err := tsl.NewTslData().FromJson(`{
  "properties": [
    {"id": "temperature", "name": "温度", "type": "float", "specs": {"unit": "℃", "min": "-40"}}
  ]
}`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "specs")
}

func TestFromJsonRejectsUnknownMinStep(t *testing.T) {
	err := tsl.NewTslData().FromJson(`{
  "properties": [
    {"id": "temperature", "name": "温度", "type": "float", "unit": "℃", "min": "-40", "step": "0.1"}
  ]
}`)
	require.Error(t, err)
}

func TestFromJsonAcceptsUnitScale(t *testing.T) {
	data := tsl.NewTslData()
	require.NoError(t, data.FromJson(`{
  "properties": [
    {"id": "temperature", "name": "温度", "type": "float", "unit": "℃", "scale": 1}
  ]
}`))
	require.Equal(t, "temperature", data.Properties[0].GetId())
}

func TestFromJsonRejectsSpecsOnObject(t *testing.T) {
	err := tsl.NewTslData().FromJson(`{
  "properties": [
    {"id": "loc", "name": "位置", "type": "object", "specs": {}, "properties": [{"id": "lat", "name": "纬度", "type": "double"}]}
  ]
}`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "specs")
}

func TestSchemaForbidsSpecs(t *testing.T) {
	s := tsl.Schema()
	require.Contains(t, fmt.Sprint(s["forbiddenKeys"]), "specs")
	byType, ok := s["propertyByType"].(map[string][]string)
	require.True(t, ok)
	require.Contains(t, byType["float"], "unit")
	require.Contains(t, byType["float"], "scale")
	require.NotContains(t, byType["float"], "specs")
}

func TestFromJsonAcceptsValidEnum(t *testing.T) {
	text := `{
  "properties": [
    {"id": "mode", "name": "mode", "type": "enum", "elements": [{"text": "开", "value": "on"}]}
  ]
}`
	data := tsl.NewTslData()
	require.NoError(t, data.FromJson(text))
	require.Len(t, data.Properties, 1)
	require.Equal(t, "mode", data.Properties[0].GetId())
}
