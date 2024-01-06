package store

import (
	"fmt"
	"go-iot/pkg/core"
	"sync"
)

func NewMockDeviceStore() core.DeviceStore {
	return &mockDeviceStore{cache: sync.Map{}, deviceData: map[string]map[string]any{}}
}

// mem device store
type mockDeviceStore struct {
	cache      sync.Map
	deviceData map[string]map[string]any
}

func (p *mockDeviceStore) Id() string {
	return "mock"
}

func (m *mockDeviceStore) GetDevice(deviceId string) *core.Device {
	device, ok := m.cache.Load(deviceId)
	if ok {
		return device.(*core.Device)
	}
	return device.(*core.Device)
}

func (m *mockDeviceStore) PutDevice(device *core.Device) {
	if device == nil {
		panic("device not be nil")
	}
	m.cache.Store(device.GetId(), device)
}

func (m *mockDeviceStore) DelDevice(deviceId string) {
	m.cache.Delete(deviceId)
}

func (m *mockDeviceStore) GetDeviceData(deviceId, key string) string {
	if v, ok := m.deviceData[deviceId]; ok {
		return fmt.Sprintf("%v", v[key])
	}
	return ""
}

func (m *mockDeviceStore) SetDeviceData(deviceId, key string, val any) {
	if v, ok := m.deviceData[deviceId]; ok {
		v[key] = val
	} else {
		m.deviceData[deviceId] = map[string]any{key: val}
	}
}

func (m *mockDeviceStore) GetProduct(productId string) *core.Product {
	product, ok := m.cache.Load(productId)
	if ok {
		return product.(*core.Product)
	}
	return nil
}

func (m *mockDeviceStore) PutProduct(product *core.Product) {
	if product == nil {
		panic("product not be nil")
	}
	if len(product.GetId()) == 0 {
		panic("product id must be present")
	}
	m.cache.Store(product.GetId(), product)
}

func (m *mockDeviceStore) DelProduct(productId string) {
	m.cache.Delete(productId)
}
