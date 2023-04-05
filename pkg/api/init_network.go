package api

import (
	"go-iot/pkg/core/boot"
	"go-iot/pkg/core/cluster"
	"go-iot/pkg/models"
	device "go-iot/pkg/models/device"
	"go-iot/pkg/models/network"
	"go-iot/pkg/models/notify"
	"go-iot/pkg/models/rule"
	"go-iot/pkg/network/servers"
	notify1 "go-iot/pkg/notify"
	"go-iot/pkg/ruleengine"

	"github.com/beego/beego/v2/core/logs"
)

func init() {
	boot.AddStartLinstener(func() {
		go startRuningNetServer()
		go startRuningRule()
		go startRuningNotify()
		go startRuningNetClient()
	})
}

func startRuningRule() {
	logs.Info("start runing rule")
	var ob models.Rule
	ob.State = models.Started
	list, err := rule.ListRule(&ob)
	if err != nil {
		logs.Error(err)
		return
	}
	for _, r := range list {
		m, err := rule.GetRuleMust(r.Id)
		if err != nil {
			logs.Error(err)
			continue
		}
		rule := ruleModelToRuleExecutor(m)
		err = ruleengine.Start(m.Id, &rule)
		if err != nil {
			logs.Error(err)
			continue
		}
		if err != nil {
			logs.Error(err)
		}
	}
}

func startRuningNotify() {
	logs.Info("start runing notify")
	var ob models.Notify
	ob.State = models.Started
	list, err := notify.ListAll(&ob)
	if err != nil {
		logs.Error(err)
		return
	}
	for _, m := range list {
		config := notify1.NotifyConfig{Config: m.Config, Template: m.Template}
		err = notify1.EnableNotify(m.Type, m.Id, config)
		if err != nil {
			logs.Error(err)
		}
	}
}

func startRuningNetServer() {
	logs.Info("start runing network")
	list, err := network.ListStartNetwork()
	if err != nil {
		logs.Error(err)
		return
	}
	for _, nw := range list {
		config, err := convertCodecNetwork(nw)
		if err != nil {
			logs.Error(err)
		}
		err = servers.StartServer(config)
		if err != nil {
			logs.Error(err)
		}
	}
}

func startRuningNetClient() {
	logs.Info("start runing netclient")
	list, err := network.ListStartNetClient()
	if err != nil {
		logs.Error(err)
		return
	}
	for _, nw := range list {
		if len(nw.Configuration) > 0 {
			devices, err := device.ListClientDeviceByProductId(nw.ProductId)
			if err != nil {
				logs.Error(err)
				continue
			}
			for _, devId := range devices {
				if cluster.Enabled() {
					if cluster.Shard(devId) {
						err = connectClientDevice(devId)
					}
				} else {
					err = connectClientDevice(devId)
				}
				if err != nil {
					logs.Error(err)
				}
			}
		}
	}
}
