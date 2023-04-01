package store

import (
	"go-iot/pkg/core"
	"sync"
)

func NewMockDeviceStore() core.DeviceStore {
	return &mockDeviceStore{}
}

// mem device store
type mockDeviceStore struct {
	cache sync.Map
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
