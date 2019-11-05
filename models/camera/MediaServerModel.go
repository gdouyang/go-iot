package camera

import (
	"go-iot/models"

	"github.com/astaxie/beego"
	_ "github.com/mattn/go-sqlite3"
)

type MediaServer struct {
	Id     int    `json:"id"` //ID
	Type   string `json:"type"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Port   int    `json:"port"`
}

func init() {
	db, _ := getDb()
	defer db.Close()
	_, err := db.Exec(`
		CREATE TABLE videoserver (
	    id_ INTEGER PRIMARY KEY AUTOINCREMENT,
		type_ VARCHAR(32) NOT NULL,
		name_ VARCHAR(32) NOT NULL,
	    status_ VARCHAR(32) NULL,
		port_ INTEGER NULL
		);
	`)
	if err != nil {
		beego.Info("table videoserver create fail:", err)
		return
	} else {
		beego.Info("table videoserver create success")
	}
	_, err = db.Exec(`
	insert into videoserver (id_,type_,name_,status_,port_) values (1,"rtmp","RTMP_SERVER","off",1935);
	insert into videoserver (id_,type_,name_,status_,port_) values (2,"hls","HLS_SERVER","off",8033);
	insert into videoserver (id_,type_,name_,status_,port_) values (3,"flv","HTTP_FLV_SERVER","off",8034);`)
	if err != nil {
		beego.Info("error accuard with table videoserver init ", err)
	}

}

// 查询
func ListMediaServer(page *models.PageQuery) (*models.PageResult, error) {
	//查询数据
	db, _ := getDb()
	defer db.Close()
	sql := "SELECT id_,type_,name_,status_,port_ FROM videoserver "
	countSql := "SELECT count(*) from videoserver"

	params := make([]interface{}, 0)
	sql += " limit ? offset ?"
	params = append(params, page.PageSize, page.PageOffset())
	rows, err := db.Query(sql, params...)
	if err != nil {
		return nil, err
	}
	var result []*MediaServer
	defer rows.Close()
	for rows.Next() {
		var c = new(MediaServer)
		rows.Scan(&c.Id, &c.Type, &c.Name, &c.Status, &c.Port)
		result = append(result, c)
	}

	rows, err = db.Query(countSql, params...)
	if err != nil {
		return nil, err
	}
	count := 0
	defer rows.Close()
	for rows.Next() {
		rows.Scan(&count)
	}

	pr := models.PageResult{PageSize: page.PageSize, PageNum: page.PageNum, Total: count, List: result}
	return &pr, nil
}

// 获取服务状态
func GetServerAllStatus() ([]*MediaServer, error) {
	db, _ := getDb()
	defer db.Close()
	sql := "SELECT id_,type_,name_,status_,port_ FROM videoserver "
	rows, err := db.Query(sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*MediaServer
	for rows.Next() {
		var c = new(MediaServer)
		rows.Scan(&c.Id, &c.Type, &c.Name, &c.Status, &c.Port)
		result = append(result, c)
	}
	return result, nil
}

// 获取服务状态
func GetServerStatus(srs string) ([]*MediaServer, error) {
	db, _ := getDb()
	defer db.Close()
	sql := "SELECT id_,type_,name_,status_,port_ FROM videoserver where type_ = ?"
	rows, err := db.Query(sql, srs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*MediaServer
	for rows.Next() {
		var c = new(MediaServer)
		rows.Scan(&c.Id, &c.Type, &c.Name, &c.Status, &c.Port)
		result = append(result, c)
	}
	return result, nil
}

// 设置服务状态
func SetServerStatus(status string, name string) error {
	db, _ := getDb()
	defer db.Close()
	sql := `update videoserver set status_ = ? where type_ = ?`
	_, err := db.Exec(sql, status, name)
	if err != nil {
		return err
	}
	return nil
}
