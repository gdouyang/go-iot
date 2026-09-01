package models

import (
	"errors"

	"go-iot/pkg/es/orm"
	"go-iot/pkg/models"

	logs "go-iot/pkg/logger"
)

// GetProductCollectorJSON 读点表 JSON。无行返回空字符串。
func GetProductCollectorJSON(productId string) (string, error) {
	if len(productId) == 0 {
		return "", errors.New("id must be present")
	}
	o := orm.NewOrm()
	row := models.ProductCollector{Id: productId}
	err := o.Read(&row)
	if err == orm.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return row.Config, nil
}

// SaveProductCollectorJSON 空 config 删行，否则按产品 ID upsert。
func SaveProductCollectorJSON(productId, config string) error {
	if len(productId) == 0 {
		return errors.New("id must be present")
	}
	if config == "" {
		return DeleteProductCollector(productId)
	}
	o := orm.NewOrm()
	row := models.ProductCollector{
		Id:         productId,
		Config:     config,
		UpdateTime: models.NewDateTime(),
	}
	exist := models.ProductCollector{Id: productId}
	err := o.Read(&exist)
	if err == orm.ErrNoRows {
		_, err = o.Insert(&row)
		if err != nil {
			logs.Errorf("insert product_collector fail %v", err)
		}
		return err
	}
	if err != nil {
		return err
	}
	_, err = o.Update(&row, "Config", "UpdateTime")
	if err != nil {
		logs.Errorf("update product_collector fail %v", err)
	}
	return err
}

func DeleteProductCollector(productId string) error {
	if len(productId) == 0 {
		return errors.New("id must be present")
	}
	o := orm.NewOrm()
	_, err := o.Delete(&models.ProductCollector{Id: productId})
	if err != nil {
		logs.Errorf("delete product_collector fail %v", err)
	}
	return err
}
