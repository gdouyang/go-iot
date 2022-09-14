package tcpserver

type tcpContext struct {
	Data []byte
}

func (ctx *tcpContext) GetMessage() interface{} {
	return ctx.Data
}

// 获取设备操作
func (ctx *tcpContext) GetDevice() error {
	return nil
}

// 获取产品操作
func (ctx *tcpContext) GetProduct() error {
	return nil
}
