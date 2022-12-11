package api

import (
	"encoding/json"
	"go-iot/models"
	base "go-iot/models/base"

	"github.com/beego/beego/v2/core/logs"
	"github.com/beego/beego/v2/server/web"
)

// 产品管理
func init() {
	ns := web.NewNamespace("/api/",
		web.NSRouter("/anon/system/config", &AnonSysConfigController{}, "get:Get"),
		web.NSRouter("/system/config", &SysConfigController{}, "post:Update"),
	)
	web.AddNamespace(ns)

	regResource(Resource{
		Id:   "sys-config",
		Name: "系统配置",
		Action: []ResourceAction{
			SaveAction,
		},
	})
}

type AnonSysConfigController struct {
	RespController
}

func (ctl *AnonSysConfigController) Get() {
	c, err := base.GetSysconfig("sysconfig")
	if err != nil {
		ctl.RespError(err)
		return
	}
	res := map[string]interface{}{}
	if c == nil {
		ctl.RespOkData(res)
		return
	}
	if len(c.Config) > 0 {
		err = json.Unmarshal([]byte(c.Config), &res)
		if err != nil {
			logs.Error(err)
		}
		ctl.RespOkData(res)
	}
}

type SysConfigController struct {
	AuthController
}

func (ctl *SysConfigController) Update() {
	var ob = struct {
		Id     string                 `json:"id"`
		Config map[string]interface{} `json:"config"`
	}{}
	err := ctl.BindJSON(&ob)
	if err != nil {
		ctl.RespError(err)
		return
	}
	str, err := json.Marshal(ob.Config)
	if err != nil {
		ctl.RespError(err)
		return
	}
	c := &models.SystemConfig{
		Id:     ob.Id,
		Config: string(str),
	}
	old, err := base.GetSysconfig(c.Id)
	if err != nil {
		ctl.RespError(err)
		return
	}
	if old == nil {
		err = base.AddSysconfig(c)
	} else {
		err = base.UpdateSysconfig(c)
	}
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOk()
}
