package websocketsocker

import "go-iot/provider/codec"

type websocketContext struct {
	deviceId  string
	productId string
	Data      []byte
	session   codec.Session
}

func (ctx *websocketContext) GetMessage() interface{} {
	return ctx.Data
}

// 获取设备操作
func (ctx *websocketContext) GetDevice() codec.Device {
	return codec.GetDeviceManager().GetDevice(ctx.deviceId)
}

// 获取产品操作
func (ctx *websocketContext) GetProduct() codec.Product {
	return codec.GetProductManager().GetProduct(ctx.productId)
}

func (ctx *websocketContext) GetSession() codec.Session {
	return ctx.session
}

func (ctx *websocketContext) MsgToString() string {
	return string(ctx.Data)
}
