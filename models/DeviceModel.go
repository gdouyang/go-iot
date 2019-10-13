package models

import (
	"encoding/json"
	"errors"

	"database/sql"

	"github.com/astaxie/beego"
	_ "github.com/mattn/go-sqlite3"
)

// 设备
type Device struct {
	Id           string         `json:"id"` //设备ID
	Sn           string         `json:"sn"` //设备SN
	Name         string         `json:"name"`
	Provider     string         `json:"provider"`   //厂商
	OnlineStatus string         `json:onlineStatus` //在线状态
	SwitchStatus []SwitchStatus `json:switchStatus`
}

// 开关状态
type SwitchStatus struct {
	Index  int    //第几路开关从0开始
	Status string //状态open,close
}

func init() {
	db, _ := getDb()
	defer db.Close()
	_, err := db.Exec(`
		CREATE TABLE device (
	    id_ VARCHAR(32) PRIMARY KEY,
	    sn_ VARCHAR(64) NULL,
	    name_ VARCHAR(64) NULL,
		provider_ VARCHAR(32) NULL,
		onlineStatus_ VARCHAR(10) NULL,
		switchStatus_ VARCHAR(128) NULL,
	    created_ DATE NULL
		);
	`)
	if err != nil {
		beego.Info("create table fail")
	}
}

func getDb() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "./goiot.db")
	if err != nil {
		beego.Info("open sqlite fail")
	}
	return db, err
}

// 分页查询设备
func ListDevice(page *PageQuery) (*PageResult, error) {
	var pr *PageResult
	var dev Device
	json.Unmarshal(page.Condition, &dev)

	//查询数据
	db, _ := getDb()
	defer db.Close()
	sql := "SELECT id_,sn_,name_,provider_ FROM device "
	countSql := "SELECT count(*) from device"
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
	defer rows.Close()
	for rows.Next() {
		var id string
		var sn string
		var name string
		var provider string
		rows.Scan(&id, &sn, &name, &provider)
		device := Device{Id: id, Sn: sn, Name: name, Provider: provider}
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

	pr = &PageResult{page.PageSize, page.PageNum, count, result}

	return pr, nil
}

func AddDevie(ob *Device) error {
	rs, err := GetDevice(ob.Id)
	if err != nil {
		return err
	}
	if rs.Id != "" {
		return errors.New("设备已存在!")
	}
	//插入数据
	db, _ := getDb()
	defer db.Close()
	stmt, _ := db.Prepare("INSERT INTO device(id_, sn_, name_, provider_) values(?,?,?,?)")

	_, err = stmt.Exec(ob.Id, ob.Sn, ob.Name, ob.Provider)
	if err != nil {
		return err
	}

	return nil
}

func UpdateDevice(ob *Device) error {
	//更新数据
	db, _ := getDb()
	defer db.Close()
	stmt, err := db.Prepare("update device set sn_=?,name_=?,provider_=? where id_=?")
	if err != nil {
		return err
	}

	_, err = stmt.Exec(ob.Sn, ob.Name, ob.Provider, ob.Id)
	if err != nil {
		beego.Error("update fail", err)
		return err
	}
	return nil
}

func DeleteDevice(ob *Device) {
	//更新数据
	db, _ := getDb()
	defer db.Close()
	stmt, _ := db.Prepare("delete from device where id_=?")

	_, err := stmt.Exec(ob.Id)
	if err != nil {
		beego.Error("delete fail", err)
	}
}

func GetDevice(deviceId string) (Device, error) {
	var result Device
	db, _ := getDb()
	defer db.Close()
	sql := "SELECT id_,sn_,name_,provider_ FROM device where id_ = ?"
	rows, err := db.Query(sql, deviceId)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var sn string
		var name string
		var provider string
		rows.Scan(&id, &sn, &name, &provider)
		result = Device{Id: id, Sn: sn, Name: name, Provider: provider}
		break
	}
	return result, nil
}
