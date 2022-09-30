package network

import (
	"encoding/json"
	"errors"
	"go-iot/models"
	"time"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
)

func init() {
}

// 分页查询设备
func ListNetwork(page *models.PageQuery) (*models.PageResult, error) {
	var pr *models.PageResult
	var n models.Network
	err1 := json.Unmarshal(page.Condition, &n)
	if err1 != nil {
		return nil, err1
	}

	//查询数据
	o := orm.NewOrm()
	qs := o.QueryTable(&models.Network{})

	id := n.Id
	if id > 0 {
		qs.Filter("id", id)
	}
	if n.Port > 0 {
		qs.Filter("port", n.Port)
	}
	if len(n.Name) > 0 {
		qs.Filter("name__contains", n.Name)
	}
	if len(n.ProductId) > 0 {
		qs.Filter("productId", n.ProductId)
	}
	if len(n.CodecId) > 0 {
		qs.Filter("codecId", n.CodecId)
	}
	if len(n.Type) > 0 {
		qs.Filter("type", n.Type)
	}
	qs.Offset(page.PageOffset())
	qs.Limit(page.PageSize)

	count, err := qs.Count()
	if err != nil {
		return nil, err
	}

	var result []models.Network
	_, err = qs.All(&result)
	if err != nil {
		return nil, err
	}

	pr = &models.PageResult{
		PageSize: page.PageSize,
		PageNum:  page.PageNum,
		Total:    count,
		Data:     result}

	return pr, nil
}

func AddNetWork(ob *models.Network) error {
	rs, err := GetNetworkByEntity(models.Network{Port: ob.Port})
	if err != nil {
		return err
	}
	if rs.Id > 0 {
		return errors.New("配置已存在")
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

func UpdateNetwork(ob *models.Network) error {
	//更新数据
	o := orm.NewOrm()
	_, err := o.Update(ob, "ProductId", "Name", "Configuration", "Script")
	if err != nil {
		logs.Error("update fail", err)
		return err
	}
	return nil
}

func DeleteNetwork(ob *models.Network) error {
	o := orm.NewOrm()
	_, err := o.Delete(ob)
	if err != nil {
		logs.Error("delete fail", err)
		return err
	}
	return nil
}

func GetNetwork(id int64) (models.Network, error) {

	o := orm.NewOrm()

	p := models.Network{Id: id}
	err := o.Read(&p)
	if err == orm.ErrNoRows {
		return models.Network{}, nil
	} else if err == orm.ErrMissPK {
		return models.Network{}, err
	} else {
		return p, nil
	}
}

func GetNetworkByEntity(p models.Network) (models.Network, error) {

	o := orm.NewOrm()

	err := o.Read(&p)
	if err == orm.ErrNoRows {
		return models.Network{}, nil
	} else if err == orm.ErrMissPK {
		return models.Network{}, err
	} else {
		return p, nil
	}
}
