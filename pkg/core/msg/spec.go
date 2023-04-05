package msg

type MessageType string

const (
	PROPERTY_REPORT = "PropertyReport" // 属性上报
	PROPERTY_READ   = "PropertyRead"   // 属性读取
	PROPERTY_WRITE  = "PropertyWrite"  // 属性设置
	DEVICE_ONLINE   = "DeviceOnline"   // 设备上线
	DEVICE_OFFLINE  = "DeviceOffline"  // 设备离线
	FUNC_INVOKE     = "FuncInvoke"     // 功能调用
)

type Message interface {
	Type() MessageType
	GetData() interface{}
}

// 属性上报
type PropertyReport struct {
	DeviceId string
	Data     map[string]interface{}
}

func (p *PropertyReport) Type() MessageType {
	return PROPERTY_REPORT
}

func (p *PropertyReport) GetData() interface{} {
	return p.Data
}

// 设备上线
type DeviceOnline struct {
	DeviceId string
}

func (p *DeviceOnline) Type() MessageType {
	return DEVICE_ONLINE
}

func (p *DeviceOnline) GetData() interface{} {
	return nil
}

// 设备离线
type DeviceOffline struct {
	DeviceOnline
}

func (p *DeviceOffline) Type() MessageType {
	return DEVICE_OFFLINE
}

// 功能调用
type FuncInvoke struct {
	FunctionId string                 `json:"functionId"`
	DeviceId   string                 `json:"deviceId"`
	ClusterId  string                 `json:"clusterId"`
	Data       map[string]interface{} `json:"data"`
	Replay     chan error             `json:"-"`
}

func (p *FuncInvoke) Type() MessageType {
	return FUNC_INVOKE
}
