package north

import (
	"encoding/json"
	"go-iot/models"
	"go-iot/models/network"
	mqttproxy "go-iot/provider/servers/mqtt"

	"github.com/beego/beego/v2/core/logs"
	"github.com/beego/beego/v2/server/web"
)

// 服务端管理
func init() {
	web.Router("/server/list", &ServerController{}, "post:List")
	web.Router("/server/add", &ServerController{}, "post:Add")
	web.Router("/server/update", &ServerController{}, "post:Add")
	web.Router("/server/delete", &ServerController{}, "delete:Delete")
	web.Router("/server/start/?:id", &ServerController{}, "post:Start")
	web.Router("/server/meters/?:id", &ServerController{}, "post:Meters")
}

type ServerController struct {
	web.Controller
}

func (c *ServerController) List() {
	var ob models.PageQuery
	json.Unmarshal(c.Ctx.Input.RequestBody, &ob)

	res, err := network.ListNetwork(&ob)
	if err != nil {
		c.Data["json"] = models.JsonResp{Success: false, Msg: err.Error()}
	} else {
		c.Data["json"] = &res
	}
	c.ServeJSON()
}

func (c *ServerController) Add() {
	var resp models.JsonResp
	resp.Success = true
	var ob models.Network
	json.Unmarshal(c.Ctx.Input.RequestBody, &ob)
	err := network.AddNetWork(&ob)
	if err != nil {
		resp.Msg = err.Error()
		resp.Success = false
	}
	c.Data["json"] = &resp
	c.ServeJSON()
}

func (c *ServerController) Delete() {
	var ob models.Network
	json.Unmarshal(c.Ctx.Input.RequestBody, &ob)
	err := network.DeleteNetwork(&ob)
	var resp models.JsonResp
	if err != nil {
		resp.Msg = err.Error()
		resp.Success = false
	}
	c.Data["json"] = &resp
	c.ServeJSON()
}

var m = map[string]*mqttproxy.Broker{}

func (c *ServerController) Start() {
	id := c.Ctx.Input.Param(":id")
	nw, err := network.GetNetwork(id)
	var resp models.JsonResp
	resp.Success = true
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
	c.Data["json"] = resp
	c.ServeJSON()
}

func (c *ServerController) Meters() {
	id := c.Ctx.Input.Param(":id")
	nw, err := network.GetNetwork(id)
	var resp models.JsonResp
	if err != nil {
		resp.Msg = err.Error()
		resp.Success = false
	} else {
		spec := &network.MQTTProxySpec{}
		spec.FromJson(nw.Configuration)
		broker := m[spec.Name]
		resp.Success = true
		if broker != nil {
			var rest = map[string]int32{}
			rest["TotalConnection"] = broker.TotalConnection()
			rest["TotalWasmVM"] = broker.TotalWasmVM()
			resp.Data = rest
		}
	}

	c.Data["json"] = resp
	c.ServeJSON()
}