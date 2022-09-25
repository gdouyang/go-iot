package codec

type DeviceLifecycle interface {
	// 设备新增
	OnCreate(ctx Context) error
	// 设备删除
	OnDelete(ctx Context) error
	// 设备修改
	OnUpdate(ctx Context) error
	// 设备状态检查
	OnStateChecker(ctx Context) error
}