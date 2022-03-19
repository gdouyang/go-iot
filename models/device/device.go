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
func ListDevice(page *models.PageQuery) (*models.PageResult, error) {
	var pr *models.PageResult
	var dev models.Device
	json.Unmarshal(page.Condition, &dev)

	//查询数据
	db, _ := models.GetDb()
	defer db.Close()
	sql := "SELECT id_,sn_,name_,provider_,type_,model_,online_status_,agent_ FROM device "
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
	var result []models.Device
	var (
		Id, Sn, Name, Provider, Type, Model, OnlineStatus, Agent string
	)
	defer rows.Close()
	for rows.Next() {
		rows.Scan(&Id, &Sn, &Name,
			&Provider, &Type, &Model, &OnlineStatus, &Agent)

		device := models.Device{Id: Id, Name: Name, OnlineStatus: OnlineStatus}
		result = append(result, device)
	}

	rows, err = db.Query(countSql, params...)
	if err != nil {
		return nil, err
	}
	var count int64 = 0
	defer rows.Close()
	for rows.Next() {
		rows.Scan(&count)
		break
	}

	pr = &models.PageResult{page.PageSize, page.PageNum, count, result}

	return pr, nil
}

func AddDevie(ob *models.Device) error {
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

	_, err = stmt.Exec(ob.Id, ob.Name, models.OFFLINE)
	if err != nil {
		return err
	}

	return nil
}

func UpdateDevice(ob *models.Device) error {
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

	_, err = stmt.Exec(ob.Name, ob.Id)
	if err != nil {
		logs.Error("update fail", err)
		return err
	}
	return nil
}

// 根据SN与provider来更新设备在线状态
func UpdateOnlineStatus(onlineStatus string, sn string, provider string) error {
	//更新数据
	db, _ := models.GetDb()
	defer db.Close()
	stmt, err := db.Prepare(`
	update led 
	set online_status_ = ?
	where sn_ = ? and provider_ = ?
	`)
	if err != nil {
		return err
	}

	_, err = stmt.Exec(onlineStatus, sn, provider)
	if err != nil {
		logs.Error("update fail", err)
		return err
	}
	return nil
}

func DeleteDevice(ob *models.Device) error {
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

func GetDevice(deviceId string) (models.Device, error) {
	var result models.Device
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
		result = models.Device{Id: Id, Name: Name, OnlineStatus: OnlineStatus}
		break
	}
	return result, nil
}

func GetDeviceByProvider(sn, provider string) (models.Device, error) {
	var result models.Device
	db, _ := models.GetDb()
	defer db.Close()
	sql := "SELECT id_,sn_,name_,provider_,type_,model_,online_status_,agent_ FROM led where sn_ = ? and provider_ = ?"
	rows, err := db.Query(sql, sn, provider)
	if err != nil {
		return result, err
	}
	var (
		Id, Name, Model, OnlineStatus string
	)
	defer rows.Close()
	for rows.Next() {
		rows.Scan(&Id, &Name, &Model, &OnlineStatus)
		result = models.Device{Id: Id, Name: Name, OnlineStatus: OnlineStatus}
		break
	}
	return result, nil
}
