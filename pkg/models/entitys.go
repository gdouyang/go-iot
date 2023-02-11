package models

import "time"

type User struct {
	Id         int64     `json:"id" orm:"pk;column(id_);auto"` // user id
	Nickname   string    `json:"nickname" orm:"column(nickname_);description(昵称)"`
	Username   string    `json:"username" orm:"column(username_);description(账号)"`
	Password   string    `json:"password,omitempty" orm:"column(password_);description(密码)"`
	Email      string    `json:"email,omitempty" orm:"column(email_);null;description(邮件)"`
	Desc       string    `json:"desc,omitempty" orm:"column(desc_);null;description(备注)"`
	EnableFlag bool      `json:"enableFlag" orm:"column(enable_flag_);description(启用标志1启用，0禁用)"`
	CreateId   int64     `json:"createId" orm:"column(create_id_);null"`
	CreateTime time.Time `json:"createTime" orm:"column(create_time_)"`
}

type Role struct {
	Id         int64     `json:"id" orm:"pk;column(id_);auto"` // role id
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
	Id     string `json:"id" orm:"pk;column(id_);size(64);"`
	Config string `json:"config" orm:"column(config_);type(text);description(配置)"`
}

// 产品
type Product struct {
	Id          string    `json:"id" orm:"pk;column(id_);size(32);description(产品ID)"`
	Name        string    `json:"name" orm:"column(name_);description(名称)"`
	TypeId      string    `json:"typeId" orm:"column(type_id_);null;description(类型)"`
	Metadata    string    `json:"metadata,omitempty" orm:"column(meta_data_);null;type(text);description(物模型)"`      // 物模型
	Metaconfig  string    `json:"metaconfig,omitempty" orm:"column(meta_config_);null;type(text);description(配置属性)"` // 配置属性
	State       bool      `json:"state" orm:"column(state_);description(1启用，0禁用)"`
	StorePolicy string    `json:"storePolicy" orm:"column(store_policy_);size(32);description(数据存储策略 es, mock)"` // 数据存储策略
	Script      string    `json:"script" orm:"column(script_);null;type(text);description(脚本)"`                  // codec脚本
	CodecId     string    `json:"codecId" orm:"column(codec_id_);size(32);null;description(编解码id)"`              // 编解码id
	Tag         string    `json:"tag,omitempty" orm:"column(tag_);null;type(text);description(标签)"`              // 标签
	Desc        string    `json:"desc" orm:"column(desc_);description(产品说明)"`
	CreateId    int64     `json:"createId" orm:"column(create_id_);null"`
	CreateTime  time.Time `json:"createTime" orm:"column(create_time_)"`
}

// 设备
type Device struct {
	Id         string    `json:"id" orm:"pk;column(id_);size(32);description(设备ID)"`
	Name       string    `json:"name" orm:"column(name_);size(64);description(设备名称)"`
	ProductId  string    `json:"productId" orm:"column(product_id_);size(32);description(产品id)"`
	State      string    `json:"state" orm:"column(state_);size(10);description(online,offline,unknow,noActive)"`   // online,offline,unknow,noActive
	Metaconfig string    `json:"metaconfig,omitempty" orm:"column(meta_config_);null;type(text);description(配置属性)"` // 配置属性
	Tag        string    `json:"tag,omitempty" orm:"column(tag_);null;type(text);description(标签)"`                  // 标签
	Desc       string    `json:"desc" orm:"column(desc_);description(产品说明)"`
	CreateId   int64     `json:"createId" orm:"column(create_id_);null"`
	CreateTime time.Time `json:"createTime" orm:"column(create_time_)"`
}

// 网络配置
type Network struct {
	Id            int64     `json:"id" orm:"pk;column(id_);auto"`
	Name          string    `json:"name" orm:"column(name_);size(64);null"`
	Port          int32     `json:"port" orm:"column(port_);description(端口号)"`
	ProductId     string    `json:"productId" orm:"column(product_id_);size(32);null;description(产品id)"`
	Configuration string    `json:"configuration" orm:"column(configuration_);null;type(text);description(网络配置)"`   // 网络配置
	Type          string    `json:"type" orm:"column(type_);size(32);description(网络类型MQTT_BROKER)"`                 // 网络类型MQTT_BROKER
	State         string    `json:"state" orm:"column(state_);size(10);description(运行状态runing,stop)"`               //运行状态runing,stop
	CertBase64    string    `json:"certBase64" orm:"column(cert_base64_);null;type(text);description(crt文件base64)"` // crt文件base64
	KeyBase64     string    `json:"keyBase64" orm:"column(key_base64_);null;type(text);description(key文件base64)"`   // key文件base64
	CreateId      int64     `json:"createId" orm:"column(create_id_);null"`
	CreateTime    time.Time `json:"createTime" orm:"column(create_time_)"`
}

// 规则
type Rule struct {
	Id          int64     `json:"id" orm:"pk;column(id_);auto"`
	Name        string    `json:"name" orm:"column(name_);size(64);null;description(名称)"`
	Type        string    `json:"type" orm:"column(type_);size(10);null;description(scene,alarm)"`
	TriggerType string    `json:"triggerType" orm:"column(trigger_type_);size(32);null;description(触发类型timer,device)"`
	ProductId   string    `json:"productId,omitempty" orm:"column(product_id_);size(64);null;description(产品)"`
	State       string    `json:"state" orm:"column(state_);size(10);description(stop,start)"`
	Cron        string    `json:"cron" orm:"column(cron_);size(32);null;description(cron)"`
	Trigger     string    `json:"trigger,omitempty" orm:"column(trigger_);null;type(text);description(触发)"`
	Actions     string    `json:"actions,omitempty" orm:"column(actions_);null;type(text);description(动作)"`
	Desc        string    `json:"desc" orm:"column(desc_);description(说明)"`
	CreateId    int64     `json:"createId" orm:"column(create_id_);null"`
	CreateTime  time.Time `json:"createTime" orm:"column(create_time_)"`
}

type RuleRelDevice struct {
	Id       int64  `json:"id" orm:"pk;column(id_);auto"`
	RuleId   int64  `json:"ruleId" orm:"column(rule_id_);description(规则ID)"`
	DeviceId string `json:"deviceId,omitempty" orm:"column(device_id_);size(64);description(设备Id)"`
}

// 告警记录
type AlarmLog struct {
	Id         int64     `json:"id" orm:"pk;column(id_);auto"`
	RuleId     int64     `json:"ruleId" orm:"column(rule_id_);description(规则Id)"`
	AlarmName  string    `json:"alarmName" orm:"column(alarm_name_);size(64);null;description(告警名称)"`
	DeviceId   string    `json:"deviceId" orm:"column(device_id_);size(64);null;description(设备ID)"`
	ProductId  string    `json:"productId" orm:"column(product_id_);size(64);null;description(产品ID)"`
	State      string    `json:"state" orm:"column(state_);size(10);description(状态open,solve)"`
	AlarmData  string    `json:"alarmData" orm:"column(alram_data_);type(text);description(告警数据)"`
	CreateId   int64     `json:"createId" orm:"column(create_id_);null"`
	CreateTime time.Time `json:"createTime" orm:"column(create_time_)"`
}

type Notify struct {
	Id         int64     `json:"id" orm:"pk;column(id_);auto"`
	Name       string    `json:"name" orm:"column(name_);size(64);null;description(名称)"`
	Type       string    `json:"type" orm:"column(type_);size(32);null;description(类型)"`
	State      string    `json:"state" orm:"column(state_);size(10);description(状态started,stopped)"`
	Desc       string    `json:"desc" orm:"column(desc_);size(255);description(说明)"`
	Config     string    `json:"config" orm:"column(config_);type(text);description(配置)"`
	Template   string    `json:"template" orm:"column(template_);type(text);description(内容模版)"`
	CreateId   int64     `json:"createId" orm:"column(create_id_);null"`
	CreateTime time.Time `json:"createTime" orm:"column(create_time_)"`
}