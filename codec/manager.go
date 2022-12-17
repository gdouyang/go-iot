package codec

import (
	"go-iot/codec/eventbus"
	"go-iot/codec/tsl"
	"sync"
)

func init() {
	RegDeviceManager(&memDeviceManager{cache: make(map[string]Device)})
	RegProductManager(&memProductManager{cache: make(map[string]Product)})
}

// session
var sessionManager *SessionManager = &SessionManager{}

func GetSessionManager() *SessionManager {
	return sessionManager
}

type SessionManager struct {
	sessionMap sync.Map
}

func (sm *SessionManager) Get(deviceId string) Session {
	if val, ok := sm.sessionMap.Load(deviceId); ok {
		return val.(Session)
	}
	return nil
}

func (sm *SessionManager) Put(deviceId string, session Session) {
	sm.sessionMap.Store(deviceId, session)
	device := GetDeviceManager().Get(deviceId)
	if device != nil {
		evt := eventbus.NewOnlineMessage(deviceId, device.GetProductId())
		eventbus.PublishOnline(&evt)
	}
}

func (sm *SessionManager) DelLocal(deviceId string) {
	if val, ok := sm.sessionMap.LoadAndDelete(deviceId); ok {
		sess := val.(Session)
		sess.Disconnect()
		device := GetDeviceManager().Get(deviceId)
		if device != nil {
			evt := eventbus.NewOfflineMessage(deviceId, device.GetProductId())
			eventbus.PublishOffline(&evt)
		}
	}
}

// db, mem, redis
var DefaultManagerId = "db"

// device and product manager
var deviceManagerMap map[string]DeviceManager = map[string]DeviceManager{}
var productManagerMap map[string]ProductManager = map[string]ProductManager{}

func GetDeviceManager() DeviceManager {
	return deviceManagerMap[DefaultManagerId]
}

func GetProductManager() ProductManager {
	return productManagerMap[DefaultManagerId]
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
	Get(deviceId string) Device
	Put(device Device)
}

// ProductManager
type ProductManager interface {
	Id() string
	Get(productId string) Product
	Put(product Product)
}

type DefaultDevice struct {
	Id        string
	ProductId string
	Data      map[string]string
	Config    map[string]string
}

func NewDevice(devieId string, productId string) *DefaultDevice {
	return &DefaultDevice{
		Id:        devieId,
		ProductId: productId,
		Data:      make(map[string]string),
		Config:    make(map[string]string),
	}
}

func (d *DefaultDevice) GetId() string {
	return d.Id
}
func (d *DefaultDevice) GetProductId() string {
	return d.ProductId
}
func (d *DefaultDevice) GetSession() Session {
	s := GetSessionManager().Get(d.Id)
	return s
}
func (d *DefaultDevice) GetData() map[string]string {
	return d.Data
}
func (d *DefaultDevice) GetConfig(key string) string {
	if v, ok := d.Config[key]; ok {
		return v
	}
	p := GetProductManager().Get(d.ProductId)
	if p != nil {
		v := p.GetConfig(key)
		return v
	}
	return ""
}

type DefaultProdeuct struct {
	Id          string
	Config      map[string]string
	StorePolicy string
	TslData     *tsl.TslData
}

func NewProduct(id string, config map[string]string, storePolicy string, tsltext string) (*DefaultProdeuct, error) {
	tslData := tsl.NewTslData()
	if len(tsltext) > 0 {
		err := tslData.FromJson(tsltext)
		if err != nil {
			return nil, err
		}
	}
	return &DefaultProdeuct{
		Id:          id,
		Config:      config,
		StorePolicy: storePolicy,
		TslData:     tslData,
	}, nil
}

func (p *DefaultProdeuct) GetId() string {
	return p.Id
}
func (p *DefaultProdeuct) GetConfig(key string) string {
	if v, ok := p.Config[key]; ok {
		return v
	}
	return ""
}

func (p *DefaultProdeuct) GetTimeSeries() TimeSeriesSave {
	return GetTimeSeries(p.StorePolicy)
}

func (p *DefaultProdeuct) GetTsl() *tsl.TslData {
	return p.TslData
}

// memDeviceManager
type memDeviceManager struct {
	sync.RWMutex
	cache map[string]Device
}

func (p *memDeviceManager) Id() string {
	return "mem"
}

func (m *memDeviceManager) Get(deviceId string) Device {
	device, ok := m.cache[deviceId]
	if ok {
		return device
	}
	return device
}

func (m *memDeviceManager) Put(device Device) {
	if device == nil {
		panic("device not be nil")
	}
	m.cache[device.GetId()] = device
}

// memProductManager
type memProductManager struct {
	sync.RWMutex
	cache map[string]Product
}

func (p *memProductManager) Id() string {
	return "mem"
}

func (m *memProductManager) Get(productId string) Product {
	product, ok := m.cache[productId]
	if ok {
		return product
	}
	return product
}

func (m *memProductManager) Put(product Product) {
	if product == nil {
		panic("product not be nil")
	}
	if len(product.GetId()) == 0 {
		panic("product id must be present")
	}
	m.cache[product.GetId()] = product
}
