package models

import (
	"go-iot/pkg/es/orm"
)

// RegisterModels registers ORM entity types (ES indices mapping).
func RegisterModels() {
	orm.RegisterModel(
		new(User), new(Role), new(UserRelRole),
		new(MenuResource), new(AuthResource), new(SystemConfig),
		new(Product), new(Device), new(Network),
		new(Rule), new(RuleRelDevice), new(AlarmLog),
		new(Notify), new(DeviceOtaLog), new(OtaFile),
	)
}

// InitDb registers ES models only.
// Process startup (seed data, restore, eventpush) is owned by internal/app.Start.
// Deprecated name kept for tests / old call sites; prefer RegisterModels.
func InitDb() {
	RegisterModels()
}
