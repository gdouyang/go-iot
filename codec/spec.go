package codec

import (
	"go-iot/codec/tsl"
	"strings"

	"github.com/beego/beego/v2/core/logs"
)

type (
	// 编解码接口
	Codec interface {
		// 设备连接时
		OnConnect(ctx Context) error
		// 接收消息
		OnMessage(ctx Context) error
		// 命令调用
		OnInvoke(ctx Context) error
		// 连接关闭
		OnClose(ctx Context) error
	}
	// 设备信息
	Device interface {
		GetId() string
		// 获取会话
		GetSession() Session
		GetData() map[string]interface{}
		GetConfig() map[string]interface{}
	}
	// 产品信息
	Product interface {
		GetId() string
		GetConfig() map[string]interface{}
		GetTimeSeries() TimeSeriesSave
		GetTslProperty() map[string]tsl.TslProperty
	}
	// 会话信息
	Session interface {
		Send(msg interface{}) error
		Disconnect() error
		GetDeviceId() string
		SetDeviceId(deviceId string)
	}
	// 上下文
	Context interface {
		GetMessage() interface{}
		GetSession() Session
		// 获取设备操作
		GetDevice() Device
		// 获取产品操作
		GetProduct() Product
	}
)

// base context
type BaseContext struct {
	DeviceId  string
	ProductId string
	Session   Session
}

func (ctx *BaseContext) DeviceOnline(deviceId string) {
	deviceId = strings.TrimSpace(deviceId)
	if len(deviceId) > 0 {
		ctx.DeviceId = deviceId
		ctx.GetSession().SetDeviceId(deviceId)
		GetSessionManager().Put(deviceId, ctx.GetSession())
	}
}

func (ctx *BaseContext) GetDevice() Device {
	return GetDeviceManager().Get(ctx.DeviceId)
}

func (ctx *BaseContext) GetProduct() Product {
	return GetProductManager().Get(ctx.ProductId)
}

func (ctx *BaseContext) GetSession() Session {
	return ctx.Session
}

// save time series data
func (ctx *BaseContext) Save(data map[string]interface{}) {
	p := ctx.GetProduct()
	if p == nil {
		logs.Error("product not found " + ctx.ProductId)
	} else {
		if _, ok := data["deviceId"]; !ok {
			data["deviceId"] = ctx.DeviceId
		}
		p.GetTimeSeries().Save(p, data)
	}
}

// 网络配置
type Network struct {
	Name          string `json:"name"`
	Port          int32  `json:"port"`
	ProductId     string `json:"productId"`
	Configuration string `json:"configuration"`
	Script        string `json:"script"`
	Type          string `json:"type"`
	CodecId       string `json:"codecId"`
}
