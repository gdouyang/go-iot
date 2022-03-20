package north

import (
	"encoding/json"
	"go-iot/models"
	mqttproxy "go-iot/provider/servers/mqtt"

	"github.com/beego/beego/v2/core/logs"
	"github.com/beego/beego/v2/server/web"
)

// 服务器管理
func init() {
	web.Router("/server/list", &MaterialController{}, "post:List")
	web.Router("/server/add", &MaterialController{}, "post:Add")
	web.Router("/server/update", &MaterialController{}, "post:Add")
	web.Router("/server/delete", &MaterialController{}, "delete:Delete")
	web.Router("/server/start", &MaterialController{}, "post:Start")
}

type MaterialController struct {
	web.Controller
}

func (this *MaterialController) List() {
	var ob models.PageQuery
	json.Unmarshal(this.Ctx.Input.RequestBody, &ob)

	this.Data["json"] = models.JsonResp{}
	this.ServeJSON()
}

func (this *MaterialController) Add() {
	var resp models.JsonResp
	resp.Success = true
	defer func() {
		this.Data["json"] = &resp
		this.ServeJSON()
	}()
	resp.Msg = ""
	resp.Success = false
}

func (this *MaterialController) Delete() {
	this.Data["json"] = &models.JsonResp{}
	this.ServeJSON()
}

var m = map[string]*mqttproxy.Broker{}

func (this *MaterialController) Start() {
	spec := &mqttproxy.Spec{
		Name:   "mqttproxy",
		Port:   1883,
		UseTLS: false,
	}
	broker := mqttproxy.NewBroker(spec)
	if broker == nil {
		logs.Error("broker %v start failed", spec.Name)
	}
	m[spec.Name] = broker
	this.Data["json"] = &models.JsonResp{}
	this.ServeJSON()
}
