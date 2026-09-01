package modbus

import "go-iot/pkg/core"

// collectContext 采集结果交给 OnMessage。GetMessage() 为 {source, groupId, properties}。
type collectContext struct {
	core.BaseContext
	msg map[string]any
}

func (c *collectContext) GetMessage() interface{} {
	return c.msg
}

func newCollectMessage(groupId string, properties map[string]any) map[string]any {
	props := make(map[string]any, len(properties))
	for k, v := range properties {
		props[k] = v
	}
	return map[string]any{
		"source":     "collect",
		"groupId":    groupId,
		"properties": props,
	}
}
