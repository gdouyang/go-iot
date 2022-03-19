package models

import (
	"encoding/json"
	"errors"

	"go-iot/models"

	"github.com/beego/beego/v2/core/logs"
)

func init() {
}

// 分页查询设备
func ListProduct(page *models.PageQuery) (*models.PageResult, error) {
	var pr *models.PageResult
	var dev models.Product
	json.Unmarshal(page.Condition, &dev)

	//查询数据
	db, _ := models.GetDb()
	defer db.Close()
	sql := "SELECT * FROM prodect "
	countSql := "SELECT count(*) from prodect"
	id := dev.Id
	params := make([]interface{}, 0)
	if id != "" {
		sql += " where id_ like ?"
		countSql += " where id_ like ?"
		params = append(params, id)
	}
	sql += " limit ? offset ?"
	params = append(params, page.PageSize, page.PageOffset())
	rows, err := db.Query(sql, params...)
	if err != nil {
		return nil, err
	}
	var result []models.Product
	var (
		Id, Name, Type string
	)
	defer rows.Close()
	for rows.Next() {
		rows.Scan(&Id, &Name,
			&Type)

		device := models.Product{Id: Id, Name: Name,
			Type: Type}
		result = append(result, device)
	}

	rows, err = db.Query(countSql, params...)
	if err != nil {
		return nil, err
	}
	count := 0
	defer rows.Close()
	for rows.Next() {
		rows.Scan(&count)
		break
	}

	pr = &models.PageResult{page.PageSize, page.PageNum, count, result}

	return pr, nil
}

func AddProduct(ob *models.Product) error {
	rs, err := GetDevice(ob.Id)
	if err != nil {
		return err
	}
	if rs.Id != "" {
		return errors.New("设备已存在!")
	}
	//插入数据
	db, _ := models.GetDb()
	defer db.Close()
	stmt, _ := db.Prepare(`
	INSERT INTO product(id_, name_, type_id_, create_time_ ) 
	values(?,?,?, now())
	`)

	_, err = stmt.Exec(ob.Id, ob.Name, ob.Type)
	if err != nil {
		return err
	}

	return nil
}

func UpdateProduct(ob *models.Product) error {
	//更新数据
	db, _ := models.GetDb()
	defer db.Close()
	stmt, err := db.Prepare(`
	update led 
	set name_=?, type_=?
	where id_=?
	`)
	if err != nil {
		return err
	}

	_, err = stmt.Exec(ob.Name, ob.Type, ob.Id)
	if err != nil {
		logs.Error("update fail", err)
		return err
	}
	return nil
}

func DeleteProduct(ob *models.Product) error {
	//更新数据
	db, _ := models.GetDb()
	defer db.Close()
	stmt, _ := db.Prepare("delete from led where id_=?")

	_, err := stmt.Exec(ob.Id)
	if err != nil {
		logs.Error("delete fail", err)
		return err
	}
	return nil
}

func GetProduct(id string) (models.Product, error) {
	var result models.Product
	db, _ := models.GetDb()
	defer db.Close()
	sql := "SELECT * FROM product where id_ = ?"
	rows, err := db.Query(sql, id)
	if err != nil {
		return result, err
	}
	var (
		Id, Name, Type string
	)
	defer rows.Close()
	for rows.Next() {
		rows.Scan(&Id, &Name, &Type)
		result = models.Product{Id: Id, Name: Name,
			Type: Type}
		break
	}
	return result, nil
}
