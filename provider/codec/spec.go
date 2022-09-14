package codec

type DeviceManager struct {
	m map[string]*Device
}

func (dm *DeviceManager) GetDevice(deviceId string) *Device {
	device := dm.m[deviceId]
	return device
}

func (dm *DeviceManager) PutDevice(deviceId string, device *Device) {
	dm.m[deviceId] = device
}

// 会话信息
type Session interface {
	Send(msg interface{}) error
	DisConnect() error
}

// 设备信息
type Device interface {
	// 获取会话
	GetSession() (*Session, error)
	GetData() map[string]interface{}
	GetConfig() map[string]interface{}
}

// 产品信息
type Product interface {
	GetConfig() map[string]interface{}
}

// 上下文
type Context interface {
	GetMessage() interface{}
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

// productId
var codecMap = map[string]Codec{}

func GetCodec(productId string) Codec {
	codec := codecMap[productId]
	return codec
}
