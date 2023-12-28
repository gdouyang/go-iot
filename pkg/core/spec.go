package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"go-iot/pkg/core/common"
	"go-iot/pkg/core/tsl"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

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
		GetDeviceId() string
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

type CodecMetaConfig struct {
	MetaConfigs []MetaConfig
	CodecId     string
}

func (p CodecMetaConfig) ToJson() string {
	b, _ := json.Marshal(p.MetaConfigs)
	return string(b)
}

// the meta config
type MetaConfig struct {
	Property string `json:"property,omitempty"`
	Type     string `json:"type,omitempty"`
	Value    string `json:"value,omitempty"`
	Buildin  bool   `json:"buildin,omitempty"`
	Desc     string `json:"desc,omitempty"`
}

// default product impl
type Product struct {
	Id          string            `json:"id"`
	Config      map[string]string `json:"config"`
	StorePolicy string            `json:"storePolicy"`
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
		Data:      sync.Map{},
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
	Data       sync.Map          `json:"-"`
	Config     map[string]string `json:"config"`
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
func (d *Device) GetData(key string) any {
	v, _ := d.Data.Load(key)
	return v
}

// 设置临时数据
func (d *Device) SetData(key string, val any) {
	d.Data.Store(key, val)
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

// 是否子设备
func (d *Device) IsSubDevice() bool {
	return d.DeviceType == SUBDEVICE
}

// base context
type BaseContext struct {
	DeviceId  string  `json:"deviceId"`
	ProductId string  `json:"productId"`
	Session   Session `json:"-"`
	device    *Device `json:"-"`
}

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
		logs.Warnf("product [%s] not exist or noActive", ctx.ProductId)
		return
	}
	if ctx.GetDevice() != nil {
		data["deviceId"] = ctx.DeviceId
	}
	p.GetTimeSeries().SaveProperties(p, data)
}

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
	replyMap.reply(ctx.DeviceId, &common.FuncInvokeReply{Success: true})
}

func (ctx *BaseContext) ReplyFail(resp string) {
	replyMap.reply(ctx.DeviceId, &common.FuncInvokeReply{Success: false, Msg: resp})
}

// 异步消息的回复
func (ctx *BaseContext) ReplyAsync(resp map[string]any) {
	reply := &common.FuncInvokeReply{Success: true}
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

func (ctx *BaseContext) HttpRequest(config map[string]interface{}) map[string]interface{} {
	return HttpRequest(config)
}

// http request func for http network
func HttpRequest(config map[string]interface{}) map[string]interface{} {
	result := map[string]interface{}{}
	path := config["url"]
	u, err := url.ParseRequestURI(fmt.Sprintf("%v", path))
	if err != nil {
		logs.Errorf(err.Error())
		result["status"] = 400
		result["message"] = err.Error()
		return result
	}
	method := fmt.Sprintf("%v", config["method"])
	client := http.Client{Timeout: time.Second * 3}
	var req *http.Request = &http.Request{
		Method: strings.ToUpper(method),
		URL:    u,
		Header: map[string][]string{},
	}
	if v, ok := config["headers"]; ok {
		h, ok := v.(map[string]interface{})
		if !ok {
			logs.Warnf("headers is not object: %v", v)
			h = map[string]interface{}{}
		}
		for key, value := range h {
			req.Header.Add(key, fmt.Sprintf("%v", value))
		}
	}
	if strings.ToLower(method) == "post" && (len(req.Header.Get("Content-Type")) == 0 || len(req.Header.Get("content-type")) == 0) {
		req.Header.Add("Content-Type", "application/json; charset=utf-8")
	}
	if data, ok := config["data"]; ok {
		if body, ok := data.(map[string]interface{}); ok {
			b, err := json.Marshal(body)
			if err != nil {
				logs.Errorf("http data parse error: %v", err)
				result["status"] = 400
				result["message"] = err.Error()
				return result
			}
			req.Body = io.NopCloser(strings.NewReader(string(b)))
		} else {
			req.Body = io.NopCloser(strings.NewReader(fmt.Sprintf("%v", data)))
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		logs.Errorf(err.Error())
		result["status"] = 0
		result["message"] = err.Error()
		return result
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		logs.Errorf(err.Error())
		result["status"] = 400
		result["message"] = err.Error()
		return result
	}
	header := map[string]string{}
	if resp.Header != nil {
		for key := range resp.Header {
			header[key] = resp.Header.Get(key)
		}
	}
	result["data"] = string(b)
	result["status"] = resp.StatusCode
	result["header"] = header
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		result["message"] = string(b)
	}
	return result
}
