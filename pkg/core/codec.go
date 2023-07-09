package core

import (
	"sync"

	logs "go-iot/pkg/logger"
)

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

func RegCodec(productId string, c Codec) {
	codecMap.Store(productId, c)
}

func NewCodec(network NetworkConf) (Codec, error) {
	c, err := codecFactory[network.CodecId](network)
	return c, err
}

var codecFactory = map[string]func(network NetworkConf) (Codec, error){}

func RegCodecCreator(codecId string, creator func(network NetworkConf) (Codec, error)) {
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

// get the device state,
// if device have session return online
// else invoke OnStateChecker method
func GetDeviceState(deviceId, productId string) string {
	sess := GetSession(deviceId)
	if sess != nil {
		return ONLINE
	} else {
		liefcycle := GetDeviceLifeCycle(productId)
		if liefcycle != nil {
			deviceOper := GetDevice(deviceId)
			if deviceOper.DeviceType == GATEWAY {
				deviceId = deviceOper.ParentId
			}
			state, err := liefcycle.OnStateChecker(&BaseContext{
				ProductId: productId,
				DeviceId:  deviceId,
			})
			if err == nil {
				if state == ONLINE {
					return ONLINE
				}
			}
		}
	}
	return OFFLINE
}
