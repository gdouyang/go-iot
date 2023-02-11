package models

import (
	"go-iot/pkg/codec"
	"go-iot/pkg/codec/eventbus"
	"go-iot/pkg/models"
	"sync"
	"time"

	"github.com/beego/beego/v2/core/logs"
)

func init() {
	deviceManager := &DbDeviceManager{cache: make(map[string]*codec.Device), stateCh: make(chan models.Device, 1000)}
	codec.RegDeviceManager(deviceManager)
	codec.RegProductManager(&DbProductManager{cache: make(map[string]*codec.Product)})
	eventbus.Subscribe(eventbus.GetOfflineTopic("*", "*"), func(msg eventbus.Message) {
		if m, ok := msg.(*eventbus.OfflineMessage); ok {
			deviceManager.stateCh <- models.Device{Id: m.DeviceId, State: models.OFFLINE}
		}
	})
	eventbus.Subscribe(eventbus.GetOnlineTopic("*", "*"), func(msg eventbus.Message) {
		if m, ok := msg.(*eventbus.OnlineMessage); ok {
			deviceManager.stateCh <- models.Device{Id: m.DeviceId, State: models.ONLINE}
		}
	})
	go deviceManager.saveState()
}

// DbDeviceManager
type DbDeviceManager struct {
	sync.RWMutex
	cache   map[string]*codec.Device
	stateCh chan models.Device
}

func (p *DbDeviceManager) Id() string {
	return "db"
}

func (m *DbDeviceManager) Get(deviceId string) *codec.Device {
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

		device = &codec.Device{
			Id:        data.Id,
			ProductId: data.ProductId,
			Config:    data.Metaconfig,
			Data:      map[string]string{},
		}
		m.Put(device)
	}
	return device
}

func (m *DbDeviceManager) Put(device *codec.Device) {
	m.cache[device.GetId()] = device
}

func (m *DbDeviceManager) Del(deviceId string) {

}
func (m *DbDeviceManager) saveState() {
	var onlineList []models.Device
	var offlineList []models.Device

	var onlineFn = func(size int) {
		if len(onlineList) >= size {
			updateOnlineStatus(onlineList, models.ONLINE)
			onlineList = onlineList[:0]
		}
	}
	var offlineFn = func(size int) {
		if len(offlineList) >= size {
			updateOnlineStatus(offlineList, models.OFFLINE)
			offlineList = offlineList[:0]
		}
	}
	for {
		select {
		case <-time.After(time.Millisecond * 3000): // every 5 sec save data
			onlineFn(0)
			offlineFn(0)
		case dev := <-m.stateCh:
			if dev.State == models.ONLINE {
				onlineList = append(onlineList, dev)
				onlineFn(100)
			} else if dev.State == models.OFFLINE {
				offlineList = append(offlineList, dev)
				offlineFn(100)
			}
		}
	}
}

func updateOnlineStatus(list []models.Device, state string) {
	if len(list) > 0 {
		var ids []string
		for _, m := range list {
			ids = append(ids, m.Id)
			product := codec.GetProduct(m.ProductId)
			if product != nil {
				product.GetTimeSeries().SaveLogs(product, codec.LogData{DeviceId: m.Id, Type: models.OFFLINE})
			}
		}
		UpdateOnlineStatusList(ids, state)
	}
}

// DbProductManager
type DbProductManager struct {
	sync.RWMutex
	cache map[string]*codec.Product
}

func (p *DbProductManager) Id() string {
	return "db"
}

func (m *DbProductManager) Get(productId string) *codec.Product {
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
			m.cache[productId] = produ
			return produ
		}
	}
	return nil
}

func (m *DbProductManager) Put(product *codec.Product) {
	if product == nil {
		panic("product not be nil")
	}
	if len(product.GetId()) == 0 {
		panic("product id must be present")
	}
	m.cache[product.GetId()] = product
}

func (m *DbProductManager) Del(deviceId string) {

}
