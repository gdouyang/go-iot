package models

import (
	"database/sql"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/config"
	"github.com/beego/beego/v2/core/logs"
	_ "github.com/go-sql-driver/mysql"
)

func init() {
	dataSourceName, err := config.String("db-dataSourceName")
	if err != nil {
		logs.Error("get dataSourceName failed")
	}
	// set default database username:password@tcp(127.0.0.1:3306)/db_name
	orm.RegisterDataBase("default", "mysql", dataSourceName)

	// register model
	// orm.RegisterModel(new(Product), new(Device), new(Network))
	orm.RegisterModelWithPrefix("i_", new(User), new(Role), new(UserRelRole), new(Product), new(Device), new(Network))

	// create table
	orm.RunSyncdb("default", false, true)
}

func GetQb() (orm.QueryBuilder, error) {
	return orm.NewQueryBuilder("mysql")
}

func GetDb() (*sql.DB, error) {
	dataSourceName, err := config.String("db-dataSourceName")
	if err != nil {
		logs.Error("get dataSourceName failed")
	}
	db, _ := sql.Open("mysql", dataSourceName)
	err = db.Ping() //连接数据库
	if err != nil {
		logs.Error("数据库连接失败")
	}
	return db, err
}
