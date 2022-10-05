package codec

import (
	"errors"
	"go-iot/codec/msg"
)

// 进行功能调用
func DoCmdInvoke(productId string, message msg.FuncInvoke) error {
	session := sessionManager.Get(message.DeviceId)
	if session == nil {
		return errors.New("设备不在线")
	}
	codec := GetCodec(productId)
	return codec.OnInvoke(&FuncInvokeContext{session: session,
		deviceId: message.DeviceId, productId: productId})
}

// 功能调用
type FuncInvokeContext struct {
	message   interface{}
	session   Session
	deviceId  string
	productId string
}

func (ctx *FuncInvokeContext) GetMessage() interface{} {
	return ctx.message
}
func (ctx *FuncInvokeContext) GetSession() Session {
	return ctx.session
}

// 获取设备操作
func (ctx *FuncInvokeContext) GetDevice() Device {
	return GetDeviceManager().Get(ctx.deviceId)
}

// 获取产品操作
func (ctx *FuncInvokeContext) GetProduct() Product {
	return GetProductManager().Get(ctx.productId)
}
