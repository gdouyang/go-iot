package models

import "time"

// 产品
type Product struct {
	Id         string    `json:"id" orm:"pk;column(id_);size(32)"` //产品ID
	Name       string    `json:"name" orm:"column(name_)"`
	TypeId     string    `json:"typeId" orm:"column(type_id_)"`
	CreateTime time.Time `json:"createTime" orm:"column(create_time_)"`
}

// 设备
type Device struct {
	Id           string    `json:"id" orm:"pk;column(id_);size(32)"` //设备ID
	Name         string    `json:"name" orm:"column(name_);size(64)"`
	ProductId    string    `json:"productId" orm:"column(product_id_);size(32)"`
	OnlineStatus string    `json:"onlineStatus"` //在线状态
	CreateTime   time.Time `json:"createTime" orm:"column(create_time_)"`
}
