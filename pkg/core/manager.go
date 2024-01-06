package core

import (
	"go-iot/pkg/eventbus"
	"sync"
)

// session
var sessionManager sync.Map

// 从Session管理器中获取设备Session
func GetSession(deviceId string) Session {
	if val, ok := sessionManager.Load(deviceId); ok {
		return val.(Session)
	}
	return nil
}

// 将设备Session放入到Session管理器中
func PutSession(deviceId string, session Session, replace bool) {
	sessionManager.Store(deviceId, session)
	device := GetDevice(deviceId)
	if device != nil && !replace {
		evt := eventbus.NewOnlineMessage(deviceId, device.GetProductId())
		eventbus.PublishOnline(&evt)
	}
}

// 从Session管理器中删除设备Session
func DelSession(deviceId string) {
	if _, ok := sessionManager.LoadAndDelete(deviceId); ok {
		device := GetDevice(deviceId)
		if device != nil {
			evt := eventbus.NewOfflineMessage(deviceId, device.GetProductId())
			eventbus.PublishOffline(&evt)
		}
	}
}

// 设备存储器，保存已发布的设备、产品，mem, redis
var defaultStore DeviceStore

// 注册设备存储器
func RegDeviceStore(c DeviceStore) {
	defaultStore = c
}

// 获取设备
func GetDevice(deviceId string) *Device {
	return defaultStore.GetDevice(deviceId)
}

// 将设备放入存储器
func PutDevice(device *Device) {
	defaultStore.PutDevice(device)
}

// 将设备从存储器中删除
func DeleteDevice(deviceId string) {
	defaultStore.DelDevice(deviceId)
}

// 获取设备数据
func GetDeviceData(deviceId, key string) string {
	return defaultStore.GetDeviceData(deviceId, key)
}

// 设置设备数据
func SetDeviceData(deviceId, key string, val string) {
	defaultStore.SetDeviceData(deviceId, key, val)
}

// 获取产品
func GetProduct(productId string) *Product {
	return defaultStore.GetProduct(productId)
}

// 将产品放入存储器
func PutProduct(product *Product) {
	defaultStore.PutProduct(product)
}

// 将产品从存储器中删除
func DeleteProduct(productId string) {
	defaultStore.DelProduct(productId)
}

// DeviceStore 设备存储器，保存已发布的设备、产品，mem, redis
type DeviceStore interface {
	Id() string
	GetDevice(deviceId string) *Device
	PutDevice(device *Device)
	DelDevice(deviceId string)
	// 获取设备数据
	GetDeviceData(deviceId, key string) string
	// 设置设备数据
	SetDeviceData(deviceId, key string, val any)
	// 获取产品
	GetProduct(productId string) *Product
	// 保存产品
	PutProduct(product *Product)
	// 删除产品
	DelProduct(productId string)
}
