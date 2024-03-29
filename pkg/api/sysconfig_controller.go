package api

import (
	"encoding/json"
	"go-iot/pkg/api/web"
	"go-iot/pkg/models"
	base "go-iot/pkg/models/base"
	"go-iot/pkg/option"
	"net/http"

	logs "go-iot/pkg/logger"
)

// 系统配置
func init() {
	var sysConfigResource = Resource{
		Id:   "sys-config",
		Name: "系统配置",
		Action: []ResourceAction{
			QueryAction,
			SaveAction,
		},
	}
	RegResource(sysConfigResource)
	web.RegisterAPI("/anon/system/config", "GET", func(w http.ResponseWriter, r *http.Request) {
		ctl := web.NewController(w, r)
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
	})
	// 保存配置
	web.RegisterAPI("/system/config", "POST", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
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
	})
	// 系统信息
	web.RegisterAPI("/system/info", "GET", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
		m := map[string]string{}
		m["release"] = option.RELEASE
		m["buildTime"] = option.BUILD_TIME
		m["commit"] = option.COMMIT
		m["repo"] = option.REPO
		ctl.RespOkData(m)
	})
	// 更新token过期时间
	web.RegisterAPI("/token/refresh", "GET", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
		ctl.RespOk()
	})
}
