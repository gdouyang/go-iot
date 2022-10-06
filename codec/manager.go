package codec

import (
	"go-iot/codec/tsl"
	"sync"
)

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

// device and product
var deviceManagerMap map[string]DeviceManager = map[string]DeviceManager{}
var productManagerMap map[string]ProductManager = map[string]ProductManager{}

func GetDeviceManager() DeviceManager {
	return deviceManagerMap["db"]
}

func GetProductManager() ProductManager {
	return productManagerMap["db"]
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
	TslProperty  map[string]tsl.TslProperty
}

func (p *DefaultProdeuct) GetId() string {
	return p.Id
}
func (p *DefaultProdeuct) GetConfig() map[string]interface{} {
	return p.Config
}

func (p *DefaultProdeuct) GetTimeSeries() TimeSeriesSave {
	return GetTimeSeries(p.TimeSeriesId)
}
func (p *DefaultProdeuct) GetTslProperty() map[string]tsl.TslProperty {
	return p.TslProperty
}
