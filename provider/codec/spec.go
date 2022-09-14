package codec

// 会话信息
type Session interface {
	Send()
}

// 设备信息
type Device interface {
	// 获取会话
	GetSession() (Session, error)
	GetData() map[string]interface{}
	GetConfig() map[string]interface{}
}

// 产品信息
type Product interface {
	GetConfig() map[string]interface{}
}

// 上下文
type Context interface {
	// 获取设备操作
	GetDevice() error
	// 获取产品操作
	GetProduct() error
}

// 编解码接口
type Codec interface {
	// 设备连接时
	OnConnect(ctx Context) error
	// 设备解码
	Decode(ctx Context) error
	// 编码
	Encode(ctx Context) error
	// 设备新增
	OnDeviceCreate(ctx Context) error
	// 设备删除
	OnDeviceDelete(ctx Context) error
	// 设备修改
	OnDeviceUpdate(ctx Context) error
	//
	OnStateChecker(ctx Context) error
}
