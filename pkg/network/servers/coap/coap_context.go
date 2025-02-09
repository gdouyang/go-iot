package coapserver

import (
	"fmt"
	"go-iot/pkg/core"
	"go-iot/pkg/logger"
	"net/url"
	"strings"

	"github.com/plgd-dev/go-coap/v3/mux"
)

type coapContext struct {
	core.BaseContext
	Data      []byte
	r         *mux.Message
	url       string
	parsedURL *url.URL
}

func (ctx *coapContext) DeviceOnline(deviceId string) {
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

func (ctx *coapContext) DeviceOffline(deviceId string) {
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

func (ctx *coapContext) GetMessage() interface{} {
	return ctx.Data
}

func (ctx *coapContext) MsgToString() string {
	return string(ctx.Data)
}

func (ctx *coapContext) GetUrl() string {
	return ctx.url
}

func (ctx *coapContext) GetQuery(key string) string {
	var v = ""
	if ctx.parsedURL == nil {
		parsedURL, err := url.Parse(ctx.url)
		ctx.parsedURL = parsedURL
		if err != nil {
			logger.Errorf("%v", err)
		}
	}
	if ctx.parsedURL != nil {
		// 提取查询参数
		queryParams := ctx.parsedURL.Query()
		v = queryParams.Get(key)
	}

	return v
}
