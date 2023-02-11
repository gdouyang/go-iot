package api

import (
	"errors"
	"go-iot/pkg/codec"
	"go-iot/pkg/codec/tsl"
	"go-iot/pkg/models"
	product "go-iot/pkg/models/device"
	"go-iot/pkg/models/network"
	"go-iot/pkg/network/servers"
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
		web.NSRouter("/:id/tsl", &ProductController{}, "put:UpdateTsl"),
		web.NSRouter("/:id/script", &ProductController{}, "put:UpdateScript"),
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
	var ob models.PageQuery[models.Product]
	err := ctl.BindJSON(&ob)
	if err != nil {
		ctl.RespError(err)
		return
	}
	res, err := product.PageProduct(&ob, ctl.GetCurrentUser().Id)
	if err != nil {
		ctl.RespError(err)
	} else {
		ctl.RespOkData(res)
	}
}

// 查询型号列表
func (ctl *ProductController) List() {
	if ctl.isForbidden(productResource, QueryAction) {
		return
	}
	res, err := product.ListAllProduct(ctl.GetCurrentUser().Id)
	if err != nil {
		ctl.RespError(err)
	} else {
		ctl.RespOkData(res)
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
	var aligns productDTO
	err := ctl.BindJSON(&aligns)
	if err != nil {
		ctl.RespError(err)
		return
	}
	if len(aligns.NetworkType) == 0 {
		ctl.RespError(errors.New("networkType must be present"))
		return
	}
	var ob = aligns.Product
	if len(ob.StorePolicy) == 0 {
		ob.StorePolicy = codec.TIME_SERISE_ES
	}
	ob.CreateId = ctl.GetCurrentUser().Id
	err = product.AddProduct(&ob, aligns.NetworkType)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOk()
}

// 更新型号信息
func (ctl *ProductController) Update() {
	if ctl.isForbidden(productResource, SaveAction) {
		return
	}
	var ob models.ProductModel
	err := ctl.BindJSON(&ob)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ob.Id = ctl.Param(":id")
	_, err = ctl.getProductAndCheckCreate(ob.Id)
	if err != nil {
		ctl.RespError(err)
		return
	}

	pro := ob.ToEnitty()
	pro.Metadata = ""
	pro.Script = ""
	err = product.UpdateProduct(&pro)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOk()
}

func (ctl *ProductController) Get() {
	if ctl.isForbidden(productResource, QueryAction) {
		return
	}

	id := ctl.Param(":id")
	p, err := ctl.getProductAndCheckCreate(id)
	if err != nil {
		ctl.RespError(err)
		return
	}
	nw, err := network.GetByProductId(id)
	if err != nil {
		ctl.RespError(err)
		return
	}
	var aligns productDTO
	aligns.ProductModel = *p
	if nw != nil {
		aligns.NetworkType = nw.Type
	}
	ctl.RespOkData(aligns)
}

// 删除型号
func (ctl *ProductController) Delete() {
	if ctl.isForbidden(productResource, DeleteAction) {
		return
	}

	productId := ctl.Param(":id")
	ob, err := ctl.getProductAndCheckCreate(productId)
	if err != nil {
		ctl.RespError(err)
		return
	}
	server := servers.GetServer(productId)
	if server != nil {
		ctl.RespError(errors.New("network is runing, please stop first"))
		return
	}
	total, err := product.CountDeviceByProductId(productId)
	if err != nil {
		ctl.RespError(err)
		return
	}
	if total > 0 {
		ctl.RespError(errors.New("product have device, can not delete"))
		return
	}
	productoper, _ := codec.NewProduct(productId, map[string]string{}, ob.StorePolicy, "")
	err = productoper.GetTimeSeries().Del(productoper)
	if err != nil {
		ctl.RespError(err)
		return
	}
	// delete product
	err = product.DeleteProduct(&models.Product{Id: productId})
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOk()
}

// publish tsl model
func (ctl *ProductController) Deploy() {
	if ctl.isForbidden(productResource, SaveAction) {
		return
	}
	id := ctl.Param(":id")
	ob, err := ctl.getProductAndCheckCreate(id)
	if err != nil {
		ctl.RespError(err)
		return
	}
	if len(strings.TrimSpace(ob.Metadata)) == 0 {
		ctl.RespError(errors.New("product have no tsl, config it first"))
		return
	}
	tsl := tsl.TslData{}
	err = tsl.FromJson(ob.Metadata)
	if err != nil {
		ctl.RespError(err)
		return
	}
	if len(tsl.Properties) == 0 {
		ctl.RespError(errors.New("tsl properties must be persent"))
		return
	}
	config := map[string]string{}
	for _, v := range ob.Metaconfig {
		config[v.Property] = v.Value
	}
	p1, err := codec.NewProduct(ob.Id, config, ob.StorePolicy, ob.Metadata)
	if err != nil {
		ctl.RespError(err)
		return
	}
	codec.PutProduct(p1)
	err = p1.GetTimeSeries().PublishModel(p1, tsl)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ob.State = true
	product.UpdateProductState(&ob.Product)
	ctl.RespOk()
}

func (ctl *ProductController) Undeploy() {
	if ctl.isForbidden(productResource, SaveAction) {
		return
	}
	productId := ctl.Param(":id")
	ob, err := ctl.getProductAndCheckCreate(productId)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ob.State = false
	product.UpdateProductState(&ob.Product)
	codec.DeleteProduct(productId)
	ctl.RespOk()
}

func (ctl *ProductController) UpdateTsl() {
	if ctl.isForbidden(productResource, SaveAction) {
		return
	}
	var ob models.Product
	err := ctl.BindJSON(&ob)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ob.Id = ctl.Param(":id")
	_, err = ctl.getProductAndCheckCreate(ob.Id)
	if err != nil {
		ctl.RespError(err)
		return
	}
	var update models.Product
	update.Id = ob.Id
	update.Metadata = ob.Metadata
	tslData := tsl.NewTslData()
	err = tslData.FromJson(update.Metadata)
	if err != nil {
		ctl.RespError(err)
		return
	}
	err = product.UpdateProduct(&update)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOk()
}

func (ctl *ProductController) UpdateScript() {
	if ctl.isForbidden(productResource, SaveAction) {
		return
	}
	var ob models.Product
	var err error
	if err = ctl.BindJSON(&ob); err != nil {
		ctl.RespError(err)
		return
	}
	productId := ctl.Param(":id")
	_, err = ctl.getProductAndCheckCreate(productId)
	if err != nil {
		ctl.RespError(err)
		return
	}
	var update models.Product = models.Product{
		Id:     productId,
		Script: ob.Script,
	}

	if err = product.UpdateProduct(&update); err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOk()
}

// get product network config
func (ctl *ProductController) GetNetwork() {
	if ctl.isForbidden(productResource, QueryAction) {
		return
	}
	productId := ctl.Param(":productId")
	_, err := ctl.getProductAndCheckCreate(productId)
	if err != nil {
		ctl.RespError(err)
		return
	}
	nw, err := network.GetByProductId(productId)
	if err != nil {
		ctl.RespError(err)
		return
	}
	if nw == nil {
		ctl.RespError(errors.New("product have no network config"))
		return
	}
	server := servers.GetServer(productId)
	if server == nil {
		nw.State = models.Stop
	} else {
		nw.State = models.Runing
	}
	ctl.RespOkData(nw)
}

// update product network
func (ctl *ProductController) UpdateNetwork() {
	if ctl.isForbidden(productResource, SaveAction) {
		return
	}
	var ob models.Network
	err := ctl.BindJSON(&ob)
	if err != nil {
		ctl.RespError(err)
		return
	}
	if len(ob.ProductId) == 0 {
		ctl.RespError(errors.New("productId must be present"))
		return
	}
	_, err = ctl.getProductAndCheckCreate(ob.ProductId)
	if err != nil {
		ctl.RespError(err)
		return
	}

	nw, err := network.GetByProductId(ob.ProductId)
	if err != nil {
		ctl.RespError(err)
		return
	}
	if nw == nil {
		nw, err = network.GetUnuseNetwork()
		if err != nil {
			ctl.RespError(err)
			return
		}
	}
	ob.Id = nw.Id
	err = network.UpdateNetwork(&ob)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOk()
}

// start server
func (ctl *ProductController) RunNetwork() {
	if ctl.isForbidden(productResource, SaveAction) {
		return
	}
	productId := ctl.Param(":productId")
	state := ctl.Ctx.Input.Query("state")
	produc, err := ctl.getProductAndCheckCreate(productId)
	if err != nil {
		ctl.RespError(err)
		return
	}
	nw, err := network.GetByProductId(productId)
	if err != nil {
		ctl.RespError(err)
		return
	}
	if nw == nil {
		ctl.RespError(errors.New("product not have network, config network first"))
		return
	}
	if len(nw.Type) == 0 {
		ctl.RespError(errors.New("type of network must be present"))
		return
	}
	if len(produc.Script) == 0 {
		ctl.RespError(errors.New("script must be present"))
		return
	}
	if codec.IsNetClientType(nw.Type) {
		ctl.RespError(errors.New("client type net cant run"))
		return
	}
	if state == "start" {
		nw.State = models.Runing
		config, err := convertCodecNetwork(*nw)
		if err != nil {
			ctl.RespError(err)
			return
		}
		err = servers.StartServer(config)
		if err != nil {
			ctl.RespError(err)
			return
		}
	} else if state == "stop" {
		nw.State = models.Stop
		err := servers.StopServer(productId)
		if err != nil {
			ctl.RespError(err)
			return
		}
	} else {
		ctl.RespError(errors.New("state must be start or stop"))
		return
	}
	network.UpdateNetwork(nw)
	ctl.RespOk()
}

func (ctl *ProductController) getProductAndCheckCreate(productId string) (*models.ProductModel, error) {
	ob1, err := product.GetProductMust(productId)
	if err != nil {
		return nil, err
	}
	if ob1.CreateId != ctl.GetCurrentUser().Id {
		return nil, errors.New("product is not you created")
	}
	return ob1, nil
}
