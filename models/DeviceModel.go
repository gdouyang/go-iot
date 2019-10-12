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
	db, err := sql.Open("sqlite3", "./foo.db")
	if err != nil {
		beego.Info("open sqlite fail")
	}
	return db, err
}

// 分页查询设备
func ListDevice(page *PageQuery) *PageResult {
	var pr *PageResult
	var dev Device
	json.Unmarshal(page.Condition, &dev)

	//查询数据
	db, _ := getDb()
	sql := "SELECT id_,sn_,name_,provider_ FROM device "
	id := dev.Id
	params := make([]interface{}, 0)
	if id != "" {
		sql += " where id like ?"
		params = append(params, id)
	}
	//	countSql := sql
	sql += " limit ? offset ?"
	params = append(params, page.PageSize, page.PageOffset())
	rows, _ := db.Query(sql, params...)
	var result []Device
	for rows.Next() {
		var id string
		var sn string
		var name string
		var provider string
		rows.Scan(&id, &sn, &name, &provider)
		device := Device{Id: id, Sn: sn, Name: name, Provider: provider}
		result = append(result, device)
	}

	count := 1
	//	rows := db.Query(countSql)
	//	rows.
	db.Close()

	pr = &PageResult{page.PageSize, page.PageNum, count, result}

	return pr
}

func AddDevie(ob *Device) error {
	rs := GetDevice(ob.Id)
	if rs.Id != "" {
		return errors.New("设备已存在!")
	}
	//插入数据
	db, _ := getDb()
	stmt, _ := db.Prepare("INSERT INTO device(id_, sn_, name_, provider_) values(?,?,?,?)")

	_, err := stmt.Exec(ob.Id, ob.Sn, ob.Name, ob.Provider)
	if err != nil {
		beego.Error("insert fail")
	}
	db.Close()
	return nil
}

func UpdateDevice(ob *Device) {
	//更新数据
	db, _ := getDb()
	stmt, _ := db.Prepare("update device set sn_=?,name_=?,provider_=? where id_=?")

	_, err := stmt.Exec(ob.Sn, ob.Name, ob.Provider, ob.Id)
	if err != nil {
		beego.Error("update fail")
	}
	db.Close()
}

func DeleteDevice(ob *Device) {
	//更新数据
	db, _ := getDb()
	stmt, _ := db.Prepare("delete from device where id_=?")

	_, err := stmt.Exec(ob.Id)
	if err != nil {
		beego.Error("delete fail")
	}
	db.Close()
}

func GetDevice(deviceId string) Device {
	var result Device
	//	mongoExecute("device", func(col *mgo.Collection) {
	//		param := bson.M{}
	//		param["id"] = deviceId
	//		col.Find(param).One(&result)
	//	})
	return result
}
