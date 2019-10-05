package models

import (
	"encoding/json"

	"gopkg.in/mgo.v2"
)

type EventType int

const (
	EVENT_JOIN = iota
	EVENT_LEAVE
	EVENT_MESSAGE
)

const (
	ECHO  = "echo"
	NORTH = "north"
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

// 得到数据偏移，默认数据从0开始
func (this *PageQuery) PageOffset() int {
	return (this.PageNum - 1) * this.PageSize
}

//
func mongoExecute(cName string, exec func(collection *mgo.Collection)) {
	session, err := mgo.Dial("127.0.0.1") //Mongodb's connection
	if err != nil {
		panic(err)
	}
	session.SetMode(mgo.Monotonic, true)

	defer session.Close()
	exec(session.DB("iot").C(cName))
}
