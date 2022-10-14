package codec

import (
	"fmt"
	"sync"

	"github.com/beego/beego/v2/core/logs"
)

// productId
var codecMap sync.Map
var deviceLifeCycleMap = map[string]DeviceLifecycle{}

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

var codecFactory = map[string]func(network Network) Codec{}

func regCodecCreator(id string, creator func(network Network) Codec) {
	_, ok := codecFactory[id]
	if ok {
		logs.Error("codec " + id + " is exist")
		return
	}
	codecFactory[id] = creator
}

func NewCodec(network Network) Codec {
	c := codecFactory[network.CodecId](network)
	switch t := c.(type) {
	case DeviceLifecycle:
		deviceLifeCycleMap[network.ProductId] = t
	default:
	}
	return c
}

func GetDeviceLifeCycle(productId string) DeviceLifecycle {
	d := deviceLifeCycleMap[productId]
	return d
}
