package models

import (
	"go-iot/pkg/core/boot"

	"github.com/beego/beego/v2/client/orm"
	_ "github.com/go-sql-driver/mysql"
)

type DbConfig struct {
	Url string
}

var DefaultDbConfig DbConfig = DbConfig{Url: "root:root@tcp(localhost:3306)/go-iot?charset=utf8&loc=Local&tls=false"}

func InitDb() {
	// set default database username:password@tcp(127.0.0.1:3306)/db_name
	orm.RegisterDataBase("default", "mysql", DefaultDbConfig.Url)

	// register model
	// orm.RegisterModel(new(Product), new(Device), new(Network))
	orm.RegisterModelWithPrefix("i_",
		new(User), new(Role), new(UserRelRole),
		new(MenuResource), new(AuthResource), new(SystemConfig),
		new(Product), new(Device), new(Network),
		new(Rule), new(RuleRelDevice), new(AlarmLog),
		new(Notify),
	)

	// create table
	orm.RunSyncdb("default", false, true)

	boot.CallStartLinstener()
}
