package network

import (
	"errors"
	"fmt"
	"go-iot/pkg/boot"
	"go-iot/pkg/models"
	"go-iot/pkg/network"

	"go-iot/pkg/es/orm"

	logs "go-iot/pkg/logger"
)

func init() {
	boot.AddStartLinstener(func() {
		o := orm.NewOrm()
		qs := o.QueryTable(&models.Network{})
		count, err := qs.Count()
		if err == nil && count == 0 {
			for i := 0; i < 10; i++ {
				AddNetWork(&models.Network{Id: int64(i + 1), Port: int32(9010 + i), State: models.Stop, ProductId: ""})
			}
			logs.Infof("init networks")
		}
	})
}

// 分页查询设备
func PageNetwork(page *models.PageQuery) (*models.PageResult[models.Network], error) {
	var pr *models.PageResult[models.Network]
	//查询数据
	o := orm.NewOrm()
	qs := o.QueryTable(&models.Network{})
	qs.SearchAfter = page.SearchAfter
	qs = qs.FilterTerm(page.Condition...)

	var result []models.Network
	_, err := qs.Limit(page.PageSize, page.PageOffset()).OrderBy("-CreateTime", "-id").All(&result)
	if err != nil {
		return nil, err
	}
	count, err := qs.Count()
	if err != nil {
		return nil, err
	}

	p := models.PageUtil(count, page.PageNum, page.PageSize, result)
	p.SearchAfter = qs.LastSort
	pr = &p

	return pr, nil
}

func AddNetWork(ob *models.Network) error {
	if !network.IsNetClientType(ob.Type) {
		if ob.Port <= 1024 || ob.Port > 65535 {
			return errors.New("invalid port number")
		}
		rs, err := GetNetworkByPort(models.Network{Port: ob.Port})
		if err != nil {
			return err
		}
		if rs != nil && rs.Id > 0 {
			return fmt.Errorf("端口[%d]已被使用", ob.Port)
		}
	}
	if len(ob.ProductId) > 0 {
		nw, err := GetByProductId(ob.ProductId)
		if err != nil {
			return err
		}
		if nw != nil {
			return fmt.Errorf("网络配置已被产品[%s]管理", ob.ProductId)
		}
	}
	//插入数据
	ob.CreateTime = models.NewDateTime()
	o := orm.NewOrm()
	_, err := o.Insert(ob)
	if err != nil {
		return err
	}
	return nil
}

func UpdateNetwork(ob *models.Network) error {
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
	if len(ob.State) > 0 {
		cols = append(cols, "State")
	}
	if len(ob.CertBase64) > 0 {
		cols = append(cols, "CertBase64")
	}
	if len(ob.KeyBase64) > 0 {
		cols = append(cols, "KeyBase64")
	}
	if len(cols) == 0 {
		return nil
	}
	o := orm.NewOrm()
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

func GetNetwork(id int64) (models.Network, error) {

	o := orm.NewOrm()

	p := models.Network{Id: id}
	err := o.Read(&p, "id")
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

func GetNetworkByPort(p models.Network) (*models.Network, error) {

	o := orm.NewOrm()

	err := o.Read(&p, "port")
	if err == orm.ErrNoRows {
		return nil, nil
	} else if err == orm.ErrMissPK {
		return nil, err
	} else {
		return &p, err
	}
}

func GetUnuseNetwork() (*models.Network, error) {
	o := orm.NewOrm()
	qs := o.QueryTable(&models.Network{})
	qs = qs.Filter("productId", "")
	result := []models.Network{}
	_, err := qs.Limit(1, 0).OrderBy("port").All(&result)
	if err != nil && err != orm.ErrNoRows {
		return nil, err
	}
	if len(result) > 0 {
		return &result[0], nil
	}
	return nil, errors.New("没有空闲的端口可以使用")
}

func BindNetworkProduct(productId, networkType string) (*models.Network, error) {
	if network.IsNetClientType(networkType) {
		nw, err := GetByProductId(productId)
		if nw == nil && err == nil {
			AddNetWork(&models.Network{
				ProductId: productId,
				Type:      networkType,
				State:     models.Stop,
			})
		}
		nw, err = GetByProductId(productId)
		return nw, err
	} else {
		nw, err := GetUnuseNetwork()
		if err == nil {
			nw.ProductId = productId
			nw.Type = networkType
			err = UpdateNetwork(nw)
		}
		return nw, err
	}
}

func UnbindNetworkProduct(productId string) error {
	nw, err := GetByProductId(productId)
	if err != nil {
		return err
	}
	if nw != nil {
		if network.IsNetClientType(nw.Type) {
			err := DeleteNetwork(nw)
			return err
		} else {
			nw.ProductId = ""
			nw.Type = ""
			nw.Configuration = ""
			nw.State = "stop"
			o := orm.NewOrm()
			_, err := o.Update(nw, "productId", "type", "Configuration", "State")
			return err
		}
	}
	return nil
}
