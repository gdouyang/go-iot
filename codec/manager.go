package codec

import (
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
}

func (sm *SessionManager) DelLocal(deviceId string) {
	if val, ok := sm.sessionMap.LoadAndDelete(deviceId); ok {
		sess := val.(Session)
		sess.Disconnect()
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

func (d *DefaultDevice) GetId() string {
	return d.Id
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

func NewProduct(id string, config map[string]string, tsId string, tsltext string) (*DefaultProdeuct, error) {
	tslData := tsl.NewTslData()
	if len(tsltext) > 0 {
		err := tslData.FromJson(tsltext)
		if err != nil {
			return nil, err
		}
	}
	return &DefaultProdeuct{
		Id:           id,
		Config:       config,
		TimeSeriesId: tsId,
		tslData:      tslData,
	}, nil
}

type DefaultProdeuct struct {
	Id           string
	Config       map[string]string
	TimeSeriesId string
	tslData      *tsl.TslData
	TslFunction  map[string]tsl.TslFunction
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
	return GetTimeSeries(p.TimeSeriesId)
}

func (p *DefaultProdeuct) GetTsl() *tsl.TslData {
	return p.tslData
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
