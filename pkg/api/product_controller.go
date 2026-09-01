package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"go-iot/pkg/api/web"
	"go-iot/pkg/cluster"
	"go-iot/pkg/core"
	"go-iot/pkg/models"
	product "go-iot/pkg/models/device"
	networkmd "go-iot/pkg/models/network"
	"go-iot/pkg/network"
	"go-iot/pkg/network/servers"
	productsvc "go-iot/pkg/service/product"
	"io"
	"net/http"
	"net/url"
)

var productResource = Resource{
	Id:   "product-mgr",
	Name: "产品管理",
	Sort: 10, // 侧栏：产品管理
	Action: []ResourceAction{
		QueryAction,
		CreateAction,
		SaveAction,
		DeleteAction,
	},
}

// 校验产品用户
func getProductAndCheckCreate(ctl *AuthController, productId string) (*models.ProductModel, error) {
	ob1, err := product.GetProductMust(productId)
	if err != nil {
		return nil, err
	}
	if ob1.CreateId != ctl.GetCurrentUser().Id {
		return nil, errors.New("product is not you created")
	}
	return ob1, nil
}

// 产品管理
func init() {
	RegResource(productResource)

	api := &productApi{}

	web.RegisterAPI("/product/page", "POST", api.page)
	web.RegisterAPI("/product/list", "GET", api.list)
	web.RegisterAPI("/product/store-policies", "GET", api.storePolicies)
	web.RegisterAPI("/product", "POST", api.add)
	web.RegisterAPI("/product/{id}", "PUT", api.update)
	web.RegisterAPI("/product/{id}", "GET", api.get)
	web.RegisterAPI("/product/{id}", "DELETE", api.delete)
	web.RegisterAPI("/product/{id}/deploy", "POST", api.deploy)
	web.RegisterAPI("/product/{id}/undeploy", "POST", api.undeploy)
	web.RegisterAPI("/product/{id}/tsl", "PUT", api.saveTsl)
	web.RegisterAPI("/product/{id}/script", "PUT", api.saveScript)
	web.RegisterAPI("/product/{id}/collector", "PUT", api.saveCollector)
	web.RegisterAPI("/product/{id}/collector", "GET", api.getCollector)
	web.RegisterAPI("/product/network/{productId}", "GET", api.getNetwork)
	web.RegisterAPI("/product/network", "PUT", api.updateNetwork)
	web.RegisterAPI("/product/network/{productId}/run", "POST", api.startNetwork)
	web.RegisterAPI("/product/{id}/export", "GET", api.exportProduct)
	web.RegisterAPI("/product/import", "POST", api.importProduct)

}

type productApi struct {
}

// storePolicies 可选时序存储策略（受 tdengine.enabled 等配置影响）
func (a *productApi) storePolicies(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	if ctl.isForbidden(productResource, QueryAction) {
		return
	}
	ctl.RespOkData(map[string]interface{}{
		"default": core.TIME_SERISE_ES,
		"list":    product.ListStorePolicies(),
	})
}

// 分页查询
func (a *productApi) page(w http.ResponseWriter, r *http.Request) {
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
}

// 查询型号列表
func (a *productApi) list(w http.ResponseWriter, r *http.Request) {
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
}

// 添加型号
func (a *productApi) add(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	if ctl.isForbidden(productResource, CreateAction) {
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
	if !network.IsValidNetType(aligns.NetworkType) {
		ctl.RespError(fmt.Errorf("不支持的网络类型: %s", aligns.NetworkType))
		return
	}
	aligns.CreateId = ctl.GetCurrentUser().Id
	err = productsvc.Add(&aligns)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOk()
}

// 更新型号信息
func (a *productApi) update(w http.ResponseWriter, r *http.Request) {
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
}

func (a *productApi) get(w http.ResponseWriter, r *http.Request) {
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
}

// 删除型号
func (a *productApi) delete(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	if ctl.isForbidden(productResource, DeleteAction) {
		return
	}

	productId := ctl.Param("id")
	if _, err := getProductAndCheckCreate(ctl, productId); err != nil {
		ctl.RespError(err)
		return
	}
	if err := productsvc.Delete(ctl.GetCurrentUser().Id, productId); err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOk()
}

// 产品发布
func (a *productApi) deploy(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	if ctl.isForbidden(productResource, SaveAction) {
		return
	}
	id := ctl.Param("id")
	if _, err := getProductAndCheckCreate(ctl, id); err != nil {
		ctl.RespError(err)
		return
	}
	if ctl.IsNotClusterRequest() {
		if err := productsvc.Deploy(ctl.GetCurrentUser().Id, id); err != nil {
			ctl.RespError(err)
			return
		}
		if p, err := product.GetProductMust(id); err == nil && p.State {
			cluster.BroadcastInvoke(ctl.Request)
		}
	} else {
		if err := productsvc.ApplyProductRuntime(id); err != nil {
			ctl.RespError(err)
			return
		}
	}
	ctl.RespOk()
}

// 撤销发布
func (a *productApi) undeploy(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	if ctl.isForbidden(productResource, SaveAction) {
		return
	}
	productId := ctl.Param("id")
	if _, err := getProductAndCheckCreate(ctl, productId); err != nil {
		ctl.RespError(err)
		return
	}
	if ctl.IsNotClusterRequest() {
		if err := productsvc.Undeploy(ctl.GetCurrentUser().Id, productId); err != nil {
			ctl.RespError(err)
			return
		}
		cluster.BroadcastInvoke(ctl.Request)
	} else {
		core.DeleteProduct(productId)
	}
	ctl.RespOk()
}

// 保存物模型
func (a *productApi) saveTsl(w http.ResponseWriter, r *http.Request) {
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
	if _, err := getProductAndCheckCreate(ctl, ob.Id); err != nil {
		ctl.RespError(err)
		return
	}
	if ctl.IsNotClusterRequest() {
		if err := productsvc.SaveTSL(ctl.GetCurrentUser().Id, ob.Id, ob.Metadata); err != nil {
			ctl.RespError(err)
			return
		}
		if p, err := product.GetProductMust(ob.Id); err == nil && p.State {
			cluster.BroadcastInvoke(ctl.Request)
		}
	} else {
		if err := productsvc.ApplyProductRuntime(ob.Id); err != nil {
			ctl.RespError(err)
			return
		}
	}
	ctl.RespOk()
}

// 保存编解码脚本
func (a *productApi) saveScript(w http.ResponseWriter, r *http.Request) {
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
	if _, err := getProductAndCheckCreate(ctl, productId); err != nil {
		ctl.RespError(err)
		return
	}
	if err := productsvc.SaveScript(ctl.GetCurrentUser().Id, productId, ob.Script); err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOk()
}

func (a *productApi) getCollector(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	if ctl.isForbidden(productResource, QueryAction) {
		return
	}
	id := ctl.Param("id")
	cfg, err := productsvc.GetCollector(ctl.GetCurrentUser().Id, id)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOkData(cfg)
}

func (a *productApi) saveCollector(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	if ctl.isForbidden(productResource, SaveAction) {
		return
	}
	id := ctl.Param("id")
	if _, err := getProductAndCheckCreate(ctl, id); err != nil {
		ctl.RespError(err)
		return
	}
	if ctl.IsNotClusterRequest() {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			ctl.RespError(err)
			return
		}
		if err := productsvc.SaveCollector(ctl.GetCurrentUser().Id, id, body); err != nil {
			ctl.RespError(err)
			return
		}
		if p, err := product.GetProductMust(id); err == nil && p.State {
			cluster.BroadcastInvoke(ctl.Request)
		}
	} else {
		if err := productsvc.ApplyCollectorRuntime(id); err != nil {
			ctl.RespError(err)
			return
		}
	}
	ctl.RespOk()
}

// 查询网络配置
func (a *productApi) getNetwork(w http.ResponseWriter, r *http.Request) {
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
}

// 修改网络配置
func (a *productApi) updateNetwork(w http.ResponseWriter, r *http.Request) {
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
}

// 启动网络
func (a *productApi) startNetwork(w http.ResponseWriter, r *http.Request) {
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
		// 尝试分配一个端口给产品
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
	switch state {
	case "start":
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
	case "stop":
		nw.State = models.Stop
		err := servers.StopServer(productId)
		if err != nil {
			ctl.RespError(err)
			return
		}
	default:
		ctl.RespError(errors.New("state must be start or stop"))
		return
	}
	if ctl.IsNotClusterRequest() {
		networkmd.UpdateNetwork(nw)
		// 调用集群接口
		cluster.BroadcastInvoke(ctl.Request)
	}
	ctl.RespOk()
}

// 产品导出
func (a *productApi) exportProduct(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	if ctl.isForbidden(productResource, SaveAction) {
		return
	}
	productId := ctl.Param("id")
	pd, err := getProductAndCheckCreate(ctl, productId)
	if err != nil {
		ctl.RespError(err)
		return
	}
	dump := productDump{ProductModel: *pd}
	if raw, err2 := productsvc.ExportCollector(ctl.GetCurrentUser().Id, productId); err2 == nil && len(raw) > 0 {
		dump.Collector = raw
	}
	data, err := json.Marshal(dump)
	if err != nil {
		ctl.RespError(err)
		return
	}
	disposition := fmt.Sprintf("attachment; filename=%s.json", url.QueryEscape(productId))
	ctl.HeaderSet("Content-Type", "application/octet-stream")
	ctl.HeaderSet("Content-Disposition", disposition)
	ctl.HeaderSet("Content-Transfer-Encoding", "binary")
	ctl.HeaderSet("Access-Control-Expose-Headers", "Content-Disposition")

	ctl.ResponseWriter.Write(data)
}

// 产品导入
func (a *productApi) importProduct(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	if ctl.isForbidden(productResource, SaveAction) {
		return
	}
	f, _, err := ctl.FormFile("file")
	if err != nil {
		ctl.RespError(err)
		return
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		ctl.RespError(err)
		return
	}
	var dump productDump
	err = json.Unmarshal(data, &dump)
	if err != nil {
		ctl.RespError(err)
		return
	}
	pd := dump.ProductModel
	if len(pd.NetworkType) == 0 {
		ctl.RespErrorParam("networkType")
		return
	}
	if !network.IsValidNetType(pd.NetworkType) {
		ctl.RespError(fmt.Errorf("不支持的网络类型: %s", pd.NetworkType))
		return
	}
	pd.CreateId = ctl.GetCurrentUser().Id
	pd.State = false
	if len(dump.Collector) > 0 {
		if err := productsvc.ValidateCollector(pd.NetworkType, dump.Collector, pd.Metadata); err != nil {
			ctl.RespError(err)
			return
		}
	}
	err = product.AddProduct(&pd)
	if err != nil {
		ctl.RespError(err)
		return
	}
	if len(dump.Collector) > 0 {
		if err := productsvc.SaveCollector(ctl.GetCurrentUser().Id, pd.Id, dump.Collector); err != nil {
			ctl.RespError(err)
			return
		}
	}
	ctl.RespOk()
}

type productDump struct {
	models.ProductModel
	Collector json.RawMessage `json:"collector,omitempty"`
}
