package core

import (
	"errors"
	"go-iot/pkg/eventbus"
	"strconv"
	"sync"
)

// 从Session管理器中获取设备Session
func GetSession(deviceId string) Session {
	return defaultSessions.Get(deviceId)
}

// 判断设备是否连接断开, session存在也不意味着设备已连接
func IsDeviceDisconnect(deviceId string) bool {
	if GetSession(deviceId) == nil {
		return true
	}
	if v, ok := deviceDisconnect.Load(deviceId); ok {
		return v.(bool)
	}
	return false
}

// deviceIsDisconnect 标记：原存于 session.GetConInfo() map，上下线事件、超时检查、
// API 查询连接信息并发读写同一 map 会 fatal: concurrent map read and map write，
// 改为包级 sync.Map 按 deviceId 隔离（生命周期随 session 增删）
var deviceDisconnect sync.Map // deviceId -> bool

// 将设备Session放入到Session管理器中
func PutSession(deviceId string, session Session, sendOnlineEvent bool) {
	defaultSessions.Put(deviceId, session)
	deviceDisconnect.Store(deviceId, false)
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
		// key 与 session 同生命周期：无论 Delete 是否成功（会话可能已被 Redis 侧过期），
		// 标记都随之清除；缺失等价于 false（在线），不影响后续新会话
		deviceDisconnect.Delete(device.Id)
		if defaultSessions.Delete(device.Id) {
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
				if session != nil {
					deviceDisconnect.Store(device.Id, true)
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

// 设备存储器，保存已发布的设备、产品，mem, redis。
// 生产仅在启动期 Reg 一次；测试中每个 case 会 Reg，而上一个 case 残留的 session
// goroutine 可能仍在并发 GetDevice 读 defaultStore——无锁读写会被 race detector
// 判定 DATA RACE。用 RWMutex 保护读写以根治。
var (
	defaultStore DeviceStore
	storeMu      sync.RWMutex
)

// RegDeviceStore 注册设备存储器（nil 静默忽略）。
func RegDeviceStore(c DeviceStore) {
	if c == nil {
		return
	}
	storeMu.Lock()
	defaultStore = c
	storeMu.Unlock()
}

// getStore 返回当前存储器快照（可为 nil）。
func getStore() DeviceStore {
	storeMu.RLock()
	s := defaultStore
	storeMu.RUnlock()
	return s
}

// GetDeviceStore 返回当前设备存储器（可为 nil）。
func GetDeviceStore() DeviceStore {
	return getStore()
}

// 获取设备
func GetDevice(deviceId string) *Device {
	s := getStore()
	if s == nil {
		return nil
	}
	return s.GetDevice(deviceId)
}

// 将设备放入存储器
func PutDevice(device *Device) error {
	s := getStore()
	if s == nil {
		return errors.New("device store not registered")
	}
	return s.PutDevice(device)
}

// 将设备从存储器中删除
func DeleteDevice(deviceId string) {
	s := getStore()
	if s == nil {
		return
	}
	s.DelDevice(deviceId)
}

// 获取设备数据
func GetDeviceData(deviceId, key string) string {
	s := getStore()
	if s == nil {
		return ""
	}
	return s.GetDeviceData(deviceId, key)
}

// 设置设备数据
func SetDeviceData(deviceId, key string, val string) {
	s := getStore()
	if s == nil {
		return
	}
	s.SetDeviceData(deviceId, key, val)
}

// 获取产品
func GetProduct(productId string) *Product {
	s := getStore()
	if s == nil {
		return nil
	}
	return s.GetProduct(productId)
}

// 将产品放入存储器
func PutProduct(product *Product) error {
	s := getStore()
	if s == nil {
		return errors.New("device store not registered")
	}
	return s.PutProduct(product)
}

// 将产品从存储器中删除
func DeleteProduct(productId string) {
	s := getStore()
	if s == nil {
		return
	}
	s.DelProduct(productId)
}

// DeviceStore 设备存储器，保存已发布的设备、产品，mem, redis
type DeviceStore interface {
	// 设备存储器ID
	Id() string
	// 获取设备
	GetDevice(deviceId string) *Device
	// 保存设备
	PutDevice(device *Device) error
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
	PutProduct(product *Product) error
	// 删除产品
	DelProduct(productId string)
}
