package models

import (
	"go-iot/pkg/core/boot"
	esorm "go-iot/pkg/core/es/orm"

	_ "github.com/go-sql-driver/mysql"
)

type DbConfig struct {
	Url string
}

var DefaultDbConfig DbConfig = DbConfig{Url: "root:root@tcp(localhost:3306)/go-iot?charset=utf8&loc=Local&tls=false"}

func InitDb() {
	// set default database username:password@tcp(127.0.0.1:3306)/db_name
	// orm.RegisterDataBase("default", "mysql", DefaultDbConfig.Url)

	// register model
	// orm.RegisterModel(new(Product), new(Device), new(Network))
	// orm.RegisterModelWithPrefix("i_",
	// 	new(User), new(Role), new(UserRelRole),
	// 	new(MenuResource), new(AuthResource), new(SystemConfig),
	// 	new(Product), new(Device), new(Network),
	// 	new(Rule), new(RuleRelDevice), new(AlarmLog),
	// 	new(Notify),
	// )

	// create table
	// orm.RunSyncdb("default", false, true)

	esorm.RegisterModel(
		new(User), new(Role), new(UserRelRole),
		new(MenuResource), new(AuthResource), new(SystemConfig),
		new(Product), new(Device), new(Network),
		new(Rule), new(RuleRelDevice), new(AlarmLog),
		new(Notify),
	)

	boot.CallStartLinstener()
}

// func GetQb() (orm.QueryBuilder, error) {
// 	return orm.NewQueryBuilder("mysql")
// }

// func GetDb() (*sql.DB, error) {
// 	db, _ := sql.Open("mysql", DefaultDbConfig.Url)
// 	err := db.Ping() //连接数据库
// 	if err != nil {
// 		logs.Error("数据库连接失败")
// 	}
// 	return db, err
// }
