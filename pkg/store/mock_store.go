package store

import (
	"errors"
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
	if !ok || device == nil {
		return nil
	}
	return device.(*core.Device)
}

func (m *mockDeviceStore) PutDevice(device *core.Device) error {
	if device == nil {
		return errors.New("device not be nil")
	}
	if len(device.GetId()) == 0 {
		return errors.New("device id must be present")
	}
	m.cache.Store(device.GetId(), device)
	return nil
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

// RefreshOfflineTimeout 刷新设备过期时间
func (m *mockDeviceStore) RefreshOfflineTimeout(deviceId string) {
	// 模拟刷新设备过期时间
}

func (m *mockDeviceStore) GetProduct(productId string) *core.Product {
	product, ok := m.cache.Load(productId)
	if !ok || product == nil {
		return nil
	}
	return product.(*core.Product)
}

func (m *mockDeviceStore) PutProduct(product *core.Product) error {
	if product == nil {
		return errors.New("product not be nil")
	}
	if len(product.GetId()) == 0 {
		return errors.New("product id must be present")
	}
	m.cache.Store(product.GetId(), product)
	return nil
}

func (m *mockDeviceStore) DelProduct(productId string) {
	m.cache.Delete(productId)
}
