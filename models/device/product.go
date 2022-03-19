package models

import (
	"encoding/json"
	"errors"

	"go-iot/models"

	"github.com/beego/beego/v2/core/logs"
)

// 产品
type Product struct {
	Id       string `json:"id"` //产品ID
	Name     string `json:"name"`
	Provider string `json:"provider"` //厂商
	Type     string `json:"type"`
}

func init() {
	db, _ := models.GetDb()
	defer db.Close()
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS product (
	    id_ VARCHAR(32) PRIMARY KEY COMMENT '',
	    name_ VARCHAR(64) NULL COMMENT '',
			type_ VARCHAR(32) NULL COMMENT '',
			model_ VARCHAR(32) NULL COMMENT '',
			create_time_ DATETIME NULL COMMENT '创建时间'
		);
	`)
	if err != nil {
		logs.Info("table led create fail:", err)
	} else {
		logs.Info("table led create success")
	}
}

// 分页查询设备
func ListProduct(page *models.PageQuery) (*models.PageResult, error) {
	var pr *models.PageResult
	var dev Device
	json.Unmarshal(page.Condition, &dev)

	//查询数据
	db, _ := models.GetDb()
	defer db.Close()
	sql := "SELECT id_,sn_,name_,provider_,type_,model_,online_status_,agent_ FROM led "
	countSql := "SELECT count(*) from led"
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
	var result []Device
	var (
		Id, Sn, Name, Provider, Type, Model, OnlineStatus, Agent string
	)
	defer rows.Close()
	for rows.Next() {
		rows.Scan(&Id, &Sn, &Name,
			&Provider, &Type, &Model, &OnlineStatus, &Agent)

		device := Device{Id: Id, Sn: Sn, Name: Name, Provider: Provider,
			Type: Type, Model: Model, OnlineStatus: OnlineStatus, Agent: Agent}
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

func AddProduct(ob *Device) error {
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
	INSERT INTO led(id_, sn_, name_, provider_, type_, model_, online_status_, agent_) 
	values(?,?,?,?,?,?,?,?)
	`)

	_, err = stmt.Exec(ob.Id, ob.Sn, ob.Name, ob.Provider, ob.Type, ob.Model, models.OFFLINE, ob.Agent)
	if err != nil {
		return err
	}

	return nil
}

func UpdateProduct(ob *Device) error {
	//更新数据
	db, _ := models.GetDb()
	defer db.Close()
	stmt, err := db.Prepare(`
	update led 
	set sn_=?,name_=?,provider_=?,type_=?,agent_=?
	where id_=?
	`)
	if err != nil {
		return err
	}

	_, err = stmt.Exec(ob.Sn, ob.Name, ob.Provider, ob.Type, ob.Agent, ob.Id)
	if err != nil {
		logs.Error("update fail", err)
		return err
	}
	return nil
}

func DeleteProduct(ob *Device) error {
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

func GetProduct(deviceId string) (Device, error) {
	var result Device
	db, _ := models.GetDb()
	defer db.Close()
	sql := "SELECT id_,sn_,name_,provider_,type_,model_,online_status_,agent_ FROM led where id_ = ?"
	rows, err := db.Query(sql, deviceId)
	if err != nil {
		return result, err
	}
	var (
		Id, Sn, Name, Provider, Type, Model, OnlineStatus, Agent string
	)
	defer rows.Close()
	for rows.Next() {
		rows.Scan(&Id, &Sn, &Name, &Provider, &Type, &Model, &OnlineStatus, &Agent)
		result = Device{Id: Id, Sn: Sn, Name: Name, Provider: Provider,
			Type: Type, Model: Model, OnlineStatus: OnlineStatus, Agent: Agent}
		break
	}
	return result, nil
}
