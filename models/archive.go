package models

import (
	"encoding/json"
)

type EventType int

const (
	EVENT_JOIN = iota
	EVENT_LEAVE
	EVENT_MESSAGE
)

const (
	ECHO    = "echo"    // 浏览器使用
	NORTH   = "north"   // 北向接口使用
	ONLINE  = "onLine"  // 在线
	OFFLINE = "offLine" // 离线
	OPEN    = "open"    // 开
	CLOSE   = "close"   // 关
)

type Event struct {
	Type      EventType // JOIN, LEAVE, MESSAGE
	Addr      string
	Timestamp int // Unix timestamp (secs)
	Content   string
}

//分页结果
type PageResult struct {
	PageSize int         `json:"pageSize"`
	PageNum  int         `json:"pageNum"`
	Total    int         `json:"total"`
	List     interface{} `json:"list"`
}

//分页查询
type PageQuery struct {
	PageSize  int             `json:"pageSize"`
	PageNum   int             `json:"pageNum"`
	Condition json.RawMessage `json:"condition"`
}

type JsonResp struct {
	Msg     string          `json:"msg"`
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
}

// 得到数据偏移，默认数据从0开始
func (this *PageQuery) PageOffset() int {
	return (this.PageNum - 1) * this.PageSize
}

// 开关状态
type SwitchStatus struct {
	Index  int    //第几路开关从0开始
	Status string //状态open,close
}
