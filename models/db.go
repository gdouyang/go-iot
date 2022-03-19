package models

import (
	"database/sql"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
	_ "github.com/go-sql-driver/mysql"
)

func init() {
	// set default database username:password@tcp(127.0.0.1:3306)/db_name
	orm.RegisterDataBase("default", "mysql", "root:root@tcp(192.168.31.197:3306)/go-iot?charset=utf8&loc=Local")

	// register model
	orm.RegisterModel(new(Product), new(Device))

	// create table
	orm.RunSyncdb("default", false, true)
}

func GetDb() (*sql.DB, error) {
	db, _ := sql.Open("mysql", "root:root@(192.168.31.197:3306)/go-iot")
	err := db.Ping() //连接数据库
	if err != nil {
		logs.Error("数据库连接失败")
	}
	return db, err
}
