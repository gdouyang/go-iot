package codec

import (
	"log"

	"github.com/beego/beego/v2/core/logs"
)

// 会话信息
type Session interface {
	Send(msg interface{}) error
	Disconnect() error
}

// 设备信息
type Device interface {
	GetId() string
	// 获取会话
	GetSession() Session
	GetData() map[string]interface{}
	GetConfig() map[string]interface{}
}

// 产品信息
type Product interface {
	GetId() string
	GetConfig() map[string]interface{}
	GetTimeSeries() TimeSeries
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

// base context
type BaseContext struct {
	DeviceId  string
	ProductId string
	Session   Session
}

func (ctx *BaseContext) GetDevice() Device {
	return GetDeviceManager().Get(ctx.DeviceId)
}

func (ctx *BaseContext) GetProduct() Product {
	return GetProductManager().Get(ctx.ProductId)
}

func (ctx *BaseContext) GetSession() Session {
	return ctx.Session
}

// save time series data
func (ctx *BaseContext) Save(data map[string]interface{}) {
	p := ctx.GetProduct()
	if p == nil {
		logs.Error("product not found " + ctx.ProductId)
	} else {
		p.GetTimeSeries().Save(ctx.ProductId, data)
	}
}

// 编解码接口
type Codec interface {
	// 设备连接时
	OnConnect(ctx Context) error
	// 接收消息
	OnMessage(ctx Context) error
	// 命令调用
	OnInvoke(ctx Context) error
	// 连接关闭
	OnClose(ctx Context) error
}

// 时序保存
type TimeSeries interface {
	Save(productId string, data map[string]interface{})
}

// 网络配置
type Network struct {
	Name          string `json:"name"`
	Port          uint16 `json:"port"`
	ProductId     string `json:"productId"`
	Configuration string `json:"configuration"`
	Script        string `json:"script"`
	Type          string `json:"type"`
	CodecId       string `json:"codecId"`
}

// 功能调用
type FuncInvokeContext struct {
	message   interface{}
	session   Session
	deviceId  string
	productId string
}

func (ctx *FuncInvokeContext) GetMessage() interface{} {
	return ctx.message
}
func (ctx *FuncInvokeContext) GetSession() Session {
	return ctx.session
}

// 获取设备操作
func (ctx *FuncInvokeContext) GetDevice() Device {
	return GetDeviceManager().Get(ctx.deviceId)
}

// 获取产品操作
func (ctx *FuncInvokeContext) GetProduct() Product {
	return GetProductManager().Get(ctx.productId)
}

// productId
var codecMap = map[string]Codec{}

func GetCodec(productId string) Codec {
	codec := codecMap[productId]
	return codec
}

var codecFactory = map[string]func(network Network) Codec{}

func regCodecCreator(id string, creator func(network Network) Codec) {
	_, ok := codecFactory[id]
	if ok {
		log.Panicln(id + " is exist")
	}
	codecFactory[id] = creator
}

func NewCodec(network Network) Codec {
	return codecFactory[network.CodecId](network)
}
