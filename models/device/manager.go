package models

import (
	"encoding/json"
	"go-iot/codec"
	"go-iot/codec/tsl"
	"sync"

	"github.com/beego/beego/v2/core/logs"
)

func init() {
	codec.RegDeviceManager(&DbDeviceManager{cache: make(map[string]codec.Device)})
	codec.RegProductManager(&DbProductManager{cache: make(map[string]codec.Product)})
}

// DbDeviceManager
type DbDeviceManager struct {
	sync.RWMutex
	cache map[string]codec.Device
}

func (p *DbDeviceManager) Id() string {
	return "db"
}

func (m *DbDeviceManager) Get(deviceId string) codec.Device {
	device, ok := m.cache[deviceId]
	if ok {
		return device
	}
	if device == nil {
		m.Lock()
		defer m.Unlock()
		data, _ := GetDevice(deviceId)
		if data == nil {
			m.cache[deviceId] = nil
			return nil
		}
		config := map[string]string{}
		if len(data.MetaConfig) > 0 {
			err := json.Unmarshal([]byte(data.MetaConfig), &config)
			if (err) != nil {
				logs.Error(err)
			}
		}

		device = &codec.DefaultDevice{
			Id:        data.Id,
			ProductId: data.ProductId,
			Config:    config,
			Data:      map[string]string{},
		}
		m.Put(device)
	}
	return device
}

func (m *DbDeviceManager) Put(device codec.Device) {
	m.cache[device.GetId()] = device
}

// DbProductManager
type DbProductManager struct {
	sync.RWMutex
	cache map[string]codec.Product
}

func (p *DbProductManager) Id() string {
	return "db"
}

func (m *DbProductManager) Get(productId string) codec.Product {
	product, ok := m.cache[productId]
	if ok {
		return product
	}
	if product == nil {
		m.Lock()
		defer m.Unlock()
		data, _ := GetProduct(productId)
		if data == nil {
			m.cache[productId] = nil
			return nil
		}
		d := tsl.TslData{}
		err := d.FromJson(data.MetaData)
		if err != nil {
			logs.Error(err)
		}
		tt := map[string]tsl.TslProperty{}
		for _, p := range d.Properties {
			tt[p.Id] = p
		}

		product = &codec.DefaultProdeuct{
			Id:           data.Id,
			Config:       map[string]string{},
			TimeSeriesId: codec.TIME_SERISE_ES,
			TslProperty:  tt,
		}
		m.Put(product)
	}
	return product
}

func (m *DbProductManager) Put(product codec.Product) {
	if product == nil {
		panic("product not be nil")
	}
	if len(product.GetId()) == 0 {
		panic("product id not be empty")
	}
	m.cache[product.GetId()] = product
}
