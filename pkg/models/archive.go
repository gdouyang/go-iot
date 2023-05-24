package models

const (
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
		PageNum:    pageNum,
		PageSize:   pageSize,
		TotalPage:  tp,
		TotalCount: count,
		FirstPage:  pageNum == 1,
		LastPage:   pageNum == tp,
		List:       list,
	}
}

// 分页查询
type PageQuery[T any] struct {
	PageNum   int `json:"pageNum"`
	PageSize  int `json:"pageSize"`
	Condition T   `json:"condition"`
}

// 得到数据偏移，默认数据从0开始
func (page *PageQuery[T]) PageOffset() int {
	return (page.PageNum - 1) * page.PageSize
}
