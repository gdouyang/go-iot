package codec

import (
	"fmt"

	"github.com/beego/beego/v2/core/logs"
)

// productId
var codecMap = map[string]Codec{}
var deviceLifeCycleMap = map[string]DeviceLifecycle{}

func GetCodec(productId string) Codec {
	codec := codecMap[productId]
	if codec == nil {
		logs.Error(fmt.Sprintf("%s not found codec", productId))
	}
	return codec
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
