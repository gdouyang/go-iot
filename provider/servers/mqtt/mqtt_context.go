package mqttserver

import "go-iot/provider/codec"

type mqttContext struct {
	deviceId  string
	productId string
	Data      []byte
	session   codec.Session
}

func (ctx *mqttContext) GetMessage() interface{} {
	return ctx.Data
}

// 获取设备操作
func (ctx *mqttContext) GetDevice() codec.Device {
	return codec.GetDeviceManager().GetDevice(ctx.deviceId)
}

// 获取产品操作
func (ctx *mqttContext) GetProduct() codec.Product {
	return codec.GetProductManager().GetProduct(ctx.productId)
}

func (ctx *mqttContext) GetSession() codec.Session {
	return ctx.session
}
