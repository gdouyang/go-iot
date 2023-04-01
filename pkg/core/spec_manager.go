package core

import (
	"go-iot/pkg/core/eventbus"
	"sync"
)

// session
var sessionManager sync.Map

// get session from sessionManager
func GetSession(deviceId string) Session {
	if val, ok := sessionManager.Load(deviceId); ok {
		return val.(Session)
	}
	return nil
}

// add session to sessionManager
func PutSession(deviceId string, session Session) {
	sessionManager.Store(deviceId, session)
	device := GetDevice(deviceId)
	if device != nil {
		evt := eventbus.NewOnlineMessage(deviceId, device.GetProductId())
		eventbus.PublishOnline(&evt)
	}
}

// del session from sessionManager
func DelSession(deviceId string) {
	if _, ok := sessionManager.LoadAndDelete(deviceId); ok {
		device := GetDevice(deviceId)
		if device != nil {
			evt := eventbus.NewOfflineMessage(deviceId, device.GetProductId())
			eventbus.PublishOffline(&evt)
		}
	}
}

// mem, redis
var defaultStore DeviceStore

func RegDeviceStore(c DeviceStore) {
	defaultStore = c
}

// get device from deviceManager
func GetDevice(deviceId string) *Device {
	return defaultStore.GetDevice(deviceId)
}

// add device to deviceManager
func PutDevice(device *Device) {
	defaultStore.PutDevice(device)
}

// delete device for deviceManager
func DeleteDevice(deviceId string) {
	defaultStore.DelDevice(deviceId)
}

// get product from productManager
func GetProduct(productId string) *Product {
	return defaultStore.GetProduct(productId)
}

// add product to productManager
func PutProduct(product *Product) {
	defaultStore.PutProduct(product)
}

// delete product from productManager
func DeleteProduct(productId string) {
	defaultStore.DelProduct(productId)
}

// DeviceStore
type DeviceStore interface {
	Id() string
	GetDevice(deviceId string) *Device
	PutDevice(device *Device)
	DelDevice(deviceId string)
	GetProduct(productId string) *Product
	PutProduct(product *Product)
	DelProduct(productId string)
}
