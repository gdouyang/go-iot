package models

const (
	ONLINE   = "online"   // 在线
	OFFLINE  = "offline"  // 离线
	NoActive = "noActive" // 未启用

	Runing  = "runing"  // 网络状态runing
	Stop    = "stop"    // 网络状态stop
	Stopped = "stopped" // stopped
	Started = "started" // started
)

// 分页结果
type PageResult[T any] struct {
	PageSize   int   `json:"pageSize"`
	PageNum    int   `json:"pageNum"`
	TotalPage  int   `json:"totalPage"`  // 总页数
	TotalCount int64 `json:"totalCount"` // 总记录数
	FirstPage  bool  `json:"firstPage"`
	LastPage   bool  `json:"lastPage"`
	List       []T   `json:"list"`
}

func PageUtil[T any](count int64, pageNum int, pageSize int, list []T) PageResult[T] {
	tp := int(count) / pageSize
	if int(count)%pageSize > 0 {
		tp = int(count)/pageSize + 1
	}
	return PageResult[T]{
		PageNum: pageNum, PageSize: pageSize, TotalPage: tp, TotalCount: count,
		FirstPage: pageNum == 1, LastPage: pageNum == tp, List: list,
	}
}

// 分页查询
type PageQuery[T any] struct {
	PageSize  int `json:"pageSize"`
	PageNum   int `json:"pageNum"`
	Condition T   `json:"condition"`
}

// 得到数据偏移，默认数据从0开始
func (page *PageQuery[T]) PageOffset() int {
	return (page.PageNum - 1) * page.PageSize
}

type JsonResp struct {
	Msg     string      `json:"message"`
	Success bool        `json:"success"`
	Data    interface{} `json:"result,omitempty"`
	Code    int         `json:"-"` // 20x, 30x, 40x, 50x
}

func JsonRespOk() JsonResp {
	return JsonResp{Success: true, Code: 200}
}

func JsonRespOkData(data interface{}) JsonResp {
	return JsonResp{Success: true, Data: data, Code: 200}
}

func JsonRespError(err error) JsonResp {
	return JsonResp{Success: false, Msg: err.Error(), Code: 400}
}

func JsonRespError1(err error, code int) JsonResp {
	return JsonResp{Success: false, Msg: err.Error(), Code: code}
}