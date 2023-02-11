package codec

import (
	"go-iot/pkg/codec/eventbus"
	"sync"
)

func init() {
	RegDeviceManager(&memDeviceManager{cache: make(map[string]*Device)})
	RegProductManager(&memProductManager{cache: make(map[string]*Product)})
}

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

// db, mem, redis
var DefaultManagerId = "mem"

// device and product manager
var deviceManagerMap map[string]DeviceManager = map[string]DeviceManager{}
var productManagerMap map[string]ProductManager = map[string]ProductManager{}

// get device from deviceManager
func GetDevice(deviceId string) *Device {
	manager := deviceManagerMap[DefaultManagerId]
	if manager != nil {
		return manager.Get(deviceId)
	}
	return nil
}

// add device to deviceManager
func PutDevice(device *Device) {
	manager := deviceManagerMap[DefaultManagerId]
	if manager != nil {
		manager.Put(device)
	}
}

// delete device for deviceManager
func DeleteDevice(deviceId string) {
	manager := deviceManagerMap[DefaultManagerId]
	if manager != nil {
		manager.Del(deviceId)
	}
}

// get product from productManager
func GetProduct(productId string) *Product {
	manager := productManagerMap[DefaultManagerId]
	if manager != nil {
		return manager.Get(productId)
	}
	return nil
}

// add product to productManager
func PutProduct(product *Product) {
	manager := productManagerMap[DefaultManagerId]
	if manager != nil {
		manager.Put(product)
	}
}

// delete product from productManager
func DeleteProduct(productId string) *Product {
	manager := productManagerMap[DefaultManagerId]
	if manager != nil {
		manager.Del(productId)
	}
	return nil
}

func RegDeviceManager(m DeviceManager) {
	deviceManagerMap[m.Id()] = m
}
func RegProductManager(m ProductManager) {
	productManagerMap[m.Id()] = m
}

// DeviceManager
type DeviceManager interface {
	Id() string
	Get(deviceId string) *Device
	Put(device *Device)
	Del(deviceId string)
}

// ProductManager
type ProductManager interface {
	Id() string
	Get(productId string) *Product
	Put(product *Product)
	Del(productId string)
}

// memDeviceManager
type memDeviceManager struct {
	sync.RWMutex
	cache map[string]*Device
}

func (p *memDeviceManager) Id() string {
	return "mem"
}

func (m *memDeviceManager) Get(deviceId string) *Device {
	device, ok := m.cache[deviceId]
	if ok {
		return device
	}
	return device
}

func (m *memDeviceManager) Put(device *Device) {
	if device == nil {
		panic("device not be nil")
	}
	m.cache[device.GetId()] = device
}

func (m *memDeviceManager) Del(deviceId string) {
	delete(m.cache, deviceId)
}

// memProductManager
type memProductManager struct {
	sync.RWMutex
	cache map[string]*Product
}

func (p *memProductManager) Id() string {
	return "mem"
}

func (m *memProductManager) Get(productId string) *Product {
	product, ok := m.cache[productId]
	if ok {
		return product
	}
	return product
}

func (m *memProductManager) Put(product *Product) {
	if product == nil {
		panic("product not be nil")
	}
	if len(product.GetId()) == 0 {
		panic("product id must be present")
	}
	m.cache[product.GetId()] = product
}

func (m *memProductManager) Del(productId string) {
	delete(m.cache, productId)
}
