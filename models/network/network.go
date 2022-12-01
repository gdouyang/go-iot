package network

import (
	"encoding/json"
	"errors"
	"fmt"
	"go-iot/codec"
	"go-iot/models"
	"time"

	"github.com/beego/beego/v2/client/orm"
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
		qs = qs.Filter("id", id)
	}
	if n.Port > 0 {
		qs = qs.Filter("port", n.Port)
	}
	if len(n.Name) > 0 {
		qs = qs.Filter("name__contains", n.Name)
	}
	if len(n.ProductId) > 0 {
		qs = qs.Filter("productId", n.ProductId)
	}
	if len(n.CodecId) > 0 {
		qs = qs.Filter("codecId", n.CodecId)
	}
	if len(n.Type) > 0 {
		qs = qs.Filter("type", n.Type)
	}

	count, err := qs.Count()
	if err != nil {
		return nil, err
	}

	var result []models.Network
	_, err = qs.Limit(page.PageSize, page.PageOffset()).All(&result)
	if err != nil {
		return nil, err
	}

	p := models.PageUtil(count, page.PageNum, page.PageSize, result)
	pr = &p

	return pr, nil
}

func ListStartNetwork() ([]models.Network, error) {
	o := orm.NewOrm()
	qs := o.QueryTable(&models.Network{})

	qs = qs.Filter("State", models.Runing)

	var result []models.Network
	_, err := qs.All(&result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func AddNetWork(ob *models.Network) error {
	if !codec.IsNetClientType(ob.Type) {
		if ob.Port <= 1024 || ob.Port > 65535 {
			return errors.New("invalid port number")
		}
		rs, err := GetNetworkByEntity(models.Network{Port: ob.Port})
		if err != nil {
			return err
		}
		if rs.Id > 0 {
			return fmt.Errorf("network exist of port[%d] ", ob.Port)
		}
	}
	if len(ob.ProductId) > 0 {
		nw, err := GetByProductId(ob.ProductId)
		if err != nil {
			return err
		}
		if nw != nil {
			return fmt.Errorf("network exist of product[%s] ", ob.ProductId)
		}
	}
	//插入数据
	o := orm.NewOrm()
	ob.CreateTime = time.Now()
	_, err := o.Insert(ob)
	if err != nil {
		return err
	}
	return nil
}

func UpdateNetwork(ob *models.Network) error {
	//更新数据
	o := orm.NewOrm()
	err := UpdateNetworkTx(ob, o)
	return err
}

func UpdateNetworkTx(ob *models.Network, o orm.DML) error {
	//更新数据
	var cols []string
	if ob.Port > 0 {
		if ob.Port <= 1024 || ob.Port > 65535 {
			return errors.New("invalid port number")
		}
		cols = append(cols, "Port")
	}
	if len(ob.ProductId) > 0 {
		cols = append(cols, "ProductId")
	}
	if len(ob.Type) > 0 {
		cols = append(cols, "Type")
	}
	if len(ob.Name) > 0 {
		cols = append(cols, "Name")
	}
	if len(ob.Configuration) > 0 {
		cols = append(cols, "Configuration")
	}
	if len(ob.Script) > 0 {
		cols = append(cols, "Script")
	}
	if len(ob.CodecId) > 0 {
		cols = append(cols, "CodecId")
	}
	if len(ob.State) > 0 {
		cols = append(cols, "State")
	}
	if len(cols) == 0 {
		return nil
	}
	_, err := o.Update(ob, cols...)
	return err
}

func DeleteNetwork(ob *models.Network) error {
	o := orm.NewOrm()
	_, err := o.Delete(ob)
	if err != nil {
		return err
	}
	return nil
}

// delete with transaction
func DeleteNetworkTx(ob *models.Network, o orm.DML) error {
	_, err := o.Delete(ob)
	if err != nil {
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

func GetByProductId(productId string) (*models.Network, error) {

	o := orm.NewOrm()

	p := models.Network{ProductId: productId}
	err := o.Read(&p, "productId")
	if err == orm.ErrNoRows {
		return nil, nil
	} else if err == orm.ErrMissPK {
		return nil, err
	} else {
		return &p, nil
	}
}

func GetNetworkByEntity(p models.Network) (*models.Network, error) {

	o := orm.NewOrm()

	err := o.Read(&p)
	if err == orm.ErrNoRows {
		return nil, nil
	} else if err == orm.ErrMissPK {
		return nil, err
	} else {
		return &p, nil
	}
}

func GetUnuseNetwork() (*models.Network, error) {
	o := orm.NewOrm()
	qs := o.QueryTable(&models.Network{})
	cond := orm.NewCondition()
	cond = cond.And("productId__isnull", true).Or("productId", "")
	var result models.Network
	err := qs.SetCond(cond).One(&result)
	if err != nil && err != orm.ErrNoRows {
		return nil, err
	}
	if len(result.ProductId) > 0 {
		return &result, nil
	}
	return nil, errors.New("network is all used")
}
