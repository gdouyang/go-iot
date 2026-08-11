package app

import (
	"encoding/json"

	"go-iot/pkg/api"
	"go-iot/pkg/bootstrap"
	"go-iot/pkg/cluster"
	"go-iot/pkg/core"
	"go-iot/pkg/logger"
	"go-iot/pkg/models"
	"go-iot/pkg/models/base"
	modelDevice "go-iot/pkg/models/device"
	modelNetWork "go-iot/pkg/models/network"
	"go-iot/pkg/models/notify"
	"go-iot/pkg/models/rule"
	"go-iot/pkg/network"
	"go-iot/pkg/network/servers"
	notify1 "go-iot/pkg/notify"
	"go-iot/pkg/ruleengine"
)

// restoreRuntime 恢复运行态（原 pkg/api/a_init boot listener）。
// 菜单资源同步执行；网络/规则/通知/client 异步，避免拖死 Start。
// 须在 models.RegisterModels 之后、且 blank import registry 已注册 api.Resources 之后调用。
func (a *App) restoreRuntime() {
	logger.Infof("app restore: begin")
	a.initMenuResources()
	go a.startRunningNetServer()
	go a.startRunningGoiotMqttCodecs()
	go a.startRunningRule()
	go a.startRunningNotify()
	go a.startRunningNetClient()
	logger.Infof("app restore: scheduled (menu sync; network/rule/notify/client async)")
}

func (a *App) initMenuResources() {
	for _, r := range api.Resources {
		var m models.MenuResource
		m.Code = r.Id
		m.Name = r.Name
		m.Sort = r.Sort
		ac, err := json.Marshal(r.Action)
		if err != nil {
			logger.Errorf("init resources error: %v", err)
		}
		m.Action = string(ac)
		old, err := base.GetMenuResourceByCode(m.Code)
		if err != nil {
			logger.Errorf("get menu resource by code error: %v", err)
			continue
		}
		if old != nil {
			m.Id = old.Id
			if err := base.UpdateMenuResource(&m); err != nil {
				logger.Errorf("update menu resource %s error: %v", m.Code, err)
			}
		} else {
			if err := base.AddMenuResource(&m); err != nil {
				logger.Errorf("add menu resource %s error: %v", m.Code, err)
			}
		}
	}
	logger.Infof("app restore: menu resource inited (%d items)", len(api.Resources))
}

func (a *App) startRunningRule() {
	logger.Infof("app restore: start running rule")
	var page models.PageQuery
	page.PageSize = 300
	page.Condition = []core.SearchTerm{{Key: "state", Value: models.Started}}
	for {
		result, err := rule.PageRule(&page, nil)
		if err != nil {
			logger.Errorf("page rule error: %v", err)
			return
		}
		page.SearchAfter = result.SearchAfter
		list := result.List
		if len(list) == 0 {
			break
		}
		for _, r := range list {
			m, err := rule.GetRuleMust(r.Id)
			if err != nil {
				logger.Errorf("get rule error: %v", err)
				continue
			}
			exec := bootstrap.RuleModelToRuleExecutor(m)
			if err = ruleengine.Start(m.Id, &exec); err != nil {
				logger.Errorf("start rule error: %v", err)
			}
		}
	}
	logger.Infof("app restore: start running rule done")
}

func (a *App) startRunningNotify() {
	logger.Infof("app restore: start running notify")
	var page models.PageQuery
	page.PageSize = 300
	page.Condition = []core.SearchTerm{{Key: "state", Value: models.Started}}
	for {
		result, err := notify.PageNotify(&page, nil)
		if err != nil {
			logger.Errorf("start notify error: %v", err)
			return
		}
		page.SearchAfter = result.SearchAfter
		list := result.List
		if len(list) == 0 {
			break
		}
		for _, m := range list {
			config := notify1.NotifyConfig{Config: m.Config, Template: m.Template}
			if err := notify1.EnableNotify(m.Type, m.Id, config); err != nil {
				logger.Errorf("start notify error: %v", err)
			}
		}
	}
	logger.Infof("app restore: start running notify done")
}

func (a *App) startRunningNetServer() {
	logger.Infof("app restore: start running network")
	var page models.PageQuery
	page.PageSize = 300
	page.Condition = []core.SearchTerm{{Key: "state", Value: models.Runing}}
	for {
		result, err := modelNetWork.PageNetwork(&page)
		if err != nil {
			logger.Errorf("start network error: %v", err)
			return
		}
		page.SearchAfter = result.SearchAfter
		list := result.List
		if len(list) == 0 {
			break
		}
		for _, nw := range list {
			config, err := bootstrap.ConvertCodecNetwork(nw)
			if err != nil {
				logger.Errorf("start network error: %v", err)
				continue
			}
			if err = servers.StartServer(config); err != nil {
				logger.Errorf("start network error: %v", err)
			}
		}
	}
	logger.Infof("app restore: start running network done")
}

// startRunningGoiotMqttCodecs 为使用平台内置 MQTT Broker 的产品加载编解码（不启 broker 进程）。
func (a *App) startRunningGoiotMqttCodecs() {
	logger.Infof("app restore: start GOIOT_MQTT_BROKER codecs")
	var page models.PageQuery
	page.PageSize = 300
	page.Condition = []core.SearchTerm{{Key: "networkType", Value: string(network.GOIOT_MQTT_BROKER)}}
	for {
		result, err := modelDevice.PageProductAll(&page)
		if err != nil {
			logger.Errorf("start GOIOT_MQTT_BROKER error: %v", err)
			return
		}
		page.SearchAfter = result.SearchAfter
		list := result.List
		if len(list) == 0 {
			break
		}
		for _, p := range list {
			if _, err := core.NewCodec(p.CodecId, p.Id, p.Script); err != nil {
				logger.Errorf("start GOIOT_MQTT_BROKER codec error: %v", err)
			}
		}
	}
	logger.Infof("app restore: start GOIOT_MQTT_BROKER codecs done")
}

func (a *App) startRunningNetClient() {
	logger.Infof("app restore: start running netclient")
	var page models.PageQuery
	page.PageSize = 300
	page.Condition = []core.SearchTerm{{Key: "port", Value: 0}, {Key: "productId", Value: "", Oper: core.NEQ}}
	for {
		result, err := modelNetWork.PageNetwork(&page)
		if err != nil {
			logger.Errorf("start netclient error: %v", err)
			return
		}
		page.SearchAfter = result.SearchAfter
		list := result.List
		if len(list) == 0 {
			break
		}
		for _, nw := range list {
			if len(nw.Configuration) == 0 {
				continue
			}
			var devicePage models.PageQuery
			devicePage.PageSize = 300
			devicePage.Condition = []core.SearchTerm{
				{Key: "State", Value: core.OFFLINE},
				{Key: "productId", Value: nw.ProductId},
			}
			for {
				r1, err := modelDevice.PageDevice(&devicePage, nil)
				if err != nil {
					logger.Errorf("start netclient error: %v", err)
					break
				}
				devicePage.SearchAfter = r1.SearchAfter
				devices := r1.List
				if len(devices) == 0 {
					break
				}
				for _, dev := range devices {
					if cluster.Enabled() && !cluster.Shard(dev.Id) {
						continue
					}
					if err := bootstrap.ConnectClientDevice(dev.Id); err != nil {
						logger.Errorf("start netclient error: %v", err)
					}
				}
			}
		}
	}
	logger.Infof("app restore: start running netclient done")
}
