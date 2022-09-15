package codec

type MockContext struct {
	deviceId  string
	productId string
	Data      []byte
	session   Session
}

func (ctx *MockContext) GetMessage() interface{} {
	return ctx.Data
}

// 获取设备操作
func (ctx *MockContext) GetDevice() Device {
	return GetDeviceManager().GetDevice(ctx.deviceId)
}

// 获取产品操作
func (ctx *MockContext) GetProduct() Product {
	return GetProductManager().GetProduct(ctx.productId)
}

func (ctx *MockContext) GetSession() Session {
	return ctx.session
}
