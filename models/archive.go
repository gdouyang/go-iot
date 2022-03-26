package models

import (
	"encoding/json"
)

const (
	ONLINE  = "onLine"  // 在线
	OFFLINE = "offLine" // 离线
	OPEN    = "open"    // 开
	CLOSE   = "close"   // 关
	// MQTT
	MQTT_BROKER = "MQTT_BROKER"
)

//分页结果
type PageResult struct {
	PageSize int         `json:"pageSize"`
	PageNum  int         `json:"pageNum"`
	Total    int64       `json:"total"`
	List     interface{} `json:"list"`
}

//分页查询
type PageQuery struct {
	PageSize  int             `json:"pageSize"`
	PageNum   int             `json:"pageNum"`
	Condition json.RawMessage `json:"condition"`
}

// 得到数据偏移，默认数据从0开始
func (this *PageQuery) PageOffset() int {
	return (this.PageNum - 1) * this.PageSize
}

type JsonResp struct {
	Msg     string          `json:"msg"`
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
}

// 开关状态
type SwitchStatus struct {
	Index  int    //第几路开关从0开始
	Status string //状态open,close
}

type IotRequest struct {
	Url  string          `json:"url"`
	Ip   string          `json:"ip"`
	Data json.RawMessage `json:"data"`
}
