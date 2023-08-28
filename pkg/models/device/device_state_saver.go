package models

import (
	"go-iot/pkg/core"
	"go-iot/pkg/eventbus"
	"sync"
	"time"
)

type deviceState struct {
	deviceId   string
	productId  string
	state      string
	createTime string
}

func init() {
	stateSaver := &deviceStateSaver{stateCh: make(chan deviceState, 1000)}

	var newDeviceState = func(deviceId, productId, state string) deviceState {
		return deviceState{deviceId: deviceId,
			productId:  productId,
			state:      state,
			createTime: time.Now().Format("2006-01-02 15:04:05.000")}
	}
	eventbus.Subscribe(eventbus.GetOfflineTopic("*", "*"), func(msg eventbus.Message) {
		if m, ok := msg.(*eventbus.OfflineMessage); ok {
			stateSaver.stateCh <- newDeviceState(m.DeviceId, m.ProductId, core.OFFLINE)
		}
	})
	eventbus.Subscribe(eventbus.GetOnlineTopic("*", "*"), func(msg eventbus.Message) {
		if m, ok := msg.(*eventbus.OnlineMessage); ok {
			stateSaver.stateCh <- newDeviceState(m.DeviceId, m.ProductId, core.ONLINE)
		}
	})
	go stateSaver.saveState()
}

type deviceStateSaver struct {
	sync.RWMutex
	stateCh chan deviceState
}

func (m *deviceStateSaver) saveState() {
	var onlineList []deviceState
	var offlineList []deviceState

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
	ticker := time.NewTicker(time.Millisecond * 3000)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C: // every 3 sec save data
			onlineFn(0)
			offlineFn(0)
		case dev := <-m.stateCh:
			if dev.state == core.ONLINE {
				onlineList = append(onlineList, dev)
				onlineFn(100)
			} else if dev.state == core.OFFLINE {
				offlineList = append(offlineList, dev)
				offlineFn(100)
			}
		}
	}
}

func updateOnlineStatus(list []deviceState, state string) {
	if len(list) > 0 {
		var ids []string
		for _, m := range list {
			ids = append(ids, m.deviceId)
			product := core.GetProduct(m.productId)
			if product != nil {
				data := core.LogData{
					DeviceId:   m.deviceId,
					Type:       state,
					CreateTime: m.createTime,
					Content:    `{"deviceId": "` + m.deviceId + `", "state": "` + state + `"}`,
				}
				product.GetTimeSeries().SaveLogs(product, data)
			}
		}
		UpdateOnlineStatusList(ids, state)
	}
}
