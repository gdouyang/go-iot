package core

import (
	"encoding/json"
	"errors"
	"go-iot/pkg/core/tsl"
	"strings"

	"github.com/beego/beego/v2/core/logs"
)

const (
	ONLINE   = "online"   // 在线
	OFFLINE  = "offline"  // 离线
	NoActive = "noActive" // 未启用

	DEVICE    = "device"    // 直连设备
	GATEWAY   = "gateway"   // 网关
	SUBDEVICE = "subdevice" // 子设备
)

var ErrNotImpl = errors.New("function not impl")

type (
	// 编解码接口
	Codec interface {
		// 设备连接时
		OnConnect(ctx MessageContext) error
		// 接收消息
		OnMessage(ctx MessageContext) error
		// 命令调用
		OnInvoke(ctx FuncInvokeContext) error
		// 连接关闭
		OnClose(ctx MessageContext) error
	}
	DeviceLifecycle interface {
		// 设备新增
		OnCreate(ctx DeviceLifecycleContext) error
		// 设备删除
		OnDelete(ctx DeviceLifecycleContext) error
		// 设备修改
		OnUpdate(ctx DeviceLifecycleContext) error
		// 设备状态检查
		OnStateChecker(ctx DeviceLifecycleContext) (string, error)
	}
	// 会话信息
	Session interface {
		Disconnect() error
		GetDeviceId() string
		SetDeviceId(deviceId string)
	}
	// 消息上下文
	MessageContext interface {
		DeviceLifecycleContext
		GetMessage() interface{}
		GetSession() Session
	}
	//  设备生命周期上下文
	DeviceLifecycleContext interface {
		// 获取设备操作
		GetDevice() *Device
		// 获取产品操作
		GetProduct() *Product
	}
)

type DefaultMetaConfig struct {
	MetaConfigs []ProductMetaConfig
	CodecId     string
}

func (p DefaultMetaConfig) ToJson() string {
	b, _ := json.Marshal(p.MetaConfigs)
	return string(b)
}

// the meta config of product
type ProductMetaConfig struct {
	Property string `json:"property,omitempty"`
	Type     string `json:"type,omitempty"`
	Value    string `json:"value,omitempty"`
	Buildin  bool   `json:"buildin,omitempty"`
	Desc     string `json:"desc,omitempty"`
}

// default product impl
type Product struct {
	Id          string
	Config      map[string]string
	StorePolicy string
	TslData     *tsl.TslData
}

func NewProduct(id string, config map[string]string, storePolicy string, tsltext string) (*Product, error) {
	tslData := tsl.NewTslData()
	if len(tsltext) > 0 {
		err := tslData.FromJson(tsltext)
		if err != nil {
			return nil, err
		}
	}
	return &Product{
		Id:          id,
		Config:      config,
		StorePolicy: storePolicy,
		TslData:     tslData,
	}, nil
}

func (p *Product) GetId() string {
	return p.Id
}
func (p *Product) GetConfig(key string) string {
	if v, ok := p.Config[key]; ok {
		return v
	}
	return ""
}

func (p *Product) GetTimeSeries() TimeSeriesSave {
	return GetTimeSeries(p.StorePolicy)
}

func (p *Product) GetTsl() *tsl.TslData {
	return p.TslData
}

// default device impl
func NewDevice(devieId string, productId string, createId int64) *Device {
	return &Device{
		Id:        devieId,
		ProductId: productId,
		CreateId:  createId,
		Data:      make(map[string]string),
		Config:    make(map[string]string),
	}
}

type Device struct {
	Id         string            `json:"id"`
	ProductId  string            `json:"productId"`
	ParentId   string            `json:"parentId"`
	DeviceType string            `json:"deviceType"`
	ClusterId  string            `json:"clusterId"` // 所在集群id
	CreateId   int64             `json:"createId"`
	Data       map[string]string `json:"data"`
	Config     map[string]string `json:"config"`
}

func (d *Device) GetId() string {
	return d.Id
}
func (d *Device) GetProductId() string {
	return d.ProductId
}
func (d *Device) GetCreateId() int64 {
	return d.CreateId
}
func (d *Device) GetSession() Session {
	s := GetSession(d.Id)
	return s
}
func (d *Device) GetData() map[string]string {
	return d.Data
}
func (d *Device) GetConfig(key string) string {
	if v, ok := d.Config[key]; ok {
		return v
	}
	p := GetProduct(d.ProductId)
	if p != nil {
		v := p.GetConfig(key)
		return v
	}
	return ""
}

// base context
type BaseContext struct {
	DeviceId  string
	ProductId string
	Session   Session
}

func (ctx *BaseContext) DeviceOnline(deviceId string) {
	deviceId = strings.TrimSpace(deviceId)
	if len(deviceId) > 0 {
		sess := GetSession(deviceId)
		if sess == nil {
			device := GetDevice(deviceId)
			if device == nil {
				logs.Warn("device [%s] not exist, skip online", deviceId)
				return
			}
			ctx.DeviceId = deviceId
			ctx.GetSession().SetDeviceId(deviceId)
			PutSession(deviceId, ctx.GetSession())
		}
	}
}

func (ctx *BaseContext) GetDevice() *Device {
	return ctx.GetDeviceById(ctx.DeviceId)
}

func (ctx *BaseContext) GetDeviceById(deviceId string) *Device {
	if len(ctx.DeviceId) == 0 {
		return nil
	}
	return GetDevice(deviceId)
}

func (ctx *BaseContext) GetProduct() *Product {
	if len(ctx.ProductId) == 0 {
		return nil
	}
	return GetProduct(ctx.ProductId)
}

func (ctx *BaseContext) GetSession() Session {
	return ctx.Session
}

func (ctx *BaseContext) GetMessage() interface{} {
	return nil
}

func (ctx *BaseContext) GetConfig(key string) string {
	device := ctx.GetDevice()
	if device == nil {
		return ""
	}
	return device.GetConfig(key)
}

// save time series data
func (ctx *BaseContext) SaveProperties(data map[string]interface{}) {
	p := ctx.GetProduct()
	if p == nil {
		logs.Warn("product [%s] not exist or noActive", ctx.ProductId)
		return
	}
	if ctx.GetDevice() == nil {
		logs.Warn("device [%s] is offline", ctx.DeviceId)
		return
	}
	data["deviceId"] = ctx.DeviceId
	p.GetTimeSeries().SaveProperties(p, data)
}

func (ctx *BaseContext) SaveEvents(eventId string, data any) {
	p := ctx.GetProduct()
	if p == nil {
		logs.Warn("product [%s] not exist or noActive", ctx.ProductId)
		return
	}
	if ctx.GetDevice() == nil {
		logs.Warn("device [%s] is offline", ctx.DeviceId)
		return
	}
	saveData := map[string]any{}
	switch d := data.(type) {
	case map[string]any:
		saveData = d
	default:
		saveData[eventId] = data
	}
	saveData["deviceId"] = ctx.DeviceId
	p.GetTimeSeries().SaveProperties(p, saveData)
}

func (ctx *BaseContext) ReplyOk() {
	replyMap.reply(ctx.DeviceId, nil)
}

func (ctx *BaseContext) ReplyFail(resp string) {
	replyMap.reply(ctx.DeviceId, errors.New(resp))
}

func (ctx *BaseContext) HttpRequest(config map[string]interface{}) map[string]interface{} {
	return HttpRequest(config)
}
