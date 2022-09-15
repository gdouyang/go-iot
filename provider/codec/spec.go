package codec

import (
	"go-iot/models"
	"log"
)

// 会话信息
type Session interface {
	Send(msg interface{}) error
	DisConnect() error
}

// 设备信息
type Device interface {
	// 获取会话
	GetSession() Session
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
	GetSession() Session
	// 获取设备操作
	GetDevice() Device
	// 获取产品操作
	GetProduct() Product
}

// 编解码接口
type Codec interface {
	// 设备连接时
	OnConnect(ctx Context) error
	// 设备解码
	Decode(ctx Context) error
	// 编码
	Encode(ctx Context) error
}

// productId
var codecMap = map[string]Codec{}

func GetCodec(productId string) Codec {
	codec := codecMap[productId]
	return codec
}

var codecFactory = map[string]func(network models.Network) Codec{}

func regCodecCreator(id string, creator func(network models.Network) Codec) {
	_, ok := codecFactory[id]
	if ok {
		log.Panicln(id + " is exist")
	}
	codecFactory[id] = creator
}

func NewCodec(network models.Network) Codec {
	return codecFactory[network.CodecId](network)
}
