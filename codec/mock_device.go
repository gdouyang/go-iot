package codec

type MockContext struct {
	DeviceId  string `json:"deviceId"`
	ProductId string `json:"productId"`
	Data      []byte
	session   Session
}

func (ctx *MockContext) GetMessage() interface{} {
	return ctx.Data
}

// 获取设备操作
func (ctx *MockContext) GetDevice() *Device {
	return GetDeviceManager().Get(ctx.DeviceId)
}

// 获取产品操作
func (ctx *MockContext) GetProduct() *Product {
	return GetProductManager().Get(ctx.ProductId)
}

func (ctx *MockContext) GetSession() Session {
	return ctx.session
}
