package models

import (
	"encoding/json"
)

const (
	ONLINE  = "onLine"  // 在线
	OFFLINE = "offLine" // 离线
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
func (page *PageQuery) PageOffset() int {
	return (page.PageNum - 1) * page.PageSize
}

type JsonResp struct {
	Msg     string      `json:"msg"`
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
}

func JsonRespOk() JsonResp {
	return JsonResp{Success: true}
}

func JsonRespOkData(data interface{}) JsonResp {
	return JsonResp{Success: true, Data: data}
}

func JsonRespError(err error) JsonResp {
	return JsonResp{Success: false, Msg: err.Error()}
}
