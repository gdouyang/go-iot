package api

import (
	"encoding/json"
	"go-iot/pkg/models"
	base "go-iot/pkg/models/base"

	logs "go-iot/pkg/logger"

	"github.com/beego/beego/v2/server/web"
)

var sysConfigResource = Resource{
	Id:   "sys-config",
	Name: "系统配置",
	Action: []ResourceAction{
		SaveAction,
	},
}

// 系统配置
func init() {
	ns := web.NewNamespace("/api/",
		web.NSRouter("/anon/system/config", &AnonSysConfigController{}, "get:Get"),
		web.NSRouter("/system/config", &SysConfigController{}, "post:Update"),
		web.NSRouter("/token/refresh", &SysConfigController{}, "get:TokenExpire"),
	)
	web.AddNamespace(ns)

	regResource(sysConfigResource)
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
			logs.Errorf("unmarshal config error: %v", err)
		}
		ctl.RespOkData(res)
	}
}

type SysConfigController struct {
	AuthController
}

func (ctl *SysConfigController) Update() {
	if ctl.isForbidden(sysConfigResource, SaveAction) {
		return
	}
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

func (ctl *SysConfigController) TokenExpire() {
	ctl.RespOk()
}
