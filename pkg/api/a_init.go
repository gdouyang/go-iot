package api

import (
	"encoding/json"
	"go-iot/pkg/boot"
	"go-iot/pkg/cluster"
	"go-iot/pkg/core"
	"go-iot/pkg/models"
	"go-iot/pkg/models/base"
	device "go-iot/pkg/models/device"
	"go-iot/pkg/models/network"
	"go-iot/pkg/models/notify"
	"go-iot/pkg/models/rule"
	"go-iot/pkg/network/servers"
	notify1 "go-iot/pkg/notify"
	"go-iot/pkg/ruleengine"

	logs "go-iot/pkg/logger"
)

func init() {
	boot.AddStartLinstener(func() {
		start := &start{}
		start.initResources()
		go start.startRuningNetServer()
		go start.startRuningRule()
		go start.startRuningNotify()
		go start.startRuningNetClient()
	})
}

type start struct {
}

func (i *start) initResources() {
	for _, r := range Resources {
		var m models.MenuResource
		m.Code = r.Id
		m.Name = r.Name
		ac, err := json.Marshal(r.Action)
		if err != nil {
			logs.Errorf("init resources error: %v", err)
		}
		m.Action = string(ac)
		old, err := base.GetMenuResourceByCode(m.Code)
		if err != nil {
			logs.Errorf("get menu resource by code error: %v", err)
			continue
		}
		if old != nil {
			m.Id = old.Id
			base.UpdateMenuResource(&m)
		} else {
			base.AddMenuResource(&m)
		}
	}
	logs.Infof("menu resource inited")
}

func (i *start) startRuningRule() {
	logs.Infof("start runing rule")
	var page models.PageQuery
	page.PageSize = 300
	page.Condition = []core.SearchTerm{{Key: "state", Value: models.Started}}
	for {
		result, err := rule.PageRule(&page, nil)
		if err != nil {
			logs.Errorf("page rule error: %v", err)
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
				logs.Errorf("get rule error: %v", err)
				continue
			}
			rule := ruleModelToRuleExecutor(m)
			err = ruleengine.Start(m.Id, &rule)
			if err != nil {
				logs.Errorf("start rule error: %v", err)
				continue
			}
			if err != nil {
				logs.Errorf("start rule error: %v", err)
			}
		}
	}
	logs.Infof("start runing rule done")
}

func (i *start) startRuningNotify() {
	logs.Infof("start runing notify")
	var page models.PageQuery
	page.PageSize = 300
	page.Condition = []core.SearchTerm{{Key: "state", Value: models.Started}}
	for {
		result, err := notify.PageNotify(&page, nil)
		if err != nil {
			logs.Errorf("start notify error: %v", err)
			return
		}
		page.SearchAfter = result.SearchAfter
		list := result.List
		if len(list) == 0 {
			break
		}
		for _, m := range list {
			config := notify1.NotifyConfig{Config: m.Config, Template: m.Template}
			err = notify1.EnableNotify(m.Type, m.Id, config)
			if err != nil {
				logs.Errorf("start notify error: %v", err)
			}
		}
	}
	logs.Infof("start runing notify done")
}

func (i *start) startRuningNetServer() {
	logs.Infof("start runing network")
	var page models.PageQuery
	page.PageSize = 300
	page.Condition = []core.SearchTerm{{Key: "state", Value: models.Runing}}
	for {
		result, err := network.PageNetwork(&page)
		if err != nil {
			logs.Errorf("start network error: %v", err)
			return
		}
		page.SearchAfter = result.SearchAfter
		list := result.List
		if len(list) == 0 {
			break
		}
		for _, nw := range list {
			config, err := convertCodecNetwork(nw)
			if err != nil {
				logs.Errorf("start network error: %v", err)
			}
			err = servers.StartServer(config)
			if err != nil {
				logs.Errorf("start network error: %v", err)
			}
		}
	}
	logs.Infof("start runing network done")
}

func (i *start) startRuningNetClient() {
	logs.Infof("start runing netclient")
	var page models.PageQuery
	page.PageSize = 300
	page.Condition = []core.SearchTerm{{Key: "port", Value: 0}, {Key: "productId", Value: "", Oper: core.NEQ}}
	for {
		result, err := network.PageNetwork(&page)
		if err != nil {
			logs.Errorf("start netclient error: %v", err)
			return
		}
		page.SearchAfter = result.SearchAfter
		list := result.List
		if len(list) == 0 {
			break
		}
		for _, nw := range list {
			if len(nw.Configuration) > 0 {
				var devicePage models.PageQuery
				devicePage.PageSize = 300
				devicePage.Condition = []core.SearchTerm{{Key: "State", Value: core.OFFLINE}, {Key: "productId", Value: nw.ProductId}}
				r1, err := device.PageDevice(&devicePage, nil)
				if err != nil {
					logs.Errorf("start netclient error: %v", err)
					continue
				}
				devicePage.SearchAfter = r1.SearchAfter
				devices := r1.List
				if len(devices) == 0 {
					break
				}
				for _, dev := range devices {
					if cluster.Enabled() {
						if cluster.Shard(dev.Id) {
							err = connectClientDevice(dev.Id)
						}
					} else {
						err = connectClientDevice(dev.Id)
					}
					if err != nil {
						logs.Errorf("start netclient error: %v", err)
					}
				}
			}
		}
	}
	logs.Infof("start runing netclient done")
}
