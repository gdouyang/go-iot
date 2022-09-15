package tcpserver

import "go-iot/provider/codec"

type tcpContext struct {
	deviceId  string
	productId string
	Data      []byte
}

func (ctx *tcpContext) GetMessage() interface{} {
	return ctx.Data
}

// 获取设备操作
func (ctx *tcpContext) GetDevice() codec.Device {
	return codec.GetDeviceManager().GetDevice(ctx.deviceId)
}

// 获取产品操作
func (ctx *tcpContext) GetProduct() codec.Product {
	return codec.GetProductManager().GetProduct(ctx.productId)
}

func (ctx *tcpContext) GetSession() codec.Session {
	return codec.GetSessionManager().GetSession(ctx.deviceId)
}