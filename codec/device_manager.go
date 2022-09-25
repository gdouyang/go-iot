package codec

import (
	"errors"
	"go-iot/codec/msg"
)

var deviceManagerIns DeviceManager = DeviceManager{m: make(map[string]Device)}
var productManager ProductManager = ProductManager{m: make(map[string]Product)}

func GetDeviceManager() *DeviceManager {
	return &deviceManagerIns
}

func GetProductManager() *ProductManager {
	return &productManager
}

// DeviceManager
type DeviceManager struct {
	m map[string]Device
}

func (dm *DeviceManager) Get(deviceId string) Device {
	device := dm.m[deviceId]
	return device
}

func (dm *DeviceManager) Put(device Device) {
	dm.m[device.GetId()] = device
}

// ProductManager
type ProductManager struct {
	m map[string]Product
}

func (pm *ProductManager) Get(productId string) Product {
	product := pm.m[productId]
	return product
}

func (pm *ProductManager) Put(product Product) {
	if product == nil {
		panic("product not be nil")
	}
	if len(product.GetId()) == 0 {
		panic("product id not be empty")
	}
	pm.m[product.GetId()] = product
}

type DefaultDevice struct {
	Id        string
	ProductId string
	Data      map[string]interface{}
	Config    map[string]interface{}
}

func (d *DefaultDevice) GetId() string {
	return d.Id
}
func (d *DefaultDevice) GetSession() Session {
	s := GetSessionManager().Get(d.Id)
	return s
}
func (d *DefaultDevice) GetData() map[string]interface{} {
	return d.Data
}
func (d *DefaultDevice) GetConfig() map[string]interface{} {
	return d.Config
}

type DefaultProdeuct struct {
	Id           string
	Config       map[string]interface{}
	TimeSeriesId string
}

func (p *DefaultProdeuct) GetId() string {
	return p.Id
}
func (p *DefaultProdeuct) GetConfig() map[string]interface{} {
	return p.Config
}

func (p *DefaultProdeuct) GetTimeSeries() TimeSeries {
	return GetTimeSeries(p.TimeSeriesId)
}

// 进行功能调用
func DoFuncInvoke(productId string, message msg.FuncInvoke) error {
	session := sessionManager.Get(message.DeviceId)
	if session == nil {
		return errors.New("设备不在线")
	}
	codec := GetCodec(productId)
	codec.Decode(&FuncInvokeContext{session: session,
		deviceId: message.DeviceId, productId: productId})
	return nil
}
