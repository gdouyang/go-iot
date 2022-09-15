package north

import (
	"encoding/json"
	"go-iot/models"
	"go-iot/models/network"
	httpserver "go-iot/provider/servers/http"
	mqttproxy "go-iot/provider/servers/mqtt"
	tcpserver "go-iot/provider/servers/tcp"
	websocketserver "go-iot/provider/servers/websocket"

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

func (c *ServerController) Start() {
	id := c.Ctx.Input.Param(":id")
	nw, err := network.GetNetwork(id)
	var resp models.JsonResp
	resp.Success = true
	if err != nil {
		resp.Msg = err.Error()
		resp.Success = false
	} else {
		switch nw.Type {
		case models.MQTT_BROKER:
			success := mqttproxy.ServerStart(nw.Configuration, nw.Script)
			if success {
				resp.Msg = "broker start success"
			} else {
				resp.Msg = "broker start failed"
				resp.Success = false
			}
		case models.TCP_SERVER:
			tcpserver.ServerSocket(nw)
		case models.HTTP_SERVER:
			httpserver.ServerStart()
		case models.WEBSOCKET_SERVER:
			websocketserver.ServerStart()
		default:
			logs.Error("unknow type %s", nw.Type)
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
		switch nw.Type {
		case models.MQTT_BROKER:
			rest := mqttproxy.Meters(nw.Configuration)
			resp.Success = true
			if rest != nil {
				resp.Data = rest
			}
		case models.TCP_SERVER:
		case models.HTTP_SERVER:
		case models.WEBSOCKET_SERVER:
		default:
			logs.Error("unknow type %s", nw.Type)
		}
	}

	c.Data["json"] = resp
	c.ServeJSON()
}
