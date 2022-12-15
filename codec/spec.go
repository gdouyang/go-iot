package codec

import (
	"errors"
	"go-iot/codec/tsl"
	"strings"

	"github.com/beego/beego/v2/core/logs"
)

type (
	// 编解码接口
	Codec interface {
		// 设备连接时
		OnConnect(ctx MessageContext) error
		// 接收消息
		OnMessage(ctx MessageContext) error
		// 命令调用
		OnInvoke(ctx MessageContext) error
		// 连接关闭
		OnClose(ctx MessageContext) error
	}
	DeviceLifecycle interface {
		// 设备新增
		OnCreate(ctx DeviceLifecycleContext) error
		// 设备删除
		OnDelete(ctx DeviceLifecycleContext) error
		// 设备修改
		OnUpdate(ctx DeviceLifecycleContext) error
		// 设备状态检查
		OnStateChecker(ctx DeviceLifecycleContext) (string, error)
	}
	// 产品信息
	Product interface {
		GetId() string
		GetConfig(key string) string
		GetTimeSeries() TimeSeriesSave
		GetTsl() *tsl.TslData
	}
	// 设备信息
	Device interface {
		GetId() string
		GetProductId() string
		// 获取会话
		GetSession() Session
		GetData() map[string]string
		GetConfig(key string) string
	}
	// 会话信息
	Session interface {
		Disconnect() error
		GetDeviceId() string
		SetDeviceId(deviceId string)
	}
	// 消息上下文
	MessageContext interface {
		DeviceLifecycleContext
		GetMessage() interface{}
		GetSession() Session
	}
	//  设备生命周期上下文
	DeviceLifecycleContext interface {
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
		device := GetDeviceManager().Get(deviceId)
		if device == nil {
			logs.Warn("device [%s] not exist, skip online", deviceId)
			return
		}
		ctx.DeviceId = deviceId
		ctx.GetSession().SetDeviceId(deviceId)
		GetSessionManager().Put(deviceId, ctx.GetSession())
	}
}

func (ctx *BaseContext) GetDevice() Device {
	return ctx.GetDeviceById(ctx.DeviceId)
}

func (ctx *BaseContext) GetDeviceById(deviceId string) Device {
	if len(ctx.DeviceId) == 0 {
		return nil
	}
	return GetDeviceManager().Get(deviceId)
}

func (ctx *BaseContext) GetProduct() Product {
	if len(ctx.ProductId) == 0 {
		return nil
	}
	return GetProductManager().Get(ctx.ProductId)
}

func (ctx *BaseContext) GetSession() Session {
	return ctx.Session
}

func (ctx *BaseContext) GetConfig(key string) string {
	device := ctx.GetDevice()
	if device == nil {
		return ""
	}
	return device.GetConfig(key)
}

// save time series data
func (ctx *BaseContext) SaveProperties(data map[string]interface{}) {
	ctx._saveProperties("", data)
}
func (ctx *BaseContext) SaveEvents(eventId string, data map[string]interface{}) {
	ctx._saveProperties(eventId, data)
}
func (ctx *BaseContext) _saveProperties(eventId string, data map[string]interface{}) {
	p := ctx.GetProduct()
	if p == nil {
		logs.Error("product not found " + ctx.ProductId)
		return
	}
	if ctx.GetDevice() == nil {
		logs.Warn("device not offline")
		return
	}
	data["deviceId"] = ctx.DeviceId
	if len(eventId) == 0 {
		p.GetTimeSeries().SaveProperties(p, data)
	} else {
		p.GetTimeSeries().SaveEvents(p, eventId, data)
	}
}

func (ctx *BaseContext) ReplyOk() {
	replyMap.reply(ctx.DeviceId, nil)
}

func (ctx *BaseContext) ReplyFail(resp string) {
	replyMap.reply(ctx.DeviceId, errors.New(resp))
}

func (ctx *BaseContext) HttpRequest(config map[string]interface{}) map[string]interface{} {
	return HttpRequest(config)
}
