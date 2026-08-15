package product

import (
	"errors"
	"fmt"
	"strings"

	"go-iot/pkg/codec"
	"go-iot/pkg/core"
	devicemd "go-iot/pkg/models/device"
	"go-iot/pkg/models"
	"go-iot/pkg/network/servers"
	"go-iot/pkg/tsl"
)

var (
	ErrNotOwner       = errors.New("product is not you created")
	ErrNotScriptCodec = errors.New("product codec is not script_codec")
)

// hooks keep ES out of unit tests while HTTP/Agent still call the same functions.
var (
	getProductMust = devicemd.GetProductMust
	updateProduct  = devicemd.UpdateProduct
	updateState    = devicemd.UpdateProductState
	addProduct     = devicemd.AddProduct
	deleteProduct  = devicemd.DeleteProduct
	countDevices   = devicemd.CountDeviceByProductId
	compileOnly    = codec.CompileOnly
	newCodec       = core.NewCodec
	putProduct     = core.PutProduct
	deleteRuntime  = core.DeleteProduct
	getServer      = servers.GetServer
)

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
	if exist.CodecId != core.Script_Codec {
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
	return updateState(&exist.Product)
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
	return deleteProduct(&models.Product{Id: productId})
}
