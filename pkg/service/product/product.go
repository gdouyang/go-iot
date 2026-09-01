package product

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"go-iot/pkg/codec"
	"go-iot/pkg/core"
	"go-iot/pkg/models"
	devicemd "go-iot/pkg/models/device"
	"go-iot/pkg/network/servers"
	"go-iot/pkg/tsl"
)

var (
	ErrNotOwner       = errors.New("product is not you created")
	ErrNotScriptCodec = errors.New("product codec is not script_codec")
)

// hooks keep ES out of unit tests while HTTP/Agent still call the same functions.
var (
	getProductMust      = devicemd.GetProductMust
	updateProduct       = devicemd.UpdateProduct
	updateState         = devicemd.UpdateProductState
	addProduct          = devicemd.AddProduct
	deleteProduct       = devicemd.DeleteProduct
	countDevices        = devicemd.CountDeviceByProductId
	compileOnly         = codec.CompileOnly
	newCodec            = core.NewCodec
	putProduct          = core.PutProduct
	deleteRuntime       = core.DeleteProduct
	getServer           = servers.GetServer
	getCollectorJSON    = devicemd.GetProductCollectorJSON
	saveCollectorJSON   = devicemd.SaveProductCollectorJSON
	deleteCollectorJSON = devicemd.DeleteProductCollector
)

func init() {
	core.SetCollectorJSONLoader(func(productId string) (string, error) {
		return getCollectorJSON(productId)
	})
}

func AssertOwner(exist *models.ProductModel, userId int64) error {
	if exist == nil || exist.CreateId != userId {
		return ErrNotOwner
	}
	return nil
}

func Add(ob *models.ProductModel) error {
	return addProduct(ob)
}

func SaveTSL(userId int64, productId, metadata string) error {
	exist, err := getProductMust(productId)
	if err != nil {
		return err
	}
	if err := AssertOwner(exist, userId); err != nil {
		return err
	}
	tslData := tsl.NewTslData()
	if err := tslData.FromJson(metadata); err != nil {
		return err
	}
	if raw, err := getCollectorJSON(productId); err != nil {
		return err
	} else if len(raw) > 0 {
		if _, err := core.GetCollectorRuntime(exist.NetworkType).Normalize([]byte(raw), tslData); err != nil {
			return fmt.Errorf("tsl still referenced by collector: %w", err)
		}
	}
	update := models.ProductModel{}
	update.Id = productId
	update.Metadata = tslData.Text
	if err := updateProduct(&update); err != nil {
		return err
	}
	if exist.State {
		exist.Metadata = update.Metadata
		p1, err := exist.ToProeuctOper()
		if err != nil {
			return fmt.Errorf("reload product oper: %w", err)
		}
		if err := putProduct(p1); err != nil {
			return err
		}
		if err := p1.GetTimeSeries().PublishModel(p1, *tslData); err != nil {
			return fmt.Errorf("sync timeseries schema: %w", err)
		}
		core.GetCollectorRuntime(exist.NetworkType).Reload(productId)
	}
	return nil
}

func SaveScript(userId int64, productId, script string) error {
	exist, err := getProductMust(productId)
	if err != nil {
		return err
	}
	if err := AssertOwner(exist, userId); err != nil {
		return err
	}
	if !core.HasCodecCreator(exist.CodecId) {
		return ErrNotScriptCodec
	}
	if err := compileOnly(script); err != nil {
		return err
	}
	update := models.ProductModel{}
	update.Id = productId
	update.Script = script
	if err := updateProduct(&update); err != nil {
		return err
	}
	_, err = newCodec(exist.CodecId, exist.Id, script)
	return err
}

func Deploy(userId int64, productId string) error {
	exist, err := getProductMust(productId)
	if err != nil {
		return err
	}
	if err := AssertOwner(exist, userId); err != nil {
		return err
	}
	if len(strings.TrimSpace(exist.Metadata)) == 0 {
		return errors.New("产品没有配置物模型，请先配置")
	}
	tslData := tsl.TslData{}
	if err := tslData.FromJson(exist.Metadata); err != nil {
		return err
	}
	if len(tslData.Properties) == 0 {
		return errors.New("物模型属性为空，请先添加属性")
	}
	if err := devicemd.ValidateStorePolicy(exist.StorePolicy); err != nil {
		return err
	}
	if raw, err := getCollectorJSON(productId); err != nil {
		return err
	} else if len(raw) > 0 {
		if _, err := core.GetCollectorRuntime(exist.NetworkType).Normalize([]byte(raw), &tslData); err != nil {
			return err
		}
	}
	p1, err := exist.ToProeuctOper()
	if err != nil {
		return err
	}
	if err := putProduct(p1); err != nil {
		return err
	}
	if err := p1.GetTimeSeries().PublishModel(p1, tslData); err != nil {
		return err
	}
	exist.State = true
	if err := updateState(&exist.Product); err != nil {
		return err
	}
	core.GetCollectorRuntime(exist.NetworkType).Reload(productId)
	return nil
}

func Undeploy(userId int64, productId string) error {
	exist, err := getProductMust(productId)
	if err != nil {
		return err
	}
	if err := AssertOwner(exist, userId); err != nil {
		return err
	}
	exist.State = false
	if err := updateState(&exist.Product); err != nil {
		return err
	}
	core.GetCollectorRuntime(exist.NetworkType).Invalidate(productId)
	deleteRuntime(productId)
	return nil
}

func Delete(userId int64, productId string) error {
	exist, err := getProductMust(productId)
	if err != nil {
		return err
	}
	if err := AssertOwner(exist, userId); err != nil {
		return err
	}
	if getServer(productId) != nil {
		return errors.New("网络服务正在运行, 请先停止")
	}
	total, err := countDevices(productId)
	if err != nil {
		return err
	}
	if total > 0 {
		return errors.New("产品下已存在设备, 请先删除设备")
	}
	productoper, _ := core.NewProduct(productId, map[string]string{}, exist.StorePolicy, "")
	if productoper != nil && productoper.GetTimeSeries() != nil {
		if err := productoper.GetTimeSeries().Del(productoper); err != nil {
			return err
		}
	}
	if err := deleteCollectorJSON(productId); err != nil {
		return err
	}
	core.GetCollectorRuntime(exist.NetworkType).Invalidate(productId)
	return deleteProduct(&models.Product{Id: productId})
}

// ValidateCollector 按网络类型解析并校验点表，不落库。
func ValidateCollector(networkType string, raw []byte, metadata string) error {
	tslData := tsl.NewTslData()
	if len(metadata) > 0 {
		if err := tslData.FromJson(metadata); err != nil {
			return err
		}
	}
	_, err := core.GetCollectorRuntime(networkType).Normalize(raw, tslData)
	return err
}

// ExportCollector 导出非空点表 JSON；空配置返回 nil。
func ExportCollector(userId int64, productId string) (json.RawMessage, error) {
	raw, err := GetCollector(userId, productId)
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 || string(raw) == "{}" {
		return nil, nil
	}
	return raw, nil
}

// GetCollector 读点表原文。无配置返回 {}。
func GetCollector(userId int64, productId string) (json.RawMessage, error) {
	exist, err := getProductMust(productId)
	if err != nil {
		return nil, err
	}
	if err := AssertOwner(exist, userId); err != nil {
		return nil, err
	}
	raw, err := getCollectorJSON(productId)
	if err != nil {
		return nil, err
	}
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "null" {
		return json.RawMessage(`{}`), nil
	}
	return json.RawMessage(raw), nil
}

// SaveCollector 校验并写入 product_collector。已发布产品只刷新本机采集缓存与会话。
func SaveCollector(userId int64, productId string, raw []byte) error {
	exist, err := getProductMust(productId)
	if err != nil {
		return err
	}
	if err := AssertOwner(exist, userId); err != nil {
		return err
	}
	tslData := tsl.NewTslData()
	if len(exist.Metadata) > 0 {
		if err := tslData.FromJson(exist.Metadata); err != nil {
			return err
		}
	}
	canonical, err := core.GetCollectorRuntime(exist.NetworkType).Normalize(raw, tslData)
	if err != nil {
		return err
	}
	out := ""
	if len(canonical) > 0 {
		out = string(canonical)
	}
	if err := saveCollectorJSON(productId, out); err != nil {
		return err
	}
	core.GetCollectorRuntime(exist.NetworkType).Reload(productId)
	return nil
}

// ApplyProductRuntime 对端/本机刷新已发布产品运行态（不写 ES）。
func ApplyProductRuntime(productId string) error {
	exist, err := getProductMust(productId)
	if err != nil {
		return err
	}
	if !exist.State {
		return nil
	}
	p1, err := exist.ToProeuctOper()
	if err != nil {
		return err
	}
	if err := putProduct(p1); err != nil {
		return err
	}
	core.GetCollectorRuntime(exist.NetworkType).Reload(productId)
	return nil
}

// ApplyCollectorRuntime 对端刷新点表缓存与本机采集器，不写 Redis 产品。
func ApplyCollectorRuntime(productId string) error {
	exist, err := getProductMust(productId)
	if err != nil {
		return err
	}
	if !exist.State {
		return nil
	}
	core.GetCollectorRuntime(exist.NetworkType).Reload(productId)
	return nil
}
