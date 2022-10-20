package api

import (
	"encoding/json"
	"go-iot/codec"
	"go-iot/codec/tsl"
	"go-iot/models"
	product "go-iot/models/device"
	"go-iot/models/network"
	"go-iot/network/servers"
	"strings"

	"github.com/beego/beego/v2/server/web"
)

// 产品管理
func init() {
	ns := web.NewNamespace("/api/product",
		web.NSRouter("/list", &ProductController{}, "post:List"),
		web.NSRouter("/", &ProductController{}, "post:Add"),
		web.NSRouter("/", &ProductController{}, "put:Update"),
		web.NSRouter("/:id", &ProductController{}, "delete:Delete"),
		web.NSRouter("/publish-model", &ProductController{}, "put:PublishModel"),
		web.NSRouter("/network/:productId", &ProductController{}, "get:Get"),
		web.NSRouter("/network", &ProductController{}, "put:UpdateNetwork"),
		web.NSRouter("/network-start/:productId", &ProductController{}, "put:StartNetwork"),
	)
	web.AddNamespace(ns)
}

type ProductController struct {
	web.Controller
}

// 查询型号列表
func (ctl *ProductController) List() {
	var ob models.PageQuery
	json.Unmarshal(ctl.Ctx.Input.RequestBody, &ob)

	res, err := product.ListProduct(&ob)
	if err != nil {
		ctl.Data["json"] = models.JsonResp{Success: false, Msg: err.Error()}
	} else {

		ctl.Data["json"] = &res
	}
	ctl.ServeJSON()
}

// 添加型号
func (ctl *ProductController) Add() {
	var resp models.JsonResp
	defer func() {
		ctl.Data["json"] = resp
		ctl.ServeJSON()
	}()
	var ob models.Product
	err := json.Unmarshal(ctl.Ctx.Input.RequestBody, &ob)
	if err != nil {
		resp = models.JsonResp{Success: false, Msg: err.Error()}
		return
	}

	err = product.AddProduct(&ob)
	if err != nil {
		resp = models.JsonResp{Success: false, Msg: err.Error()}
		return
	}
	resp = models.JsonResp{Success: true}
}

// 更新型号信息
func (ctl *ProductController) Update() {
	var resp models.JsonResp
	defer func() {
		ctl.Data["json"] = resp
		ctl.ServeJSON()
	}()
	var ob models.Product
	err := json.Unmarshal(ctl.Ctx.Input.RequestBody, &ob)
	if err != nil {
		resp = models.JsonResp{Success: false, Msg: err.Error()}
		return
	}
	err = product.UpdateProduct(&ob)
	if err != nil {
		resp = models.JsonResp{Success: false, Msg: err.Error()}
		return
	}
	resp = models.JsonResp{Success: true}
}

// 删除型号
func (ctl *ProductController) Delete() {
	var resp models.JsonResp
	defer func() {
		ctl.Data["json"] = resp
		ctl.ServeJSON()
	}()

	id := ctl.Ctx.Input.Param(":id")
	var ob *models.Product = &models.Product{
		Id: id,
	}
	// when delete product stop server first
	s := servers.GetServer(id)
	if s != nil {
		err := s.Stop()
		if err != nil {
			resp = models.JsonResp{Success: false, Msg: err.Error()}
			return
		}
	}
	// then delete product
	err := product.DeleteProduct(ob)
	if err != nil {
		resp = models.JsonResp{Success: false, Msg: err.Error()}
		return
	}
	resp = models.JsonResp{Success: true}
}

// publish tsl model
func (ctl *ProductController) PublishModel() {
	var resp models.JsonResp
	defer func() {
		ctl.Data["json"] = resp
		ctl.ServeJSON()
	}()

	var ob models.Product
	err := json.Unmarshal(ctl.Ctx.Input.RequestBody, &ob)
	if err != nil {
		resp = models.JsonResp{Success: false, Msg: err.Error()}
		return
	}
	if len(strings.TrimSpace(ob.Id)) == 0 || len(strings.TrimSpace(ob.MetaData)) == 0 {
		resp = models.JsonResp{Success: false, Msg: "id and metaData must present"}
		return
	}
	tsl := tsl.TslData{}
	err = tsl.FromJson(ob.MetaData)
	if err != nil {
		resp = models.JsonResp{Success: false, Msg: err.Error()}
		return
	}
	product := codec.GetProductManager().Get(ob.Id)
	if product == nil {
		product = codec.NewProduct(ob.Id, make(map[string]string), codec.TIME_SERISE_ES)
		codec.GetProductManager().Put(product)
	}
	err = product.GetTimeSeries().PublishModel(product, tsl)
	if err != nil {
		resp = models.JsonResp{Success: false, Msg: err.Error()}
		return
	}
	resp = models.JsonResp{Success: true}
}

// get product network config
func (c *ProductController) GetNetwork() {
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

// update product network
func (c *ProductController) UpdateNetwork() {
	var resp models.JsonResp
	var ob models.Network

	defer func() {
		c.Data["json"] = &resp
		c.ServeJSON()
	}()
	resp.Success = false
	json.Unmarshal(c.Ctx.Input.RequestBody, &ob)
	if len(ob.ProductId) == 0 {
		resp.Msg = "productId not be empty"
		return
	}

	nw, err := network.GetByProductId(ob.ProductId)
	if err != nil {
		resp.Msg = err.Error()
		return
	}
	if nw == nil {
		nw, err = network.GetUnuseNetwork()
		if err != nil {
			resp.Msg = err.Error()
			return
		}
		ob.Id = nw.Id
	}
	if len(nw.Script) == 0 || len(nw.Type) == 0 {
		resp.Msg = "script and type not be empty"
		return
	}
	err = network.UpdateNetwork(&ob)
	if err != nil {
		resp.Msg = err.Error()
		return
	}
	resp.Success = true
}

// start server
func (c *ProductController) StartNetwork() {
	var resp models.JsonResp
	id := c.Ctx.Input.Param(":productId")
	defer c.ServeJSON()

	nw, err := network.GetByProductId(id)
	if err != nil {
		resp.Msg = err.Error()
		resp.Success = false
		c.Data["json"] = resp
		return
	}
	if nw == nil {
		resp.Msg = "product not have network, config network first"
		resp.Success = false
		c.Data["json"] = resp
		return
	}
	resp.Success = true
	if len(nw.Script) == 0 || len(nw.Type) == 0 {
		resp.Msg = "script and type not be empty"
		resp.Success = false
		c.Data["json"] = resp
		return
	}
	config := convertCodecNetwork(*nw)
	err = servers.StartServer(config)
	if err != nil {
		resp.Msg = err.Error()
		resp.Success = false
	}
	c.Data["json"] = resp
}
