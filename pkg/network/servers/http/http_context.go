package httpserver

import (
	"fmt"
	"go-iot/pkg/core"
	"net/http"
	"strings"
)

type httpContext struct {
	core.BaseContext
	Data []byte
	r    *http.Request
}

func (ctx *httpContext) DeviceOnline(deviceId string) {
	deviceId = strings.TrimSpace(deviceId)
	if len(deviceId) > 0 {
		device := ctx.GetDeviceById(deviceId)
		if device == nil {
			panic(fmt.Errorf("device [%s] is null", deviceId))
		}
		if device.GetProductId() != ctx.ProductId {
			panic(fmt.Errorf("device [%s] product error: %s != %s", deviceId, ctx.ProductId, device.GetProductId()))
		}
		core.DeviceOnlineEvent(deviceId, ctx.ProductId)
	}
}

func (ctx *httpContext) DeviceOffline(deviceId string) {
	deviceId = strings.TrimSpace(deviceId)
	if len(deviceId) > 0 {
		device := ctx.GetDeviceById(deviceId)
		if device == nil {
			panic(fmt.Errorf("device [%s] is null", deviceId))
		}
		if device.GetProductId() != ctx.ProductId {
			panic(fmt.Errorf("device [%s] product error: %s != %s", deviceId, ctx.ProductId, device.GetProductId()))
		}
		core.DeviceOfflineEvent(deviceId, ctx.ProductId)
	}
}

func (ctx *httpContext) GetMessage() interface{} {
	return ctx.Data
}

func (ctx *httpContext) MsgToString() string {
	return string(ctx.Data)
}

func (ctx *httpContext) GetHeader(key string) string {
	if ctx.r == nil {
		return ""
	}
	return ctx.r.Header.Get(key)
}

func (ctx *httpContext) GetUrl() string {
	return ctx.r.RequestURI
}

func (ctx *httpContext) GetQuery(key string) string {
	return ctx.r.Form.Get(key)
}

func (ctx *httpContext) GetForm(key string) string {
	return ctx.r.Form.Get(key)
}
