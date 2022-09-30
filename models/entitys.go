package models

import "time"

type User struct {
	Id         int64     `json:"id" orm:"pk;column(id_);auto"`
	Nickname   string    `json:"nickname" orm:"column(nickname_);description(昵称)"`
	Username   string    `json:"username" orm:"column(username_);description(账号)"`
	Password   string    `json:"password" orm:"column(password_);description(密码)"`
	CreateId   int64     `json:"createId" orm:"column(create_id_);null"`
	CreateTime time.Time `json:"createTime" orm:"column(create_time_)"`
}

type Role struct {
	Id         int64     `json:"id" orm:"pk;column(id_);auto"`
	Name       string    `json:"name" orm:"column(name_);description(角色名)"`
	CreateId   int64     `json:"createId" orm:"column(create_id_);null"`
	CreateTime time.Time `json:"createTime" orm:"column(create_time_)"`
}

type UserRelRole struct {
	Id     int64 `json:"id" orm:"pk;column(id_);auto"`
	UserId int64 `json:"userId" orm:"column(user_id_);description(用户ID)"`
	RoleId int64 `json:"roleId" orm:"column(role_id_);description(角色ID)"`
}

// 产品
type Product struct {
	Id         string    `json:"id" orm:"pk;column(id_);size(32);description(产品ID)"`
	Name       string    `json:"name" orm:"column(name_);description(名称)"`
	TypeId     string    `json:"typeId" orm:"column(type_id_);null;description(类型)"`
	MetaData   string    `json:"metaData" orm:"column(meta_data_);null;description(物模型)"`
	MetaConfig string    `json:"metaConfig" orm:"column(meta_config_);null;description(配置属性)"`
	CreateId   int64     `json:"createId" orm:"column(create_id_);null"`
	CreateTime time.Time `json:"createTime" orm:"column(create_time_)"`
}

// 设备
type Device struct {
	Id           string    `json:"id" orm:"pk;column(id_);size(32);description(设备ID)"`
	Name         string    `json:"name" orm:"column(name_);size(64);description(设备名称)"`
	ProductId    string    `json:"productId" orm:"column(product_id_);size(32);description(产品id)"`
	OnlineStatus string    `json:"onlineStatus" orm:"column(online_status_);size(10);description(在线状态online,offline)"`
	MetaConfig   string    `json:"metaConfig" orm:"column(meta_config_);null;description(配置属性)"`
	CreateId     int64     `json:"createId" orm:"column(create_id_);null"`
	CreateTime   time.Time `json:"createTime" orm:"column(create_time_)"`
}

// 网络配置
type Network struct {
	Id            int64     `json:"id" orm:"pk;column(id_);auto"`
	Name          string    `json:"name" orm:"column(name_);size(64);null"`
	Port          int32     `json:"port" orm:"column(port_);description(端口号)"`
	ProductId     string    `json:"productId" orm:"column(product_id_);size(32);null;description(产品id)"`
	Configuration string    `json:"configuration" orm:"column(configuration_);null;type(text);description(网络配置)"`
	Script        string    `json:"script" orm:"column(script_);null;type(text);description(脚本)"`
	Type          string    `json:"type" orm:"column(type_);size(32);description(网络类型MQTT_BROKER)"`
	CodecId       string    `json:"codecId" orm:"column(codec_id_);size(32);null;description(编解码id)"`
	CreateId      int64     `json:"createId" orm:"column(create_id_);null"`
	CreateTime    time.Time `json:"createTime" orm:"column(create_time_)"`
}
