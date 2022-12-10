package api

import (
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
		web.NSRouter("/list", &ProductController{}, "get:List"),
		web.NSRouter("/", &ProductController{}, "post:Add"),
		web.NSRouter("/:id", &ProductController{}, "put:Update"),
		web.NSRouter("/:id", &ProductController{}, "get:Get"),
		web.NSRouter("/:id", &ProductController{}, "delete:Delete"),
		web.NSRouter("/:id/deploy", &ProductController{}, "post:Deploy"),
		web.NSRouter("/:id/undeploy", &ProductController{}, "post:Undeploy"),
		web.NSRouter("/:id/modify-tsl", &ProductController{}, "put:ModifyTsl"),
		web.NSRouter("/network/:productId", &ProductController{}, "get:GetNetwork"),
		web.NSRouter("/network", &ProductController{}, "put:UpdateNetwork"),
		web.NSRouter("/network/:productId/run", &ProductController{}, "post:RunNetwork"),
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
	var resp = models.JsonRespOk()
	defer func() {
		ctl.Data["json"] = resp
		ctl.ServeJSON()
	}()
	var ob models.PageQuery
	err := ctl.BindJSON(&ob)
	if err != nil {
		resp = models.JsonRespError(err)
	}
	res, err := product.ListProduct(&ob)
	if err != nil {
		resp = models.JsonRespError(err)
	} else {
		resp = models.JsonRespOkData(res)
	}
}

// 查询型号列表
func (ctl *ProductController) List() {
	if ctl.isForbidden(productResource, QueryAction) {
		return
	}
	var resp = models.JsonRespOk()
	defer func() {
		ctl.Data["json"] = resp
		ctl.ServeJSON()
	}()
	res, err := product.ListAllProduct()
	if err != nil {
		resp = models.JsonRespError(err)
	} else {
		resp = models.JsonRespOkData(res)
	}
}

type productDTO struct {
	models.ProductModel
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
	var ob models.ProductModel
	err := ctl.BindJSON(&ob)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	ob.Metadata = ""
	pro := ob.ToEnitty()
	err = product.UpdateProduct(&pro)
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
	p, err := product.GetProductMust(id)
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
	aligns.ProductModel = *p
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
	ob, err := product.GetProductMust(id)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}

	if len(strings.TrimSpace(ob.Id)) == 0 || len(strings.TrimSpace(ob.Metadata)) == 0 {
		resp = models.JsonRespError(errors.New("id and metadata must present"))
		return
	}
	tsl := tsl.TslData{}
	err = tsl.FromJson(ob.Metadata)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	if len(tsl.Properties) == 0 {
		resp = models.JsonRespError(errors.New("tsl properties is empty"))
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
	product.UpdateProductState(&ob.Product)
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
	ob, err := product.GetProductMust(id)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	ob.State = false
	product.UpdateProductState(&ob.Product)
}

func (ctl *ProductController) ModifyTsl() {
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
	var update models.Product
	update.Id = ob.Id
	update.Metadata = ob.Metadata
	tslData := tsl.NewTslData()
	err = tslData.FromJson(update.Metadata)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	err = product.UpdateProduct(&update)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	resp = models.JsonRespOk()
}

// get product network config
func (ctl *ProductController) GetNetwork() {
	if ctl.isForbidden(productResource, QueryAction) {
		return
	}
	var resp = models.JsonRespOk()
	productId := ctl.Ctx.Input.Param(":productId")

	defer func() {
		ctl.Data["json"] = resp
		ctl.ServeJSON()
	}()

	nw, err := network.GetByProductId(productId)
	if err != nil {
		resp = models.JsonRespError(err)
	} else {
		server := servers.GetServer(productId)
		if server == nil {
			nw.State = models.Stop
		} else {
			nw.State = models.Runing
		}
		resp.Data = nw
	}
}

// update product network
func (ctl *ProductController) UpdateNetwork() {
	if ctl.isForbidden(productResource, SaveAction) {
		return
	}
	var resp = models.JsonRespOk()

	defer func() {
		ctl.Data["json"] = resp
		ctl.ServeJSON()
	}()
	var ob models.Network
	err := ctl.BindJSON(&ob)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	if len(ob.ProductId) == 0 {
		resp = models.JsonRespError(errors.New("productId not be empty"))
		return
	}

	nw, err := network.GetByProductId(ob.ProductId)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	if nw == nil {
		nw, err = network.GetUnuseNetwork()
		if err != nil {
			resp = models.JsonRespError(err)
			return
		}
	}
	ob.Id = nw.Id
	if len(nw.CodecId) == 0 {
		ob.CodecId = codec.CodecIdScriptCode
	}
	err = network.UpdateNetwork(&ob)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
}

// start server
func (ctl *ProductController) RunNetwork() {
	if ctl.isForbidden(productResource, SaveAction) {
		return
	}
	var resp = models.JsonRespOk()
	productId := ctl.Ctx.Input.Param(":productId")
	state := ctl.Ctx.Input.Query("state")
	defer func() {
		ctl.Data["json"] = resp
		ctl.ServeJSON()
	}()

	nw, err := network.GetByProductId(productId)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	if nw == nil {
		resp = models.JsonRespError(errors.New("product not have network, config network first"))
		return
	}
	if codec.IsNetClientType(nw.Type) {
		resp = models.JsonRespError(errors.New("client type net cant run"))
		return
	}
	if len(nw.Type) == 0 {
		resp = models.JsonRespError(errors.New("type of network not be empty"))
		return
	}
	if len(nw.Script) == 0 {
		resp = models.JsonRespError(errors.New("script not be empty"))
		return
	}
	if state == "start" {
		nw.State = models.Runing
		config := convertCodecNetwork(*nw)
		err = servers.StartServer(config)
		if err != nil {
			resp = models.JsonRespError(err)
			return
		}
	} else if state == "stop" {
		nw.State = models.Stop
		err := servers.StopServer(productId)
		if err != nil {
			resp = models.JsonRespError(err)
			return
		}
	} else {
		resp = models.JsonRespError(errors.New("state must be start or stop"))
		return
	}
	network.UpdateNetwork(nw)
}
