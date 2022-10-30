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
	web.Controller
}

func (ctl *AnonSysConfigController) Get() {
	var resp = models.JsonRespOk()
	defer func() {
		ctl.Data["json"] = resp
		ctl.ServeJSON()
	}()
	c, err := base.GetSysconfig("sysconfig")
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	res := map[string]interface{}{}
	if c == nil {
		resp.Data = res
		return
	}
	if len(c.Config) > 0 {
		err = json.Unmarshal([]byte(c.Config), &res)
		if err != nil {
			logs.Error(err)
		}
		resp.Data = res
	}
}

type SysConfigController struct {
	AuthController
}

func (ctl *SysConfigController) Update() {
	var resp models.JsonResp
	defer func() {
		ctl.Data["json"] = resp
		ctl.ServeJSON()
	}()
	var ob = struct {
		Id     string
		Config map[string]interface{}
	}{}
	err := json.Unmarshal(ctl.Ctx.Input.RequestBody, &ob)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	str, err := json.Marshal(ob.Config)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	c := &models.SystemConfig{
		Id:     ob.Id,
		Config: string(str),
	}
	old, err := base.GetSysconfig(c.Id)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	if old == nil {
		err = base.UpdateSysconfig(c)
	} else {
		err = base.AddSysconfig(c)
	}
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	resp = models.JsonRespOk()
}
