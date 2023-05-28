package models

import (
	"errors"
	"fmt"

	"go-iot/pkg/core"
	"go-iot/pkg/models"

	"go-iot/pkg/models/network"

	"github.com/beego/beego/v2/client/orm"

	"github.com/beego/beego/v2/core/logs"
)

// 分页查询设备
func PageProduct(page *models.PageQuery[models.Product], createId int64) (*models.PageResult[models.Product], error) {
	var pr *models.PageResult[models.Product]
	var dev models.Product = page.Condition

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
	qs = qs.Filter("createId", createId)

	var result []models.Product
	var cols = []string{"Id", "Name", "TypeId", "State", "StorePolicy", "Desc", "CreateId", "CreateTime"}
	_, err := qs.Limit(page.PageSize, page.PageOffset()).OrderBy("-CreateTime").All(&result, cols...)
	if err != nil {
		return nil, err
	}
	count, err := qs.Count()
	if err != nil {
		return nil, err
	}

	p := models.PageUtil(count, page.PageNum, page.PageSize, result)
	pr = &p

	return pr, nil
}

func ListAllProduct(createId int64) ([]models.Product, error) {
	//查询数据
	o := orm.NewOrm()
	qs := o.QueryTable(models.Product{})
	qs = qs.Filter("createId", createId)

	var result []models.Product
	var cols = []string{"Id", "Name", "TypeId", "State", "StorePolicy", "Desc", "CreateId", "CreateTime"}
	_, err := qs.All(&result, cols...)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func AddProduct(ob *models.ProductModel) error {
	if len(ob.Id) == 0 || len(ob.Name) == 0 {
		return errors.New("id and name must be present")
	}
	if len(ob.Id) > 32 {
		return errors.New("id length must less 32")
	}
	if !DeviceIdValid(ob.Id) {
		return errors.New("productId is invalid")
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
	ob.CreateTime = models.NewDateTime()
	ob.CodecId = core.Script_Codec
	if len(ob.StorePolicy) == 0 {
		ob.StorePolicy = core.TIME_SERISE_ES
	}
	mc := core.GetNetworkMetaConfig(ob.NetworkType)
	if len(mc.CodecId) > 0 {
		ob.CodecId = mc.CodecId
	}
	entity := ob.ToEnitty()
	entity.Metaconfig = mc.ToJson()
	_, err = o.Insert(entity)
	if err != nil {
		return err
	}
	_, err = network.BindNetworkProduct(entity.Id, entity.NetworkType)
	if err != nil {
		logs.Error("bind network error: ", err)
	}
	return err
}

func UpdateProduct(ob *models.ProductModel) error {
	if len(ob.Id) == 0 {
		return errors.New("id must be present")
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
	if len(ob.Script) > 0 {
		columns = append(columns, "Script")
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
		return errors.New("id must be present")
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
		return errors.New("id must be present")
	}
	o := orm.NewOrm()
	num, err := o.Delete(ob)
	if err != nil {
		return err
	}
	if num == 0 {
		return errors.New("product not exist")
	}
	err = network.UnbindNetworkProduct(ob.Id)
	if err != nil {
		return err
	}
	return err
}

func GetProduct(id string) (*models.ProductModel, error) {

	o := orm.NewOrm()

	p := models.Product{Id: id}
	err := o.Read(&p, "id")
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
