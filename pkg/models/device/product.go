package models

import (
	"errors"
	"fmt"

	"go-iot/pkg/core"
	"go-iot/pkg/models"
	"go-iot/pkg/network"
	"go-iot/pkg/option"

	networkmd "go-iot/pkg/models/network"

	"go-iot/pkg/es/orm"

	logs "go-iot/pkg/logger"
)

// ValidateStorePolicy 校验产品时序存储策略是否可用。
func ValidateStorePolicy(storePolicy string) error {
	switch storePolicy {
	case core.TIME_SERISE_ES, core.TIME_SERISE_MOCK, "":
		return nil
	case core.TIME_SERISE_TDENGINE:
		if !option.TdengineEnabled() {
			return errors.New("TDengine 未启用，请在配置中设置 tdengine.enabled: true")
		}
		return nil
	default:
		return fmt.Errorf("不支持的时序存储策略: %s", storePolicy)
	}
}

// ListStorePolicies 返回当前环境可选的时序存储策略（供前端下拉，不含 mock）。
func ListStorePolicies() []map[string]string {
	list := []map[string]string{
		{"value": core.TIME_SERISE_ES, "label": "Elasticsearch"},
	}
	if option.TdengineEnabled() {
		list = append(list, map[string]string{
			"value": core.TIME_SERISE_TDENGINE,
			"label": "TDengine",
		})
	}
	return list
}

// 分页查询设备
func PageProduct(page *models.PageQuery, createId int64) (*models.PageResult[models.Product], error) {
	var pr *models.PageResult[models.Product]

	//查询数据
	o := orm.NewOrm()
	qs := o.QueryTable(models.Product{})
	qs = qs.FilterTerm(page.Condition...)
	qs = qs.Filter("createId", createId)
	qs.SearchAfter = page.SearchAfter
	var result []models.Product
	var cols = []string{"Id", "Name", "TypeId", "NetworkType", "State", "StorePolicy", "RetentionMonths", "Desc", "CreateId", "CreateTime"}
	_, err := qs.Limit(page.PageSize, page.PageOffset()).OrderBy("-CreateTime", "-id").All(&result, cols...)
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

func PageProductAll(page *models.PageQuery) (*models.PageResult[models.Product], error) {
	var pr *models.PageResult[models.Product]

	//查询数据
	o := orm.NewOrm()
	qs := o.QueryTable(models.Product{})
	qs = qs.FilterTerm(page.Condition...)
	qs.SearchAfter = page.SearchAfter
	var result []models.Product
	var cols = []string{"Id", "Name", "NetworkType", "State", "StorePolicy", "RetentionMonths", "Script", "CodecId", "CreateId", "CreateTime"}
	_, err := qs.Limit(page.PageSize, page.PageOffset()).OrderBy("-CreateTime", "-id").All(&result, cols...)
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

func ListAllProduct(createId int64) ([]models.Product, error) {
	//查询数据
	o := orm.NewOrm()
	qs := o.QueryTable(models.Product{})
	qs = qs.Filter("createId", createId)

	var result []models.Product
	var cols = []string{"Id", "Name", "TypeId", "NetworkType", "State", "StorePolicy", "RetentionMonths", "Desc", "CreateId", "CreateTime"}
	_, err := qs.All(&result, cols...)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func AddProduct(ob *models.ProductModel) error {
	ob.Id = NormalizeId(ob.Id)
	if len(ob.Id) == 0 || len(ob.Name) == 0 {
		return errors.New("id and name must be present")
	}
	if len(ob.Id) > 32 {
		return errors.New("产品ID长度不能超过32")
	}
	if !DeviceIdValid(ob.Id) {
		return errors.New("产品ID格式错误")
	}
	if len(ob.NetworkType) == 0 {
		return errors.New("networkType must be present")
	}
	if !network.IsValidNetType(ob.NetworkType) {
		return fmt.Errorf("不支持的网络类型: %s", ob.NetworkType)
	}
	rs, err := getProductById(ob.Id)
	if err != nil {
		return err
	}
	if rs != nil {
		return fmt.Errorf("产品[%s]已存在", ob.Id)
	}
	//插入数据
	ob.CreateTime = models.NewDateTime()
	ob.CodecId = core.Script_Codec
	if len(ob.StorePolicy) == 0 {
		ob.StorePolicy = core.TIME_SERISE_ES
	}
	if err := ValidateStorePolicy(ob.StorePolicy); err != nil {
		return err
	}
	if ob.RetentionMonths != nil && *ob.RetentionMonths < 0 {
		return errors.New("retentionMonths must be >= 0")
	}
	mc := network.GetNetworkMetaConfig(ob.NetworkType)
	if len(mc.CodecId) > 0 {
		ob.CodecId = mc.CodecId
	}
	entity := ob.ToEnitty()
	if len(ob.Metaconfig) == 0 {
		entity.Metaconfig = mc.ToJson()
	}
	if err := ensureCanBindNetwork(entity.NetworkType); err != nil {
		return err
	}
	if err := insertProductEntity(&entity); err != nil {
		return err
	}
	_, err = bindNetworkProduct(entity.Id, entity.NetworkType)
	if err != nil {
		logs.Errorf("bind network error: %v", err)
		if delErr := deleteProductRow(entity.Id); delErr != nil {
			logs.Errorf("compensate delete product %s after bind fail: %v", entity.Id, delErr)
		}
		return err
	}
	return nil
}

// hooks allow unit tests to drive AddProduct without Elasticsearch.
var (
	getProductById      = GetProduct
	lookupUnusedNetwork = networkmd.GetUnuseNetwork
	insertProductEntity = func(entity *models.Product) error {
		o := orm.NewOrm()
		_, err := o.Insert(entity)
		return err
	}
	bindNetworkProduct = networkmd.BindNetworkProduct
	deleteProductRow   = func(id string) error {
		return DeleteProduct(&models.Product{Id: id})
	}
)

func ensureCanBindNetwork(networkType string) error {
	if network.IsNetClientType(networkType) {
		return nil
	}
	_, err := lookupUnusedNetwork()
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
	// RetentionMonths：>=0 写入（0=跟随系统全局，>0=产品配置）；-1 清空为 null（同样跟随全局）
	if ob.RetentionMonths != nil {
		if *ob.RetentionMonths < -1 {
			return errors.New("retentionMonths must be >= 0 or -1 to clear")
		}
		if *ob.RetentionMonths == -1 {
			ob.RetentionMonths = nil
		}
		columns = append(columns, "RetentionMonths")
	}
	if len(columns) == 0 {
		return nil
	}
	//更新数据
	o := orm.NewOrm()
	_, err := o.Update(ob.ToEnitty(), columns...)
	if err != nil {
		logs.Errorf("update fail %v", err)
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
		logs.Errorf("update fail %v", err)
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
	err = networkmd.UnbindNetworkProduct(ob.Id)
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
