package api

import (
	"encoding/json"
	"errors"
	"go-iot/codec"
	"go-iot/codec/tsl"
	"go-iot/models"
	product "go-iot/models/device"
	"go-iot/models/network"
	"go-iot/network/servers"
	"strings"

	"github.com/beego/beego/v2/server/web"
)

var productResource = Resource{
	Id:   "product-mgr",
	Name: "产品",
	Action: []ResourceAction{
		QueryAction,
		CretaeAction,
		SaveAction,
		DeleteAction,
	},
}

// 产品管理
func init() {
	ns := web.NewNamespace("/api/product",
		web.NSRouter("/page", &ProductController{}, "post:Page"),
		web.NSRouter("/list", &ProductController{}, "post:List"),
		web.NSRouter("/", &ProductController{}, "post:Add"),
		web.NSRouter("/:id", &ProductController{}, "put:Update"),
		web.NSRouter("/:id", &ProductController{}, "get:Get"),
		web.NSRouter("/:id", &ProductController{}, "delete:Delete"),
		web.NSRouter("/:id/deploy", &ProductController{}, "put:Deploy"),
		web.NSRouter("/:id/undeploy", &ProductController{}, "put:Undeploy"),
		web.NSRouter("/network/:productId", &ProductController{}, "get:GetNetwork"),
		web.NSRouter("/network", &ProductController{}, "put:UpdateNetwork"),
		web.NSRouter("/network-start/:productId", &ProductController{}, "put:StartNetwork"),
	)
	web.AddNamespace(ns)

	regResource(productResource)
}

type ProductController struct {
	AuthController
}

func (ctl *ProductController) Page() {
	if ctl.isForbidden(productResource, QueryAction) {
		return
	}
	var ob models.PageQuery
	ctl.BindJSON(&ob)

	res, err := product.ListProduct(&ob)
	if err != nil {
		ctl.Data["json"] = models.JsonRespError(err)
	} else {

		ctl.Data["json"] = models.JsonRespOkData(res)
	}
	ctl.ServeJSON()
}

// 查询型号列表
func (ctl *ProductController) List() {
	if ctl.isForbidden(productResource, QueryAction) {
		return
	}

	res, err := product.ListAllProduct()
	if err != nil {
		ctl.Data["json"] = models.JsonRespError(err)
	} else {

		ctl.Data["json"] = models.JsonRespOkData(res)
	}
	ctl.ServeJSON()
}

type productDTO struct {
	models.Product
	NetworkType string `json:"networkType"`
}

// 添加型号
func (ctl *ProductController) Add() {
	if ctl.isForbidden(productResource, CretaeAction) {
		return
	}
	var resp models.JsonResp
	defer func() {
		ctl.Data["json"] = resp
		ctl.ServeJSON()
	}()
	var aligns productDTO
	err := ctl.BindJSON(&aligns)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	if len(aligns.NetworkType) == 0 {
		resp = models.JsonRespError(errors.New("networkType not empty"))
		return
	}
	var ob = aligns.Product
	if len(ob.StorePolicy) == 0 {
		ob.StorePolicy = codec.TIME_SERISE_ES
	}

	err = product.AddProduct(&ob, aligns.NetworkType)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	resp = models.JsonRespOk()
}

// 更新型号信息
func (ctl *ProductController) Update() {
	if ctl.isForbidden(productResource, SaveAction) {
		return
	}
	var resp models.JsonResp
	defer func() {
		ctl.Data["json"] = resp
		ctl.ServeJSON()
	}()
	var ob models.Product
	err := ctl.BindJSON(&ob)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	err = product.UpdateProduct(&ob)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	resp = models.JsonRespOk()
}

func (ctl *ProductController) Get() {
	if ctl.isForbidden(productResource, QueryAction) {
		return
	}
	var resp models.JsonResp
	defer func() {
		ctl.Data["json"] = resp
		ctl.ServeJSON()
	}()

	id := ctl.Ctx.Input.Param(":id")
	p, err := product.GetProduct(id)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	nw, err := network.GetByProductId(id)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	var aligns productDTO
	aligns.Product = *p
	if nw != nil {
		aligns.NetworkType = nw.Type
	}
	resp = models.JsonRespOkData(aligns)
}

// 删除型号
func (ctl *ProductController) Delete() {
	if ctl.isForbidden(productResource, DeleteAction) {
		return
	}
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
			resp = models.JsonRespError(err)
			return
		}
	}
	// then delete product
	err := product.DeleteProduct(ob)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	resp = models.JsonRespOk()
}

// publish tsl model
func (ctl *ProductController) Deploy() {
	if ctl.isForbidden(productResource, SaveAction) {
		return
	}
	var resp = models.JsonRespOk()
	defer func() {
		ctl.Data["json"] = resp
		ctl.ServeJSON()
	}()

	id := ctl.Ctx.Input.Param(":id")
	ob, err := product.GetProduct(id)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	if ob == nil {
		resp = models.JsonRespError(errors.New("not exist"))
		return
	}

	if len(strings.TrimSpace(ob.Id)) == 0 || len(strings.TrimSpace(ob.Metadata)) == 0 {
		resp = models.JsonResp{Success: false, Msg: "id and metaData must present"}
		return
	}
	tsl := tsl.TslData{}
	err = tsl.FromJson(ob.Metadata)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	p1 := codec.GetProductManager().Get(ob.Id)
	if p1 == nil {
		p1 = codec.NewProduct(ob.Id, make(map[string]string), codec.TIME_SERISE_ES)
		codec.GetProductManager().Put(p1)
	}
	err = p1.GetTimeSeries().PublishModel(p1, tsl)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	ob.State = true
	product.UpdateProductState(ob)
}

func (ctl *ProductController) Undeploy() {
	if ctl.isForbidden(productResource, SaveAction) {
		return
	}
	var resp = models.JsonRespOk()
	defer func() {
		ctl.Data["json"] = resp
		ctl.ServeJSON()
	}()
	id := ctl.Ctx.Input.Param(":id")
	ob, err := product.GetProduct(id)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	if ob == nil {
		resp = models.JsonRespError(errors.New("not exist"))
		return
	}
	ob.State = false
	product.UpdateProductState(ob)
}

// get product network config
func (ctl *ProductController) GetNetwork() {
	if ctl.isForbidden(productResource, QueryAction) {
		return
	}
	var resp models.JsonResp
	resp.Success = true
	id := ctl.Ctx.Input.Param(":productId")

	defer ctl.ServeJSON()

	nw, err := network.GetByProductId(id)
	if err != nil {
		resp.Msg = err.Error()
		resp.Success = false
	} else {
		resp.Data = nw
	}
	ctl.Data["json"] = &resp
}

// update product network
func (ctl *ProductController) UpdateNetwork() {
	if ctl.isForbidden(productResource, SaveAction) {
		return
	}
	var resp models.JsonResp
	var ob models.Network

	defer func() {
		ctl.Data["json"] = &resp
		ctl.ServeJSON()
	}()
	resp.Success = false
	json.Unmarshal(ctl.Ctx.Input.RequestBody, &ob)
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
	err = network.UpdateNetwork(&ob)
	if err != nil {
		resp.Msg = err.Error()
		return
	}
	resp.Success = true
}

// start server
func (ctl *ProductController) StartNetwork() {
	if ctl.isForbidden(productResource, SaveAction) {
		return
	}
	var resp models.JsonResp
	id := ctl.Ctx.Input.Param(":productId")
	defer ctl.ServeJSON()

	nw, err := network.GetByProductId(id)
	if err != nil {
		resp.Msg = err.Error()
		resp.Success = false
		ctl.Data["json"] = resp
		return
	}
	if nw == nil {
		resp.Msg = "product not have network, config network first"
		resp.Success = false
		ctl.Data["json"] = resp
		return
	}
	resp.Success = true
	if len(nw.Script) == 0 || len(nw.Type) == 0 {
		resp.Msg = "script and type not be empty"
		resp.Success = false
		ctl.Data["json"] = resp
		return
	}
	config := convertCodecNetwork(*nw)
	err = servers.StartServer(config)
	if err != nil {
		resp.Msg = err.Error()
		resp.Success = false
	}
	ctl.Data["json"] = resp
}
