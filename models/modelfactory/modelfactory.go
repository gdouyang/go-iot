package modelfactory

import (
	"go-iot/models/agent"
	"go-iot/models/led"
	"go-iot/models/operates"

	"github.com/astaxie/beego"
)

var onlineChannel = make(chan operates.DeviceOnlineStatus, 10)

func init() {
	// 监听设备在线状态，并修改数据库中的状态
	go func() {
		// 处理在线状态事件
		for {
			o := <-onlineChannel
			beego.Info("UpdateOnlineStatus")
			if o.Type == "agent" {
				err := agent.UpdateOnlineStatus(o.OnlineStatus, o.Sn)
				if err != nil {
					beego.Error(err.Error())
				}
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
	dev = operates.Device{Id: l.Id, Sn: l.Sn, Name: l.Name, Provider: l.Provider, Agent: l.Agent}
	return dev, nil
}
