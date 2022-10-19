package api

import (
	"encoding/json"
	"go-iot/codec"
	"go-iot/models"
	"go-iot/models/network"
	"go-iot/network/servers"

	"github.com/beego/beego/v2/server/web"
)

// 服务端管理
func init() {
	ns := web.NewNamespace("/api/product",
		// web.NSRouter("/list", &ServerController{}, "post:List"),
		web.NSRouter("/:productId", &ServerController{}, "get:Get"),
		// web.NSRouter("/", &ServerController{}, "post:Add"),
		web.NSRouter("/", &ServerController{}, "put:Update"),
		// web.NSRouter("/:id", &ServerController{}, "delete:Delete"),
		web.NSRouter("/start-server/:productId", &ServerController{}, "put:Start"),
		web.NSRouter("/meters/:productId", &ServerController{}, "get:Meters"),
	)
	web.AddNamespace(ns)
}

type ServerController struct {
	web.Controller
}

// func (c *ServerController) List() {
// 	var ob models.PageQuery
// 	json.Unmarshal(c.Ctx.Input.RequestBody, &ob)

// 	defer c.ServeJSON()

// 	res, err := network.ListNetwork(&ob)
// 	if err != nil {
// 		c.Data["json"] = models.JsonResp{Success: false, Msg: err.Error()}
// 	} else {
// 		c.Data["json"] = &res
// 	}
// }

func (c *ServerController) Get() {
	var resp models.JsonResp
	resp.Success = true
	id := c.Ctx.Input.Param(":productId")

	defer c.ServeJSON()

	nw, err := network.GetByProductId(id)
	if err != nil {
		resp.Msg = err.Error()
		resp.Success = false
	} else {
		resp.Data = nw
	}
	c.Data["json"] = &resp
}

// func (c *ServerController) Add() {
// 	var resp models.JsonResp
// 	resp.Success = true
// 	var ob models.Network

// 	defer c.ServeJSON()

// 	json.Unmarshal(c.Ctx.Input.RequestBody, &ob)
// 	if ob.Port <= 1024 || ob.Port > 65535 {
// 		resp.Msg = "Invalid port number"
// 		resp.Success = false
// 	} else {
// 		err := network.AddNetWork(&ob)
// 		if err != nil {
// 			resp.Msg = err.Error()
// 			resp.Success = false
// 		}
// 	}
// 	c.Data["json"] = &resp
// }

func (c *ServerController) Update() {
	var resp models.JsonResp
	resp.Success = true
	var ob models.Network

	defer c.ServeJSON()

	json.Unmarshal(c.Ctx.Input.RequestBody, &ob)
	if len(ob.ProductId) == 0 {
		resp.Msg = "productId not be empty"
		resp.Success = false
		c.Data["json"] = resp
		return
	}

	_, err := network.GetByProductId(ob.ProductId)
	if err != nil {
		resp.Msg = err.Error()
		resp.Success = false
	} else {
		err = network.UpdateNetwork(&ob)
		if err != nil {
			resp.Msg = err.Error()
			resp.Success = false
		}
	}
	c.Data["json"] = &resp
}

// func (c *ServerController) Delete() {
// 	var ob models.Network
// 	json.Unmarshal(c.Ctx.Input.RequestBody, &ob)
// 	err := network.DeleteNetwork(&ob)
// 	var resp models.JsonResp
// 	if err != nil {
// 		resp.Msg = err.Error()
// 		resp.Success = false
// 	}
// 	c.Data["json"] = &resp
// 	c.ServeJSON()
// }

func (c *ServerController) Start() {
	var resp models.JsonResp
	id := c.Ctx.Input.Param(":productId")
	defer c.ServeJSON()

	nw, err := network.GetByProductId(id)
	resp.Success = true
	if err != nil {
		resp.Msg = err.Error()
		resp.Success = false
	} else {
		if len(nw.Script) == 0 || len(nw.Type) == 0 {
			resp.Msg = "script and type not be empty"
			resp.Success = false
			c.Data["json"] = resp
			return
		}
		config := convertCodecNetwork(nw)
		err = servers.StartServer(config)
		if err != nil {
			resp.Msg = err.Error()
			resp.Success = false
		}
	}
	c.Data["json"] = resp
}

func convertCodecNetwork(nw models.Network) codec.NetworkConf {
	config := codec.NetworkConf{
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

	id := c.Ctx.Input.Param(":productId")
	nw, err := network.GetByProductId(id)
	if err != nil {
		resp.Msg = err.Error()
		resp.Success = false
	} else {
		s := servers.GetServer(nw.ProductId)
		if s != nil {
			m := map[string]interface{}{}
			m["totalConnection"] = s.TotalConnection()
			resp.Data = m
			resp.Success = true
		}
	}

	c.Data["json"] = resp
}
