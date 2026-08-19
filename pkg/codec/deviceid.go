package codec

import (
	"strings"

	"go-iot/pkg/core"
)

func extractDeviceId(param interface{}) string {
	if param == nil {
		return ""
	}
	if ctx, ok := param.(core.DeviceLifecycleContext); ok && ctx != nil {
		if d := ctx.GetDevice(); d != nil && d.Id != "" {
			return d.Id
		}
	}
	if ctx, ok := param.(core.MessageContext); ok && ctx != nil {
		if s := ctx.GetSession(); s != nil {
			if id := strings.TrimSpace(s.GetDeviceId()); id != "" {
				return id
			}
		}
	}
	switch c := param.(type) {
	case *core.BaseContext:
		return strings.TrimSpace(c.DeviceId)
	case core.BaseContext:
		return strings.TrimSpace(c.DeviceId)
	case *core.FuncInvokeContext:
		return strings.TrimSpace(c.DeviceId)
	case core.FuncInvokeContext:
		return strings.TrimSpace(c.DeviceId)
	}
	return ""
}
