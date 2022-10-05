package models

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	"go-iot/codec"
	"go-iot/codec/tsl"
	"go-iot/models"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
)

func init() {
	codec.RegProductManager(&DbProductManager{cache: make(map[string]codec.Product)})
}

type DbProductManager struct {
	sync.RWMutex
	cache map[string]codec.Product
}

func (p *DbProductManager) Id() string {
	return "db"
}

func (m *DbProductManager) Get(productId string) codec.Product {
	product, ok := m.cache[productId]
	if ok {
		return product
	}
	if product == nil {
		m.Lock()
		defer m.Unlock()
		data, _ := GetProduct(productId)
		if data == nil {
			m.cache[productId] = nil
			return nil
		}
		d := tsl.TslData{}
		err := d.FromJson(data.MetaData)
		if err != nil {
			logs.Error(err)
		}
		tt := map[string]tsl.TslProperty{}
		for _, p := range d.Properties {
			tt[p.Id] = p
		}

		product = &codec.DefaultProdeuct{
			Id:           data.Id,
			Config:       map[string]interface{}{},
			TimeSeriesId: "es",
			TslProperty:  tt,
		}
		m.Put(product)
	}
	return product
}

func (m *DbProductManager) Put(product codec.Product) {
	if product == nil {
		panic("product not be nil")
	}
	if len(product.GetId()) == 0 {
		panic("product id not be empty")
	}
	m.cache[product.GetId()] = product
}

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
		qs = qs.Filter("id", id)
	}
	if len(dev.Name) > 0 {
		qs = qs.Filter("name__contains", dev.Name)
	}

	count, err := qs.Count()
	if err != nil {
		return nil, err
	}

	var result []models.Product
	_, err = qs.Limit(page.PageSize, page.PageOffset()).All(&result)
	if err != nil {
		return nil, err
	}

	p := models.PageUtil(count, page.PageNum, page.PageSize, result)
	pr = &p

	return pr, nil
}

func AddProduct(ob *models.Product) error {
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
	if len(rs.Id) > 0 {
		return errors.New("设备已存在")
	}
	//插入数据
	o := orm.NewOrm()
	ob.CreateTime = time.Now()
	_, err = o.Insert(ob)
	if err != nil {
		return err
	}
	return nil
}

func UpdateProduct(ob *models.Product) error {
	if len(ob.Id) == 0 || len(ob.Name) == 0 {
		return errors.New("id and name not be empty")
	}
	if len(ob.Id) > 32 {
		return errors.New("id length must less 32")
	}
	var columns []string
	columns = append(columns, "Name")
	if len(ob.TypeId) > 0 {
		columns = append(columns, "TypeId")
	}
	if len(ob.MetaData) > 0 {
		columns = append(columns, "MetaData")
	}
	if len(ob.MetaConfig) > 0 {
		columns = append(columns, "MetaConfig")
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

func DeleteProduct(ob *models.Product) error {
	if len(ob.Id) == 0 {
		return errors.New("id not be empty")
	}
	o := orm.NewOrm()
	_, err := o.Delete(ob)
	if err != nil {
		logs.Error("delete fail", err)
		return err
	}
	return nil
}

func GetProduct(id string) (*models.Product, error) {

	o := orm.NewOrm()

	p := models.Product{Id: id}
	err := o.Read(&p)
	if err == orm.ErrNoRows {
		return nil, nil
	} else if err == orm.ErrMissPK {
		return nil, err
	} else {
		return &p, nil
	}
}
