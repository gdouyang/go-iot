package api

import (
	"encoding/json"
	"go-iot/codec"
	"go-iot/codec/tsl"
	"go-iot/models"
	product "go-iot/models/device"
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
	product := codec.GetProductManager().Get(ob.Id)
	if product == nil {
		product = codec.NewProduct(ob.Id, make(map[string]string), codec.TIME_SERISE_ES)
		codec.GetProductManager().Put(product)
	}
	tsl := tsl.TslData{}
	err = tsl.FromJson(ob.MetaData)
	if err != nil {
		resp = models.JsonResp{Success: false, Msg: err.Error()}
		return
	}
	err = product.GetTimeSeries().PublishModel(product, tsl)
	if err != nil {
		resp = models.JsonResp{Success: false, Msg: err.Error()}
		return
	}
	resp = models.JsonResp{Success: true}
}
