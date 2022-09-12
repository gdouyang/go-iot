package models

import "time"

type User struct {
	Id       string `json:"id" orm:"pk;column(id_);size(32)"` //产品ID
	Name     string `json:"name" orm:"column(name_)"`
	Password string `json:"password" orm:"column(password_)"`
}

type Role struct {
	Id   string `json:"id" orm:"pk;column(id_);size(32)"` //产品ID
	Name string `json:"name" orm:"column(name_)"`
}

// 产品
type Product struct {
	Id         string    `json:"id" orm:"pk;column(id_);size(32)"` //产品ID
	Name       string    `json:"name" orm:"column(name_)"`
	TypeId     string    `json:"typeId" orm:"column(type_id_)"`
	MetaData   string    `json:"metaData" orm:"column(meta_data_)"`
	CreateTime time.Time `json:"createTime" orm:"column(create_time_)"`
}

// 设备
type Device struct {
	Id           string    `json:"id" orm:"pk;column(id_);size(32)"` //设备ID
	Name         string    `json:"name" orm:"column(name_);size(64)"`
	ProductId    string    `json:"productId" orm:"column(product_id_);size(32)"`
	OnlineStatus string    `json:"onlineStatus" orm:"column(online_status_);size(32);description(在线状态)"`
	CreateTime   time.Time `json:"createTime" orm:"column(create_time_)"`
}

// 网络配置
type Network struct {
	Id            string    `json:"id" orm:"pk;column(id_);size(32)"`
	Name          string    `json:"name" orm:"column(name_);size(64)"`
	Port          uint16    `json:"port" orm:"column(port_)"`
	ProductId     string    `json:"productId" orm:"column(product_id_);size(32);description(产品id)"`
	Configuration string    `json:"configuration" orm:"column(configuration_);null;type(text);description(网络配置)"`
	Script        string    `json:"script" orm:"column(script_);null;type(text);description(脚本)"`
	Type          string    `json:"type" orm:"column(type_);size(32);description(网络类型MQTT_BROKER)"`
	CreateTime    time.Time `json:"createTime" orm:"column(create_time_)"`
}
