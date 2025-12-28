package core

import (
	"go-iot/pkg/eventbus"
	"strconv"
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

// 判断设备是否连接断开, session存在也不意味着设备已连接
func IsDeviceDisconnect(deviceId string) bool {
	session := GetSession(deviceId)
	if session == nil {
		return true
	}
	if session.GetConInfo() != nil {
		if v, ok := session.GetConInfo()[deviceIsDisconnect]; ok {
			return v.(bool)
		}
	}
	return false
}

// 将设备Session放入到Session管理器中
func PutSession(deviceId string, session Session, sendOnlineEvent bool) {
	sessionManager.Store(deviceId, session)
	device := GetDevice(deviceId)
	if device != nil && sendOnlineEvent {
		DeviceOnlineEvent(deviceId, device.GetProductId())
	}
	// 发送离线命令
	sendOfflineCommands(deviceId)
}

// 从Session管理器中删除设备Session, 并触发离线事件
func DelSessionByUserDisconnect(deviceId string) {
	DelSessionWithOfflineReason(deviceId, "disconnect")
}

func DelSessionWithOfflineReason(deviceId string, msg string) {
	device := GetDevice(deviceId)
	if device != nil {
		if _, ok := sessionManager.LoadAndDelete(device.Id); ok {
			DeviceOfflineEvent(device.Id, device.GetProductId(), msg)
		}
	}
}

// 判断设备是否配置了离线时间, 如果配置了离线超时时间，且超时时间大于0，则不触发离线事件,
// 否则从Session管理器中删除设备Session, 并触发离线事件
func DelSessionWithTimeoutCheck(deviceId string) {
	device := GetDevice(deviceId)
	if device != nil {
		timeoutStr := device.GetConfig(DEVICE_TIMEOUT_KEY)
		// 如果配置了离线超时时间，且超时时间大于0，则不触发离线事件
		if len(timeoutStr) > 0 {
			timeout, err := strconv.Atoi(timeoutStr)
			if err == nil && timeout > 0 {
				session := GetSession(device.Id)
				if session != nil && session.GetConInfo() != nil {
					session.GetConInfo()[deviceIsDisconnect] = true
				} else {
					DelSessionByUserDisconnect(deviceId)
				}
			}
		} else {
			DelSessionByUserDisconnect(deviceId)
		}
	}
}

// 设备上线事件
func DeviceOnlineEvent(deviceId, productId string) {
	evt := eventbus.NewOnlineMessage(deviceId, productId)
	eventbus.PublishOnline(&evt)
}

// 设备离线事件
func DeviceOfflineEvent(deviceId, productId string, message string) {
	evt := eventbus.NewOfflineMessage(deviceId, productId, message)
	eventbus.PublishOffline(&evt)
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
	// 设备存储器ID
	Id() string
	// 获取设备
	GetDevice(deviceId string) *Device
	// 保存设备
	PutDevice(device *Device)
	// 删除设备
	DelDevice(deviceId string)
	// 刷新设备过期时间
	RefreshOfflineTimeout(deviceId string)
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
