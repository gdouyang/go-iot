package models

import (
	"go-iot/pkg/core"
	"go-iot/pkg/core/eventbus"
	"go-iot/pkg/models"
	"sync"
	"time"
)

func init() {
	deviceManager := &DbDeviceManager{cache: make(map[string]*core.Device), stateCh: make(chan models.Device, 1000)}
	eventbus.Subscribe(eventbus.GetOfflineTopic("*", "*"), func(msg eventbus.Message) {
		if m, ok := msg.(*eventbus.OfflineMessage); ok {
			deviceManager.stateCh <- models.Device{Id: m.DeviceId, State: core.OFFLINE}
		}
	})
	eventbus.Subscribe(eventbus.GetOnlineTopic("*", "*"), func(msg eventbus.Message) {
		if m, ok := msg.(*eventbus.OnlineMessage); ok {
			deviceManager.stateCh <- models.Device{Id: m.DeviceId, State: core.ONLINE}
		}
	})
	go deviceManager.saveState()
}

// DbDeviceManager
type DbDeviceManager struct {
	sync.RWMutex
	cache   map[string]*core.Device
	stateCh chan models.Device
}

func (m *DbDeviceManager) saveState() {
	var onlineList []models.Device
	var offlineList []models.Device

	var onlineFn = func(size int) {
		if len(onlineList) >= size {
			updateOnlineStatus(onlineList, core.ONLINE)
			onlineList = onlineList[:0]
		}
	}
	var offlineFn = func(size int) {
		if len(offlineList) >= size {
			updateOnlineStatus(offlineList, core.OFFLINE)
			offlineList = offlineList[:0]
		}
	}
	for {
		select {
		case <-time.After(time.Millisecond * 3000): // every 3 sec save data
			onlineFn(0)
			offlineFn(0)
		case dev := <-m.stateCh:
			if dev.State == core.ONLINE {
				onlineList = append(onlineList, dev)
				onlineFn(100)
			} else if dev.State == core.OFFLINE {
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
			product := core.GetProduct(m.ProductId)
			if product != nil {
				product.GetTimeSeries().SaveLogs(product, core.LogData{DeviceId: m.Id, Type: core.OFFLINE})
			}
		}
		UpdateOnlineStatusList(ids, state)
	}
}
