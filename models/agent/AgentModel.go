package agent

import (
	"database/sql"
	"encoding/json"
	"errors"
	"go-iot/models"

	"github.com/astaxie/beego"
	_ "github.com/mattn/go-sqlite3"
)

type Agent struct {
	Id           int    `json:"id"` //ID
	Sn           string `json:"sn"` //SN
	Name         string `json:"name"`
	OnlineStatus string `json:"onlineStatus"` //在线状态
}

func init() {
	db, _ := getDb()
	defer db.Close()
	_, err := db.Exec(`
		CREATE TABLE agent (
	    id_ INTEGER PRIMARY KEY AUTOINCREMENT,
	    sn_ VARCHAR(64) NULL,
	    name_ VARCHAR(64) NULL,
		online_status_ VARCHAR(10) NULL,
	    created_ DATE NULL
		);
	`)
	if err != nil {
		beego.Info("table agent create fail:", err)
	} else {
		beego.Info("table agent create success")
	}
}

func getDb() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "./db/agent.db")
	if err != nil {
		beego.Info("open sqlite fail")
	}
	return db, err
}

// 分页查询
func ListAgent(page *models.PageQuery) (*models.PageResult, error) {
	var pr *models.PageResult
	var dev Agent
	json.Unmarshal(page.Condition, &dev)

	//查询数据
	db, _ := getDb()
	defer db.Close()
	sql := "SELECT id_,sn_,name_,online_status_ FROM agent "
	countSql := "SELECT count(*) from agent"
	sn := dev.Sn
	params := make([]interface{}, 0)
	if sn != "" {
		sql += " where sn_ like ?"
		countSql += " where sn_ like ?"
		params = append(params, sn)
	}
	sql += " limit ? offset ?"
	params = append(params, page.PageSize, page.PageOffset())
	rows, err := db.Query(sql, params...)
	if err != nil {
		return nil, err
	}
	var result []Agent
	var (
		Id                     int
		Sn, Name, OnlineStatus string
	)
	defer rows.Close()
	for rows.Next() {
		rows.Scan(&Id, &Sn, &Name, &OnlineStatus)

		agent := Agent{Id: Id, Sn: Sn, Name: Name, OnlineStatus: OnlineStatus}
		result = append(result, agent)
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

	pr = &models.PageResult{PageSize: page.PageSize, PageNum: page.PageNum, Total: count, List: result}

	return pr, nil
}

func AddAgent(ob *Agent) error {
	rs, err := GetAgent(ob.Sn)
	if err != nil {
		return err
	}
	if rs.Sn != "" {
		return errors.New("Agent已存在!")
	}
	//插入数据
	db, _ := getDb()
	defer db.Close()
	stmt, _ := db.Prepare(`
	INSERT INTO agent(sn_, name_, online_status_) values(?,?,?)
	`)

	_, err = stmt.Exec(ob.Sn, ob.Name, models.OFFLINE)
	if err != nil {
		return err
	}

	return nil
}

func UpdateAgent(ob *Agent) error {
	//更新数据
	db, _ := getDb()
	defer db.Close()
	stmt, err := db.Prepare(`
	update agent set sn_=?,name_=? where id_=?
	`)
	if err != nil {
		return err
	}

	_, err = stmt.Exec(ob.Sn, ob.Name, ob.Id)
	if err != nil {
		beego.Error("update fail", err)
		return err
	}
	return nil
}

// 根据SN与provider来更新设备在线状态
func UpdateOnlineStatus(onlineStatus string, sn string) error {
	//更新数据
	db, _ := getDb()
	defer db.Close()
	stmt, err := db.Prepare(`
	update agent 
	set online_status_ = ?
	where sn_ = ?
	`)
	if err != nil {
		return err
	}

	_, err = stmt.Exec(onlineStatus, sn)
	if err != nil {
		beego.Error("update fail", err)
		return err
	}
	return nil
}

func DeleteAgent(ob *Agent) {
	//更新数据
	db, _ := getDb()
	defer db.Close()
	stmt, _ := db.Prepare("delete from agent where id_=?")

	_, err := stmt.Exec(ob.Id)
	if err != nil {
		beego.Error("delete fail", err)
	}
}

func GetAgent(sn string) (Agent, error) {
	var result Agent
	db, _ := getDb()
	defer db.Close()
	sql := "SELECT id_,sn_,name_,online_status_ FROM agent where sn_ = ?"
	rows, err := db.Query(sql, sn)
	if err != nil {
		return result, err
	}
	var (
		Id                     int
		Sn, Name, OnlineStatus string
	)
	defer rows.Close()
	for rows.Next() {
		rows.Scan(&Id, &Sn, &Name, &OnlineStatus)
		result = Agent{Id: Id, Sn: Sn, Name: Name, OnlineStatus: OnlineStatus}
		break
	}
	return result, nil
}
