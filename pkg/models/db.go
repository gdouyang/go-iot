package models

import (
	"go-iot/pkg/core/boot"

	"go-iot/pkg/core/es/orm"
)

func InitDb() {
	// register model
	orm.RegisterModel(
		new(User), new(Role), new(UserRelRole),
		new(MenuResource), new(AuthResource), new(SystemConfig),
		new(Product), new(Device), new(Network),
		new(Rule), new(RuleRelDevice), new(AlarmLog),
		new(Notify),
	)

	// create table
	boot.CallStartLinstener()
}
