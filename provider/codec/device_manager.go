package codec

import (
	"errors"
	"go-iot/provider/codec/msg"
)

var deviceManagerIns DeviceManager = DeviceManager{}
var productManager ProductManager = ProductManager{}

func GetDeviceManager() *DeviceManager {
	return &deviceManagerIns
}

func GetProductManager() *ProductManager {
	return &productManager
}

type DeviceManager struct {
	m map[string]Device
}

func (dm *DeviceManager) GetDevice(deviceId string) Device {
	device := dm.m[deviceId]
	return device
}

func (dm *DeviceManager) PutDevice(deviceId string, device Device) {
	dm.m[deviceId] = device
}

type ProductManager struct {
	m map[string]Product
}

func (pm *ProductManager) GetProduct(productId string) Product {
	product := pm.m[productId]
	return product
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
	config map[string]interface{}
}

func (product *DefaultProdeuct) GetConfig() map[string]interface{} {
	return product.config
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
