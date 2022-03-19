package modelfactory

import (
	led "go-iot/models/device"
	"go-iot/models/operates"

	"github.com/beego/beego/v2/core/logs"
)

var onlineChannel = make(chan operates.DeviceOnlineStatus, 10)

func init() {
	// 监听设备在线状态，并修改数据库中的状态
	go func() {
		// 处理在线状态事件
		for {
			o := <-onlineChannel
			logs.Info("UpdateOnlineStatus")
			if o.Type == "agent" {
				// err := agent.UpdateOnlineStatus(o.OnlineStatus, o.Sn)
				// if err != nil {
				// 	logs.Error(err.Error())
				// }
			} else {
				led.UpdateOnlineStatus(o.OnlineStatus, o.Sn, o.Provider)
			}
		}
	}()
}

// 发布在线状态事件
func FireOnlineStatus(o operates.DeviceOnlineStatus) {
	onlineChannel <- o
}

func GetDevice(id string) (operates.Device, error) {
	var dev operates.Device
	l, err := led.GetDevice(id)
	if err != nil {
		return dev, err
	}
	dev = operates.Device{Id: l.Id, Name: l.Name}
	return dev, nil
}

func GetDeviceByProvider(sn, provider string) (operates.Device, error) {
	var dev operates.Device
	l, err := led.GetDeviceByProvider(sn, provider)
	if err != nil {
		return dev, err
	}
	dev = operates.Device{Id: l.Id, Name: l.Name}
	return dev, nil
}
