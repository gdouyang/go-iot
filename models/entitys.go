package models

import "time"

type User struct {
	Id         int64     `json:"id" orm:"pk;column(id_);auto"`
	Nickname   string    `json:"nickname" orm:"column(nickname_);description(昵称)"`
	Username   string    `json:"username" orm:"column(username_);description(账号)"`
	Password   string    `json:"password,omitempty" orm:"column(password_);description(密码)"`
	EnableFlag bool      `json:"enableFlag" orm:"column(enable_flag_);description(启用标志1启用，0禁用)"`
	CreateId   int64     `json:"createId" orm:"column(create_id_);null"`
	CreateTime time.Time `json:"createTime" orm:"column(create_time_)"`
}

type Role struct {
	Id         int64     `json:"id" orm:"pk;column(id_);auto"`
	Name       string    `json:"name" orm:"column(name_);description(角色名)"`
	Desc       string    `json:"desc" orm:"column(desc_);description(描述)"`
	CreateId   int64     `json:"createId" orm:"column(create_id_);null"`
	CreateTime time.Time `json:"createTime" orm:"column(create_time_)"`
}

type UserRelRole struct {
	Id     int64 `json:"id" orm:"pk;column(id_);auto"`
	UserId int64 `json:"userId" orm:"column(user_id_);description(用户ID)"`
	RoleId int64 `json:"roleId" orm:"column(role_id_);description(角色ID)"`
}

type MenuResource struct {
	Id         int64     `json:"id" orm:"pk;column(id_);auto"`
	Name       string    `json:"name" orm:"column(name_);description(角色名)"`
	Code       string    `json:"code" orm:"column(code_);description(资源编码)"`
	Sort       int32     `json:"sort" orm:"column(sort_);description(排序)"`
	Action     string    `json:"action" orm:"column(action_);null;type(text);description(权限集)"`
	CreateId   int64     `json:"createId" orm:"column(create_id_);null"`
	CreateTime time.Time `json:"createTime" orm:"column(create_time_)"`
}

// 授权资源
type AuthResource struct {
	Id         int64     `json:"id" orm:"pk;column(id_);auto"`
	ResType    string    `json:"resType" orm:"column(type_);description(ROLE,USER)"`
	Code       string    `json:"code" orm:"column(resource_code_);description(资源编码)"`
	Sort       int32     `json:"sort" orm:"column(sort_);description(排序)"`
	ObjId      int64     `json:"objId" orm:"column(obj_id_);description(角色id或用户id)"`
	Action     string    `json:"action" orm:"column(action_);null;type(text);description(权限集)"`
	CreateId   int64     `json:"createId" orm:"column(create_id_);null"`
	CreateTime time.Time `json:"createTime" orm:"column(create_time_)"`
}

type SystemConfig struct {
	Id     string `json:"id" orm:"pk;column(id_);"`
	Config string `json:"config" orm:"column(config_);type(text);description(配置)"`
}

// 产品
type Product struct {
	Id     string `json:"id" orm:"pk;column(id_);size(32);description(产品ID)"`
	Name   string `json:"name" orm:"column(name_);description(名称)"`
	TypeId string `json:"typeId" orm:"column(type_id_);null;description(类型)"`
	// 物模型
	MetaData string `json:"metaData" orm:"column(meta_data_);null;type(text);description(物模型)"`
	// 配置属性
	MetaConfig string    `json:"metaConfig" orm:"column(meta_config_);null;type(text);description(配置属性)"`
	CreateId   int64     `json:"createId" orm:"column(create_id_);null"`
	CreateTime time.Time `json:"createTime" orm:"column(create_time_)"`
}

// 设备
type Device struct {
	Id           string `json:"id" orm:"pk;column(id_);size(32);description(设备ID)"`
	Name         string `json:"name" orm:"column(name_);size(64);description(设备名称)"`
	ProductId    string `json:"productId" orm:"column(product_id_);size(32);description(产品id)"`
	OnlineStatus string `json:"onlineStatus" orm:"column(online_status_);size(10);description(在线状态online,offline)"`
	// 配置属性
	MetaConfig string    `json:"metaConfig" orm:"column(meta_config_);null;type(text);description(配置属性)"`
	CreateId   int64     `json:"createId" orm:"column(create_id_);null"`
	CreateTime time.Time `json:"createTime" orm:"column(create_time_)"`
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
