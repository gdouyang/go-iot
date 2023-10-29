package api

import (
	"errors"
	"go-iot/pkg/api/web"
	"go-iot/pkg/cluster"
	"go-iot/pkg/core"
	"go-iot/pkg/core/tsl"
	"go-iot/pkg/models"
	product "go-iot/pkg/models/device"
	networkmd "go-iot/pkg/models/network"
	"go-iot/pkg/network"
	"go-iot/pkg/network/servers"
	"net/http"
	"strings"
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
	getProductAndCheckCreate := func(ctl *AuthController, productId string) (*models.ProductModel, error) {
		ob1, err := product.GetProductMust(productId)
		if err != nil {
			return nil, err
		}
		if ob1.CreateId != ctl.GetCurrentUser().Id {
			return nil, errors.New("product is not you created")
		}
		return ob1, nil
	}
	// 分页查询
	web.RegisterAPI("/product/page", "POST", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
		if ctl.isForbidden(productResource, QueryAction) {
			return
		}
		var ob models.PageQuery
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
	})
	// 查询型号列表
	web.RegisterAPI("/product/list", "GET", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
		if ctl.isForbidden(productResource, QueryAction) {
			return
		}
		res, err := product.ListAllProduct(ctl.GetCurrentUser().Id)
		if err != nil {
			ctl.RespError(err)
		} else {
			ctl.RespOkData(res)
		}
	})
	// 添加型号
	web.RegisterAPI("/product", "POST", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
		if ctl.isForbidden(productResource, CretaeAction) {
			return
		}
		var aligns models.ProductModel
		err := ctl.BindJSON(&aligns)
		if err != nil {
			ctl.RespError(err)
			return
		}
		if len(aligns.NetworkType) == 0 {
			ctl.RespErrorParam("networkType")
			return
		}
		aligns.CreateId = ctl.GetCurrentUser().Id
		err = product.AddProduct(&aligns)
		if err != nil {
			ctl.RespError(err)
			return
		}
		ctl.RespOk()
	})
	// 更新型号信息
	web.RegisterAPI("/product/{id}", "PUT", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
		if ctl.isForbidden(productResource, SaveAction) {
			return
		}
		var ob models.ProductModel
		err := ctl.BindJSON(&ob)
		if err != nil {
			ctl.RespError(err)
			return
		}
		ob.Id = ctl.Param("id")
		_, err = getProductAndCheckCreate(ctl, ob.Id)
		if err != nil {
			ctl.RespError(err)
			return
		}

		ob.Metadata = ""
		ob.Script = ""
		err = product.UpdateProduct(&ob)
		if err != nil {
			ctl.RespError(err)
			return
		}
		ctl.RespOk()
	})
	web.RegisterAPI("/product/{id}", "GET", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
		if ctl.isForbidden(productResource, QueryAction) {
			return
		}

		id := ctl.Param("id")
		p, err := getProductAndCheckCreate(ctl, id)
		if err != nil {
			ctl.RespError(err)
			return
		}
		nw, err := networkmd.GetByProductId(id)
		if err != nil {
			ctl.RespError(err)
			return
		}
		if nw != nil {
			p.NetworkType = nw.Type
		}
		ctl.RespOkData(p)
	})
	// 删除型号
	web.RegisterAPI("/product/{id}", "DELETE", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
		if ctl.isForbidden(productResource, DeleteAction) {
			return
		}

		productId := ctl.Param("id")
		ob, err := getProductAndCheckCreate(ctl, productId)
		if err != nil {
			ctl.RespError(err)
			return
		}
		server := servers.GetServer(productId)
		if server != nil {
			ctl.RespError(errors.New("网络服务正在运行, 请先停止"))
			return
		}
		total, err := product.CountDeviceByProductId(productId)
		if err != nil {
			ctl.RespError(err)
			return
		}
		if total > 0 {
			ctl.RespError(errors.New("产品下已存在设备, 请先删除设备"))
			return
		}
		productoper, _ := core.NewProduct(productId, map[string]string{}, ob.StorePolicy, "")
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
	})
	// 产品发布
	web.RegisterAPI("/product/{id}/deploy", "POST", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
		if ctl.isForbidden(productResource, SaveAction) {
			return
		}
		id := ctl.Param("id")
		ob, err := getProductAndCheckCreate(ctl, id)
		if err != nil {
			ctl.RespError(err)
			return
		}
		if len(strings.TrimSpace(ob.Metadata)) == 0 {
			ctl.RespError(errors.New("产品没有配置物模型，请先配置"))
			return
		}
		tsl := tsl.TslData{}
		err = tsl.FromJson(ob.Metadata)
		if err != nil {
			ctl.RespError(err)
			return
		}
		if len(tsl.Properties) == 0 {
			ctl.RespError(errors.New("物模型属性为空，请先添加属性"))
			return
		}
		config := map[string]string{}
		for _, v := range ob.Metaconfig {
			config[v.Property] = v.Value
		}
		p1, err := core.NewProduct(ob.Id, config, ob.StorePolicy, ob.Metadata)
		if err != nil {
			ctl.RespError(err)
			return
		}
		core.PutProduct(p1)
		err = p1.GetTimeSeries().PublishModel(p1, tsl)
		if err != nil {
			ctl.RespError(err)
			return
		}
		ob.State = true
		product.UpdateProductState(&ob.Product)
		ctl.RespOk()
	})
	// 撤销发布
	web.RegisterAPI("/product/{id}/undeploy", "POST", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
		if ctl.isForbidden(productResource, SaveAction) {
			return
		}
		productId := ctl.Param("id")
		ob, err := getProductAndCheckCreate(ctl, productId)
		if err != nil {
			ctl.RespError(err)
			return
		}
		ob.State = false
		if ctl.IsNotClusterRequest() {
			product.UpdateProductState(&ob.Product)
			cluster.BroadcastInvoke(ctl.Request)
		}
		core.DeleteProduct(productId)
		// 调用集群接口
		ctl.RespOk()
	})
	// 保存物模型
	web.RegisterAPI("/product/{id}/tsl", "PUT", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
		if ctl.isForbidden(productResource, SaveAction) {
			return
		}
		var ob models.Product
		err := ctl.BindJSON(&ob)
		if err != nil {
			ctl.RespError(err)
			return
		}
		ob.Id = ctl.Param("id")
		_, err = getProductAndCheckCreate(ctl, ob.Id)
		if err != nil {
			ctl.RespError(err)
			return
		}
		var update models.ProductModel
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
	})
	// 保存编解码脚本
	web.RegisterAPI("/product/{id}/script", "PUT", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
		if ctl.isForbidden(productResource, SaveAction) {
			return
		}
		var ob models.Product
		var err error
		if err = ctl.BindJSON(&ob); err != nil {
			ctl.RespError(err)
			return
		}
		productId := ctl.Param("id")
		_, err = getProductAndCheckCreate(ctl, productId)
		if err != nil {
			ctl.RespError(err)
			return
		}
		update := models.ProductModel{}
		update.Id = productId
		update.Script = ob.Script
		if err = product.UpdateProduct(&update); err != nil {
			ctl.RespError(err)
			return
		}
		ctl.RespOk()
	})
	// 查询网络配置
	web.RegisterAPI("/product/network/{productId}", "GET", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
		if ctl.isForbidden(productResource, QueryAction) {
			return
		}
		productId := ctl.Param("productId")
		product, err := getProductAndCheckCreate(ctl, productId)
		if err != nil {
			ctl.RespError(err)
			return
		}
		nw, err := networkmd.GetByProductId(productId)
		if err != nil {
			ctl.RespError(err)
			return
		}
		if nw == nil {
			// client
			nw, err = networkmd.BindNetworkProduct(productId, product.NetworkType)
			if err != nil {
				ctl.RespError(err)
				return
			}
			ctl.RespOkData(nw)
			return
		}
		server := servers.GetServer(productId)
		if server == nil {
			nw.State = models.Stop
		} else {
			nw.State = models.Runing
		}
		ctl.RespOkData(nw)
	})
	// 修改网络配置
	web.RegisterAPI("/product/network", "PUT", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
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
		_, err = getProductAndCheckCreate(ctl, ob.ProductId)
		if err != nil {
			ctl.RespError(err)
			return
		}

		nw, err := networkmd.GetByProductId(ob.ProductId)
		if err != nil {
			ctl.RespError(err)
			return
		}
		if nw == nil {
			nw, err = networkmd.GetUnuseNetwork()
			if err != nil {
				ctl.RespError(err)
				return
			}
		}
		ob.Id = nw.Id
		err = networkmd.UpdateNetwork(&ob)
		if err != nil {
			ctl.RespError(err)
			return
		}
		ctl.RespOk()
	})
	// 启动网络
	web.RegisterAPI("/product/network/{productId}/run", "POST", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
		if ctl.isForbidden(productResource, SaveAction) {
			return
		}
		productId := ctl.Param("productId")
		state := ctl.QueryMust("state")
		produc, err := getProductAndCheckCreate(ctl, productId)
		if err != nil {
			ctl.RespError(err)
			return
		}
		nw, err := networkmd.GetByProductId(productId)
		if err != nil {
			ctl.RespError(err)
			return
		}
		if nw == nil {
			nw, err = networkmd.GetUnuseNetwork()
			if err != nil {
				ctl.RespError(err)
				return
			}
			err = networkmd.UpdateNetwork(&models.Network{
				Id:        nw.Id,
				ProductId: productId,
				Type:      produc.NetworkType,
			})
			if err != nil {
				ctl.RespError(err)
				return
			}
		}
		if len(nw.Type) == 0 {
			ctl.RespError(errors.New("产品没有配置网络类型，请先配置"))
			return
		}
		if len(produc.Script) == 0 {
			ctl.RespError(errors.New("产品没有配置编解码，请先配置"))
			return
		}
		if network.IsNetClientType(nw.Type) {
			ctl.RespError(errors.New("客户端类型产品不能启动网络服务"))
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
		if ctl.IsNotClusterRequest() {
			networkmd.UpdateNetwork(nw)
			// 调用集群接口
			cluster.BroadcastInvoke(ctl.Request)
		}
		ctl.RespOk()
	})

	RegResource(productResource)
}
