package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"go-iot/pkg/eventbus"
	"go-iot/pkg/tsl"
	"strconv"
	"strings"

	logs "go-iot/pkg/logger"
)

const (
	ONLINE   = "online"   // 在线
	OFFLINE  = "offline"  // 离线
	NoActive = "noActive" // 未启用

	DEVICE    = "device"    // 设备
	GATEWAY   = "gateway"   // 网关
	SUBDEVICE = "subdevice" // 子设备
)

// 函数没有实现
var ErrFunctionNotImpl = errors.New("function not impl")

type (
	// 编解码接口
	Codec interface {
		// 设备连接
		OnConnect(ctx MessageContext) error
		// 接收消息
		OnMessage(ctx MessageContext) error
		// 命令调用
		OnInvoke(ctx FuncInvokeContext) error
		// 连接关闭
		OnClose(ctx MessageContext) error
	}
	DeviceLifecycle interface {
		// 设备发布
		OnDeviceDeploy(ctx DeviceLifecycleContext) error
		// 设备取消发布
		OnDeviceUnDeploy(ctx DeviceLifecycleContext) error
		// 设备状态检查
		OnStateChecker(ctx DeviceLifecycleContext) (string, error)
	}
	// 会话信息
	Session interface {
		// 断开连接并使离线
		Disconnect() error
		// 获取设备id
		GetDeviceId() string
		// 设置设备id，对于无法重连接中得到设备id的场景需要手动调用
		SetDeviceId(deviceId string)
		// 关闭会话
		Close() error
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

// 编辑码器配置，有些编辑码需要提供一些默认的配置这样产品添加时就不需要手动加配置
type CodecMetaConfig struct {
	MetaConfigs []MetaConfig
	CodecId     string
}

func (p CodecMetaConfig) ToJson() string {
	b, _ := json.Marshal(p.MetaConfigs)
	return string(b)
}

// 元数据配置结构
type MetaConfig struct {
	Property string `json:"property,omitempty"`
	Type     string `json:"type,omitempty"`
	Value    string `json:"value,omitempty"`
	Buildin  bool   `json:"buildin,omitempty"`
	Desc     string `json:"desc,omitempty"`
}

// 产品
type Product struct {
	Id          string            `json:"id"`
	Config      map[string]string `json:"config"`
	StorePolicy string            `json:"storePolicy"`
	NetworkType string            `json:"networkType"`
	TslData     *tsl.TslData      `json:"-"`
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
		Config:    make(map[string]string),
	}
}

// 设备
type Device struct {
	Id         string            `json:"id"`
	ProductId  string            `json:"productId"`
	ParentId   string            `json:"parentId"`
	DeviceType string            `json:"devType"`
	ClusterId  string            `json:"clusterId"` // 所在集群id
	CreateId   int64             `json:"-"`
	Config     map[string]string `json:"-"`
	Name       string            `json:"name"`
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

// 获取临时数据
func (d *Device) GetData(key string) string {
	return GetDeviceData(d.Id, key)
}

// 获取设备临时数据int
func (d *Device) GetDataInt(key string) int {
	i, err := strconv.ParseInt(GetDeviceData(d.Id, key), 10, 64)
	if err != nil {
		logs.Errorf("device [%s] GetDataInt error: %v", d.Id, err)
	}
	return int(i)
}

// 设置临时数据
func (d *Device) SetData(key string, val string) {
	SetDeviceData(d.Id, key, val)
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

// 设置设备配置
func (d *Device) SetConfig(key string, value string) {
	d.Config[key] = value
}

// 是否子设备
func (d *Device) IsSubDevice() bool {
	return d.DeviceType == SUBDEVICE
}

// debug输出日志
func (d *Device) Debug(v any) {
	DebugLog(d.Id, d.ProductId, fmt.Sprintf("%v", v))
}

// 调试日志
func DebugLog(deviceId, productId string, v string) {
	if deviceId == "" {
		deviceId = "-"
	}
	eventbus.PublishDebug(eventbus.NewDebugMessage(deviceId, productId, v))
}

// base context
type BaseContext struct {
	DeviceId  string  `json:"deviceId"`
	ProductId string  `json:"productId"`
	Session   Session `json:"-"`
	device    *Device `json:"-"`
}

// 设备上线，调用后设备状态改为上线
func (ctx *BaseContext) DeviceOnline(deviceId string) {
	deviceId = strings.TrimSpace(deviceId)
	if len(deviceId) > 0 {
		oldSession := GetSession(deviceId)
		replace := false
		if oldSession != nil && oldSession != ctx.GetSession() {
			replace = true
			logs.Infof("device [%s] a new connection come in, old session close", deviceId)
			oldSession.Close()
		}
		device := GetDevice(deviceId)
		if device == nil {
			logs.Warnf("device [%s] not exist or noActive, close session", deviceId)
			ctx.GetSession().Close()
			return
		}
		ctx.DeviceId = deviceId
		ctx.GetSession().SetDeviceId(deviceId)
		PutSession(deviceId, ctx.GetSession(), replace)
	}
}

func (ctx *BaseContext) GetDevice() *Device {
	if ctx.device != nil {
		return ctx.device
	}
	return ctx.GetDeviceById(ctx.DeviceId)
}

func (ctx *BaseContext) GetDeviceById(deviceId string) *Device {
	d := GetDevice(deviceId)
	if d != nil {
		ctx.device = d
	}
	return d
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

// 获取产品配置
func (ctx *BaseContext) GetConfig(key string) string {
	device := ctx.GetDevice()
	if device == nil {
		return ""
	}
	return device.GetConfig(key)
}

// 保存设备属性的时序数据
func (ctx *BaseContext) SaveProperties(data map[string]interface{}) {
	p := ctx.GetProduct()
	if p == nil {
		logs.Warnf("product [%s] not exist or noActive", ctx.ProductId)
		return
	}
	if ctx.GetDevice() != nil {
		data["deviceId"] = ctx.DeviceId
	}
	p.GetTimeSeries().SaveProperties(p, data)
}

// 保存设备事件的时序数据
func (ctx *BaseContext) SaveEvents(eventId string, data any) {
	p := ctx.GetProduct()
	if p == nil {
		logs.Warnf("product [%s] not exist or noActive", ctx.ProductId)
		return
	}
	saveData := map[string]any{}
	switch d := data.(type) {
	case map[string]any:
		saveData = d
	default:
		saveData[eventId] = data
	}
	if ctx.GetDevice() != nil {
		saveData["deviceId"] = ctx.DeviceId
	}
	p.GetTimeSeries().SaveEvents(p, eventId, saveData)
}

func (ctx *BaseContext) ReplyOk() {
	replyMap.reply(ctx.DeviceId, &FuncInvokeReply{Success: true})
}

func (ctx *BaseContext) ReplyFail(resp string) {
	replyMap.reply(ctx.DeviceId, &FuncInvokeReply{Success: false, Msg: resp})
}

// 异步消息的回复
func (ctx *BaseContext) ReplyAsync(resp map[string]any) {
	reply := &FuncInvokeReply{Success: true}
	msg, ok := resp["msg"]
	if ok {
		reply.Msg = fmt.Sprintf("%v", msg)
	}
	if fmt.Sprintf("%v", resp["success"]) == "false" {
		reply.Success = false
	}
	traceId, ok := resp["traceId"]
	if ok {
		reply.TraceId = fmt.Sprintf("%v", traceId)
	}
	replyLogAsync(ctx.GetProduct(), ctx.DeviceId, reply)
}
