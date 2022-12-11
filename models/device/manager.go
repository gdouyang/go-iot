package models

import (
	"go-iot/codec"
	"go-iot/codec/eventbus"
	"go-iot/models"
	"sync"

	"github.com/beego/beego/v2/core/logs"
)

func init() {
	codec.RegDeviceManager(&DbDeviceManager{cache: make(map[string]codec.Device)})
	codec.RegProductManager(&DbProductManager{cache: make(map[string]codec.Product)})
	eventbus.Subscribe(eventbus.GetOfflineTopic("*", "*"), func(msg eventbus.Message) {
		if m, ok := msg.(*eventbus.OfflineMessage); ok {
			UpdateOnlineStatus(m.DeviceId, models.OFFLINE)
		}
	})
	eventbus.Subscribe(eventbus.GetOnlineTopic("*", "*"), func(msg eventbus.Message) {
		if m, ok := msg.(*eventbus.OnlineMessage); ok {
			UpdateOnlineStatus(m.DeviceId, models.ONLINE)
		}
	})
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

		device = &codec.DefaultDevice{
			Id:        data.Id,
			ProductId: data.ProductId,
			Config:    data.Metaconfig,
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
		config := map[string]string{}
		for _, item := range data.Metaconfig {
			config[item.Property] = item.Value
		}
		storePolicy := data.StorePolicy
		if len(storePolicy) == 0 {
			storePolicy = codec.TIME_SERISE_ES
		}
		produ, err := codec.NewProduct(data.Id, config, data.StorePolicy, data.Metadata)
		if err != nil {
			logs.Error("newProduct error: ", err)
		} else {
			product = produ
			m.Put(produ)
		}
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
