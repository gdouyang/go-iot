package core

import (
	"fmt"
	"sync"

	"github.com/beego/beego/v2/core/logs"
)

// codec reg
var codecMap sync.Map
var deviceLifeCycleMap sync.Map

func GetCodec(productId string) Codec {
	val, ok := codecMap.Load(productId)
	if val == nil || !ok {
		logs.Error(fmt.Sprintf("%s not found core", productId))
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
		logs.Error("core " + codecId + " is exist")
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
