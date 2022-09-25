package codec

import (
	"errors"
	"go-iot/codec/msg"
)

var deviceManagerIns DeviceManager = DeviceManager{}
var productManager ProductManager = ProductManager{m: map[string]Product{}}

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

func (dm *DeviceManager) Put(id string, device Device) {
	dm.m[id] = device
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

type defaultDevice struct {
	session Session
	data    map[string]interface{}
	config  map[string]interface{}
}

func (device *defaultDevice) GetSession() Session {
	return device.session
}
func (device *defaultDevice) GetData() map[string]interface{} {
	return device.data
}
func (device *defaultDevice) GetConfig() map[string]interface{} {
	return device.config
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
	session := sessionManager.GetSession(message.DeviceId)
	if session == nil {
		return errors.New("设备不在线")
	}
	codec := GetCodec(productId)
	codec.Decode(&FuncInvokeContext{session: session,
		deviceId: message.DeviceId, productId: productId})
	return nil
}
