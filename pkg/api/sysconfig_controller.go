package api

import (
	"encoding/json"
	"go-iot/pkg/api/web"
	"go-iot/pkg/models"
	base "go-iot/pkg/models/base"

	logs "go-iot/pkg/logger"
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
	web.RegisterAPI("/anon/system/config", "GET", &AnonSysConfigController{}, "Get")
	web.RegisterAPI("/system/config", "POST", &SysConfigController{}, "Update")
	web.RegisterAPI("/system/config", "GET", &SysConfigController{}, "TokenExpire")

	RegResource(sysConfigResource)
}

type AnonSysConfigController struct {
	web.RespController
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
