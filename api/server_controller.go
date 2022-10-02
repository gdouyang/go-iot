package api

import (
	"encoding/json"
	"go-iot/codec"
	"go-iot/models"
	"go-iot/models/network"
	httpserver "go-iot/network/servers/http"
	mqttserver "go-iot/network/servers/mqtt"
	tcpserver "go-iot/network/servers/tcp"
	websocketserver "go-iot/network/servers/websocket"
	"strconv"

	"github.com/beego/beego/v2/core/logs"
	"github.com/beego/beego/v2/server/web"
)

// 服务端管理
func init() {
	ns := web.NewNamespace("/api/server",
		web.NSRouter("/list", &ServerController{}, "post:List"),
		web.NSRouter("/", &ServerController{}, "post:Add"),
		web.NSRouter("/", &ServerController{}, "put:Update"),
		web.NSRouter("/:id", &ServerController{}, "delete:Delete"),
		web.NSRouter("/start/:id", &ServerController{}, "get:Start"),
		web.NSRouter("/meters/:id", &ServerController{}, "get:Meters"),
	)
	web.AddNamespace(ns)
}

type ServerController struct {
	web.Controller
}

func (c *ServerController) List() {
	var ob models.PageQuery
	json.Unmarshal(c.Ctx.Input.RequestBody, &ob)

	defer c.ServeJSON()

	res, err := network.ListNetwork(&ob)
	if err != nil {
		c.Data["json"] = models.JsonResp{Success: false, Msg: err.Error()}
	} else {
		c.Data["json"] = &res
	}
}

func (c *ServerController) Add() {
	var resp models.JsonResp
	resp.Success = true
	var ob models.Network

	defer c.ServeJSON()

	json.Unmarshal(c.Ctx.Input.RequestBody, &ob)
	if ob.Port <= 1024 || ob.Port > 65535 {
		resp.Msg = "Invalid port number"
		resp.Success = false
	} else {
		err := network.AddNetWork(&ob)
		if err != nil {
			resp.Msg = err.Error()
			resp.Success = false
		}
	}
	c.Data["json"] = &resp
}

func (c *ServerController) Update() {
	var resp models.JsonResp
	resp.Success = true
	var ob models.Network

	defer c.ServeJSON()

	json.Unmarshal(c.Ctx.Input.RequestBody, &ob)
	if ob.Port <= 1024 || ob.Port > 65535 {
		resp.Msg = "Invalid port number"
		resp.Success = false
	} else {
		err := network.AddNetWork(&ob)
		if err != nil {
			resp.Msg = err.Error()
			resp.Success = false
		}
	}
	c.Data["json"] = &resp
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
	var resp models.JsonResp
	id := c.Ctx.Input.Param(":id")
	defer c.ServeJSON()

	_id, err := strconv.Atoi(id)
	if err != nil {
		resp.Msg = "broker start failed"
		resp.Success = false
		return
	}
	nw, err := network.GetNetwork(int64(_id))
	resp.Success = true
	if err != nil {
		resp.Msg = err.Error()
		resp.Success = false
	} else {
		switch nw.Type {
		case models.MQTT_BROKER:
			config := convertCodecNetwork(nw)
			success := mqttserver.ServerStart(config)
			if success {
				resp.Msg = "broker start success"
			} else {
				resp.Msg = "broker start failed"
				resp.Success = false
			}
		case models.TCP_SERVER:
			config := convertCodecNetwork(nw)
			tcpserver.ServerSocket(config)
		case models.HTTP_SERVER:
			httpserver.ServerStart()
		case models.WEBSOCKET_SERVER:
			config := convertCodecNetwork(nw)
			websocketserver.ServerStart(config)
		default:
			logs.Error("unknow type %s", nw.Type)
		}
	}
	c.Data["json"] = resp
}

func convertCodecNetwork(nw models.Network) codec.Network {
	config := codec.Network{
		Name:          nw.Name,
		Port:          nw.Port,
		ProductId:     nw.ProductId,
		Configuration: nw.Configuration,
		Script:        nw.Script,
		Type:          nw.Type,
		CodecId:       nw.CodecId,
	}
	return config
}

func (c *ServerController) Meters() {
	var resp models.JsonResp
	defer c.ServeJSON()

	id := c.Ctx.Input.Param(":id")
	_id, err := strconv.Atoi(id)
	if err != nil {
		resp.Msg = "broker start failed"
		resp.Success = false
		return
	}
	nw, err := network.GetNetwork(int64(_id))
	if err != nil {
		resp.Msg = err.Error()
		resp.Success = false
	} else {
		switch nw.Type {
		case models.MQTT_BROKER:
			rest := mqttserver.Meters(nw.Configuration)
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
}
