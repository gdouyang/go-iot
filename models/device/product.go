package models

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"go-iot/codec"
	"go-iot/models"

	"go-iot/models/network"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
)

// 分页查询设备
func ListProduct(page *models.PageQuery) (*models.PageResult, error) {
	var pr *models.PageResult
	var dev models.Product
	err1 := json.Unmarshal(page.Condition, &dev)
	if err1 != nil {
		return nil, err1
	}

	//查询数据
	o := orm.NewOrm()
	qs := o.QueryTable(models.Product{})

	id := dev.Id
	if len(id) > 0 {
		qs = qs.Filter("id__contains", id)
	}
	if len(dev.Name) > 0 {
		qs = qs.Filter("name__contains", dev.Name)
	}

	count, err := qs.Count()
	if err != nil {
		return nil, err
	}

	var result []models.Product
	var cols = []string{"Id", "Name", "TypeId", "State", "StorePolicy", "Desc", "CreateId", "CreateTime"}
	_, err = qs.Limit(page.PageSize, page.PageOffset()).All(&result, cols...)
	if err != nil {
		return nil, err
	}

	p := models.PageUtil(count, page.PageNum, page.PageSize, result)
	pr = &p

	return pr, nil
}

func ListAllProduct() ([]models.Product, error) {
	//查询数据
	o := orm.NewOrm()
	qs := o.QueryTable(models.Product{})

	var result []models.Product
	var cols = []string{"Id", "Name", "TypeId", "State", "StorePolicy", "Desc", "CreateId", "CreateTime"}
	_, err := qs.All(&result, cols...)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func AddProduct(ob *models.Product, networkType string) error {
	if len(ob.Id) == 0 || len(ob.Name) == 0 {
		return errors.New("id and name not be empty")
	}
	if len(ob.Id) > 32 {
		return errors.New("id length must less 32")
	}
	rs, err := GetProduct(ob.Id)
	if err != nil {
		return err
	}
	if rs != nil {
		return fmt.Errorf("product %s exist", ob.Id)
	}
	//插入数据
	o := orm.NewOrm()
	ob.CreateTime = time.Now()
	err = o.DoTx(func(ctx context.Context, txOrm orm.TxOrmer) error {
		if codec.TCP_CLIENT == codec.NetClientType(networkType) || codec.MQTT_CLIENT == codec.NetClientType(networkType) {
			list := []models.ProductMetaConfig{
				{Property: "host", Type: "string", Buildin: true, Desc: "The host of remote [127.0.0.1]"},
				{Property: "port", Type: "number", Buildin: true, Desc: "The port of remote"},
			}
			if codec.MQTT_CLIENT == codec.NetClientType(networkType) {
				list = append(list, models.ProductMetaConfig{Property: "clientId", Type: "string", Buildin: true, Desc: ""})
				list = append(list, models.ProductMetaConfig{Property: "username", Type: "string", Buildin: true, Desc: ""})
				list = append(list, models.ProductMetaConfig{Property: "password", Type: "password", Buildin: true, Desc: ""})
			}
			c, _ := json.Marshal(list)
			ob.Metaconfig = string(c)
		}
		_, err := txOrm.Insert(ob)
		if err != nil {
			return err
		}
		if codec.IsNetClientType(networkType) {
			err := network.AddNetWork(&models.Network{
				ProductId: ob.Id,
				Type:      networkType,
				State:     models.Stop,
			})
			return err
		} else {
			nw, err := network.GetUnuseNetwork()
			if err != nil {
				return err
			}
			nw.ProductId = ob.Id
			nw.Type = networkType
			err = network.UpdateNetworkTx(nw, txOrm)
			return err
		}
	})

	return err
}

func UpdateProduct(ob *models.Product) error {
	if len(ob.Id) == 0 {
		return errors.New("id not be empty")
	}
	if len(ob.Id) > 32 {
		return errors.New("id length must less 32")
	}
	var columns []string
	if len(ob.Name) > 0 {
		columns = append(columns, "Name")
	}
	if len(ob.TypeId) > 0 {
		columns = append(columns, "TypeId")
	}
	if len(ob.Metadata) > 0 {
		columns = append(columns, "Metadata")
	}
	if len(ob.Metaconfig) > 0 {
		columns = append(columns, "Metaconfig")
	}
	if len(ob.Desc) > 0 {
		columns = append(columns, "Desc")
	}
	if len(columns) == 0 {
		return nil
	}
	//更新数据
	o := orm.NewOrm()
	_, err := o.Update(ob, columns...)
	if err != nil {
		logs.Error("update fail", err)
		return err
	}
	return nil
}

func UpdateProductState(ob *models.Product) error {
	if len(ob.Id) == 0 {
		return errors.New("id not be empty")
	}
	//更新数据
	o := orm.NewOrm()
	_, err := o.Update(ob, "State")
	if err != nil {
		logs.Error("update fail", err)
		return err
	}
	return nil
}

func DeleteProduct(ob *models.Product) error {
	if len(ob.Id) == 0 {
		return errors.New("id not be empty")
	}
	o := orm.NewOrm()
	err := o.DoTx(func(ctx context.Context, txOrm orm.TxOrmer) error {
		_, err := txOrm.Delete(ob)
		if err != nil {
			return err
		}
		nw, err := network.GetByProductId(ob.Id)
		if err != nil {
			return err
		}
		if nw != nil {
			if codec.IsNetClientType(nw.Type) {
				err := network.DeleteNetworkTx(nw, txOrm)
				return err
			} else {
				nw.ProductId = ""
				nw.Type = ""
				nw.Configuration = ""
				nw.Script = ""
				nw.State = "stop"
				err = network.UpdateNetworkTx(nw, txOrm)
				return err
			}
		}
		return err
	})
	return err
}

func GetProduct(id string) (*models.ProductModel, error) {

	o := orm.NewOrm()

	p := models.Product{Id: id}
	err := o.Read(&p)
	if err == orm.ErrNoRows {
		return nil, nil
	} else if err == orm.ErrMissPK {
		return nil, err
	} else {
		m := &models.ProductModel{}
		m.FromEnitty(p)
		return m, nil
	}
}

func GetProductMust(id string) (*models.ProductModel, error) {
	p, err := GetProduct(id)
	if err != nil {
		return nil, err
	} else if p == nil {
		return nil, fmt.Errorf("product [%s] not exist", id)
	}
	return p, nil
}
