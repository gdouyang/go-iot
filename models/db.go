package models

import (
	"database/sql"

	"github.com/beego/beego/v2/core/logs"
	_ "github.com/go-sql-driver/mysql"
)

func GetDb() (*sql.DB, error) {
	db, _ := sql.Open("mysql", "root:root@(192.168.31.197:3306)/go-iot")
	err := db.Ping() //连接数据库
	if err != nil {
		logs.Error("数据库连接失败")
	}
	return db, err
}
