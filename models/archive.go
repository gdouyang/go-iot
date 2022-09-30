package models

import (
	"encoding/json"
)

const (
	ONLINE  = "onLine"  // 在线
	OFFLINE = "offLine" // 离线
	OPEN    = "open"    // 开
	CLOSE   = "close"   // 关
	// MQTT服务端
	MQTT_BROKER = "MQTT_BROKER"
	// TCP服务端
	TCP_SERVER = "TCP_SERVER"
	// HTTP服务端
	HTTP_SERVER = "HTTP_SERVER"
	// WebSocket服务端
	WEBSOCKET_SERVER = "WEBSOCKET_SERVER"
)

// 分页结果
type PageResult struct {
	PageSize   int         `json:"pageSize"`
	PageNum    int         `json:"pageNum"`
	TotalPage  int         `json:"totalPage"`  // 总页数
	TotalCount int64       `json:"totalCount"` // 总记录数
	FirstPage  bool        `json:"firstPage"`
	LastPage   bool        `json:"lastPage"`
	List       interface{} `json:"list"`
}

func PageUtil(count int64, pageNum int, pageSize int, list interface{}) PageResult {
	tp := int(count) / pageSize
	if int(count)%pageSize > 0 {
		tp = int(count)/pageSize + 1
	}
	return PageResult{
		PageNum: pageNum, PageSize: pageSize, TotalPage: tp, TotalCount: count,
		FirstPage: pageNum == 1, LastPage: pageNum == tp, List: list,
	}
}

// 分页查询
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
	Msg     string      `json:"msg"`
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
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
