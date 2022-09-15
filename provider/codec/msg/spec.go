package msg

// 属性上报
type PropertyReport struct {
	DeviceId string
	Data     map[string]interface{}
}

// 设备上线
type DeviceOnline struct {
	DeviceId string
}

// 设备离线
type DeviceOffline struct {
	DeviceOnline
}

// 功能调用
type FuncInvoke struct {
	PropertyReport
}
