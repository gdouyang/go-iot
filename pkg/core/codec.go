package core

import (
	"sync"

	logs "go-iot/pkg/logger"
)

// 脚本编解码
const Script_Codec = "script_codec"

// codec reg
var codecMap sync.Map
var deviceLifeCycleMap sync.Map

func GetCodec(productId string) Codec {
	val, ok := codecMap.Load(productId)
	if val == nil || !ok {
		logs.Errorf("%s not found core", productId)
	} else {
		core := val.(Codec)
		return core
	}
	return nil
}

// 注册编解码器
func RegCodec(productId string, c Codec) {
	codecMap.Store(productId, c)
}

func NewCodec(codecId, productId, script string) (Codec, error) {
	c, err := codecFactory[codecId](productId, script)
	return c, err
}

var codecFactory = map[string]func(productId, script string) (Codec, error){}

func RegCodecCreator(codecId string, creator func(productId, script string) (Codec, error)) {
	_, ok := codecFactory[codecId]
	if ok {
		logs.Errorf("core %s is exist", codecId)
		return
	}
	codecFactory[codecId] = creator
}

// device lifecycle
func RegDeviceLifeCycle(productId string, liefcycle DeviceLifecycle) {
	// val, ok := deviceLifeCycleMap.Load(productId)
	// if val == nil || !ok {
	deviceLifeCycleMap.Store(productId, liefcycle)
	// }
}

func GetDeviceLifeCycle(productId string) DeviceLifecycle {
	val, ok := deviceLifeCycleMap.Load(productId)
	if val != nil && ok {
		v := val.(DeviceLifecycle)
		return v
	}
	return nil
}

// 获取设备状态，当设备有session时返回online, 否则调用OnStateChecker方法来查询
func GetDeviceState(deviceId, productId string) string {
	sess := GetSession(deviceId)
	if sess != nil {
		return ONLINE
	} else {
		liefcycle := GetDeviceLifeCycle(productId)
		if liefcycle != nil {
			state, err := liefcycle.OnStateChecker(&BaseContext{
				ProductId: productId,
				DeviceId:  deviceId,
			})
			if err == nil && state == ONLINE {
				return ONLINE
			}
		}
	}
	return OFFLINE
}

// 设备发布
func OnDeviceDeploy(device *Device) error {
	liefcycle := GetDeviceLifeCycle(device.ProductId)
	if liefcycle != nil {
		var deviceId = device.Id
		err := liefcycle.OnDeviceDeploy(&BaseContext{
			ProductId: device.ProductId,
			DeviceId:  deviceId,
			device:    device,
		})
		if err != ErrFunctionNotImpl {
			return err
		}
	}
	return nil
}

// 设备取消发布
func OnDeviceUnDeploy(device *Device) error {
	liefcycle := GetDeviceLifeCycle(device.ProductId)
	if liefcycle != nil {
		var deviceId = device.Id
		err := liefcycle.OnDeviceUnDeploy(&BaseContext{
			ProductId: device.ProductId,
			DeviceId:  deviceId,
			device:    device,
		})
		if err != ErrFunctionNotImpl {
			return err
		}
	}
	return nil
}
