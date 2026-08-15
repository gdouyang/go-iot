package tsl

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// Schema is the contract returned by get_tsl_schema. Fields match FromJson / the product editor.
func Schema() map[string]any {
	return map[string]any{
		"types":                      []string{TypeEnum, TypeInt, TypeLong, TypeString, TypeBool, TypeFloat, TypeDouble, TypeDate, TypePassword, TypeFile, TypeObject},
		"idPattern":                  `^[0-9a-zA-Z_\-]+$`,
		"arraySupported":             false,
		"enumValidEnforced":          true,
		"deviceIdReservedInFromJson": false,
		"notes": []string{
			"属性字段与 type 平级，禁止套 specs。",
			"禁止 min、step、maxLength；string/password 最大长度字段是 max。",
			"float/double 用 unit、scale；int/long 只用 unit。",
			"enum.elements 每项只有 text、value，且不能为空。",
			"deviceId 不是解析保留字，但 SaveProperties 会注入，物模型不要再定义同名属性。",
		},
		"root": map[string]any{
			"properties": "属性数组",
			"events":     "事件数组（元素同属性，常用 object）",
			"functions":  "功能数组",
		},
		"propertyCommon": []string{"id", "name", "type", "description", "expands"},
		"propertyByType": map[string][]string{
			TypeInt:      {"unit"},
			TypeLong:     {"unit"},
			TypeFloat:    {"unit", "scale"},
			TypeDouble:   {"unit", "scale"},
			TypeString:   {"max"},
			TypePassword: {"max"},
			TypeBool:     {"trueText", "trueValue", "falseText", "falseValue"},
			TypeDate:     {"format"},
			TypeEnum:     {"elements"},
			TypeFile:     {"bodyType"},
			TypeObject:   {"properties"},
		},
		"forbiddenKeys": []string{"specs", "min", "step", "maxLength"},
		"enumElement":   []string{"text", "value"},
		"function":      []string{"id", "name", "async", "inputs", "output", "expands", "description"},
		"exampleProperty": map[string]any{
			"id": "temperature", "name": "温度", "type": TypeFloat, "unit": "℃", "scale": 1,
		},
	}
}

// SchemaMarkdown is a compact prompt section built from Schema(), so it stays in sync with FromJson.
func SchemaMarkdown() string {
	s := Schema()
	var b strings.Builder
	b.WriteString("类型：")
	b.WriteString(joinAny(s["types"]))
	b.WriteString("\nid：")
	b.WriteString(fmt.Sprint(s["idPattern"]))
	b.WriteString("\n公共字段：")
	b.WriteString(joinAny(s["propertyCommon"]))
	b.WriteString("\n按类型附加字段：\n")
	if byType, ok := s["propertyByType"].(map[string][]string); ok {
		keys := make([]string, 0, len(byType))
		for k := range byType {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			b.WriteString("- ")
			b.WriteString(k)
			b.WriteString("：")
			if len(byType[k]) == 0 {
				b.WriteString("无")
			} else {
				b.WriteString(strings.Join(byType[k], "、"))
			}
			b.WriteString("\n")
		}
	}
	b.WriteString("禁止字段：")
	b.WriteString(joinAny(s["forbiddenKeys"]))
	b.WriteString("\nenum.elements：")
	b.WriteString(joinAny(s["enumElement"]))
	b.WriteString("\nfunction：")
	b.WriteString(joinAny(s["function"]))
	if raw, err := json.Marshal(s["exampleProperty"]); err == nil {
		b.WriteString("\n示例：")
		b.WriteString(string(raw))
	}
	if notes, ok := s["notes"].([]string); ok && len(notes) > 0 {
		b.WriteString("\n")
		for _, n := range notes {
			b.WriteString("- ")
			b.WriteString(n)
			b.WriteString("\n")
		}
	}
	return strings.TrimSpace(b.String())
}

func joinAny(v any) string {
	switch t := v.(type) {
	case []string:
		return strings.Join(t, "、")
	default:
		return fmt.Sprint(v)
	}
}
