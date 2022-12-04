package codec

import (
	"fmt"
	"sync"

	"github.com/beego/beego/v2/core/logs"
)

// productId
var codecMap sync.Map
var deviceLifeCycleMap sync.Map

func GetCodec(productId string) Codec {
	val, ok := codecMap.Load(productId)
	if val == nil || !ok {
		logs.Error(fmt.Sprintf("%s not found codec", productId))
	} else {
		codec := val.(Codec)
		return codec
	}
	return nil
}

func RegCodec(productId string, c Codec) {
	codecMap.Store(productId, c)
}

func NewCodec(network NetworkConf) Codec {
	c := codecFactory[network.CodecId](network)
	return c
}

var codecFactory = map[string]func(network NetworkConf) Codec{}

func regCodecCreator(id string, creator func(network NetworkConf) Codec) {
	_, ok := codecFactory[id]
	if ok {
		logs.Error("codec " + id + " is exist")
		return
	}
	codecFactory[id] = creator
}

func regDeviceLifeCycle(id string, liefcycle DeviceLifecycle) {
	val, ok := deviceLifeCycleMap.Load(id)
	if val == nil || !ok {
		deviceLifeCycleMap.Store(id, liefcycle)
	}
}

func GetDeviceLifeCycle(productId string) DeviceLifecycle {
	val, ok := deviceLifeCycleMap.Load(productId)
	if val == nil || !ok {
		logs.Error(fmt.Sprintf("%s not found DeviceLifecycle", productId))
	} else {
		v := val.(DeviceLifecycle)
		return v
	}
	return nil
}
