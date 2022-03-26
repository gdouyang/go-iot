package north

import (
	"encoding/json"
	"go-iot/models"
	"go-iot/models/network"
	mqttproxy "go-iot/provider/servers/mqtt"

	"github.com/beego/beego/v2/core/logs"
	"github.com/beego/beego/v2/server/web"
)

// 服务器管理
func init() {
	web.Router("/server/list", &ServerController{}, "post:List")
	web.Router("/server/add", &ServerController{}, "post:Add")
	web.Router("/server/update", &ServerController{}, "post:Add")
	web.Router("/server/delete", &ServerController{}, "delete:Delete")
	web.Router("/server/start/?:id", &ServerController{}, "post:Start")
}

type ServerController struct {
	web.Controller
}

func (this *ServerController) List() {
	var ob models.PageQuery
	json.Unmarshal(this.Ctx.Input.RequestBody, &ob)

	res, err := network.ListNetwork(&ob)
	if err != nil {
		this.Data["json"] = models.JsonResp{Success: false, Msg: err.Error()}
	} else {
		this.Data["json"] = &res
	}
	this.ServeJSON()
}

func (this *ServerController) Add() {
	var resp models.JsonResp
	resp.Success = true
	var ob models.Network
	json.Unmarshal(this.Ctx.Input.RequestBody, &ob)
	err := network.AddNetWork(&ob)
	if err != nil {
		resp.Msg = err.Error()
		resp.Success = false
	}
	this.Data["json"] = &resp
	this.ServeJSON()
}

func (this *ServerController) Delete() {
	var ob models.Network
	json.Unmarshal(this.Ctx.Input.RequestBody, &ob)
	err := network.DeleteNetwork(&ob)
	var resp models.JsonResp
	if err != nil {
		resp.Msg = err.Error()
		resp.Success = false
	}
	this.Data["json"] = &resp
	this.ServeJSON()
}

var m = map[string]*mqttproxy.Broker{}

func (this *ServerController) Start() {
	id := this.Ctx.Input.Param(":id")
	nw, err := network.GetNetwork(id)
	var resp models.JsonResp
	if err != nil {
		resp.Msg = err.Error()
		resp.Success = false
	} else {
		if nw.Type == models.MQTT_BROKER {
			spec := &network.MQTTProxySpec{}
			spec.FromJson(nw.Configuration)
			broker := mqttproxy.NewBroker(spec, nw.Script)
			if broker == nil {
				logs.Error("broker %v start failed", spec.Name)
				resp.Msg = "broker start failed"
				resp.Success = false
			} else {
				resp.Msg = "broker start success"
				m[spec.Name] = broker
			}
		}
	}
	this.Data["json"] = resp
	this.ServeJSON()
}
