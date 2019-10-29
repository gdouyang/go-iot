package led

import (
	"database/sql"
	"encoding/json"
	"errors"

	"go-iot/models"

	"github.com/astaxie/beego"
	_ "github.com/mattn/go-sqlite3"
)

// 设备
type Device struct {
	Id           string                `json:"id"` //设备ID
	Sn           string                `json:"sn"` //设备SN
	Name         string                `json:"name"`
	Provider     string                `json:"provider"` //厂商
	Type         string                `json:"type"`
	Model        string                `json:"model"`
	OnlineStatus string                `json:"onlineStatus"` //在线状态
	SwitchStatus []models.SwitchStatus `json:"switchStatus"`
	Agent        string                `json:"agent"`
}

func init() {
	db, _ := getDb()
	defer db.Close()
	_, err := db.Exec(`
		CREATE TABLE led (
	    id_ VARCHAR(32) PRIMARY KEY,
	    sn_ VARCHAR(64) NULL,
	    name_ VARCHAR(64) NULL,
		provider_ VARCHAR(32) NULL,
		type_ VARCHAR(32) NULL,
		model_ VARCHAR(32) NULL,
		online_status_ VARCHAR(10) NULL,
		switch_status_ VARCHAR(128) NULL,
		agent_ VARCHAR(32) NULL,
	    created_ DATE NULL
		);
	`)
	if err != nil {
		beego.Info("table led create fail:", err)
	} else {
		beego.Info("table led create success")
	}
}

func getDb() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "./db/led.db")
	if err != nil {
		beego.Info("open sqlite fail")
	}
	return db, err
}

// 分页查询设备
func ListDevice(page *models.PageQuery) (*models.PageResult, error) {
	var pr *models.PageResult
	var dev Device
	json.Unmarshal(page.Condition, &dev)

	//查询数据
	db, _ := getDb()
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

func UpdateDevice(ob *Device) error {
	//更新数据
	db, _ := getDb()
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
		beego.Error("update fail", err)
		return err
	}
	return nil
}

// 根据SN与provider来更新设备在线状态
func UpdateOnlineStatus(onlineStatus string, sn string, provider string) error {
	//更新数据
	db, _ := getDb()
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
		beego.Error("update fail", err)
		return err
	}
	return nil
}

func DeleteDevice(ob *Device) error {
	//更新数据
	db, _ := getDb()
	defer db.Close()
	stmt, _ := db.Prepare("delete from led where id_=?")

	_, err := stmt.Exec(ob.Id)
	if err != nil {
		beego.Error("delete fail", err)
		return err
	}
	return nil
}

func GetDevice(deviceId string) (Device, error) {
	var result Device
	db, _ := getDb()
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

func GetDeviceByProvider(sn, provider string) (Device, error) {
	var result Device
	db, _ := getDb()
	defer db.Close()
	sql := "SELECT id_,sn_,name_,provider_,type_,model_,online_status_,agent_ FROM led where sn_ = ? and provider_ = ?"
	rows, err := db.Query(sql, sn, provider)
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
