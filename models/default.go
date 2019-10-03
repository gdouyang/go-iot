package models

//
type PageResult struct {
	PageSize int         `json:"pageSize"`
	PageNum  int         `json:"pageNum"`
	Total    int         `json:"total"`
	List     interface{} `json:"list"`
}

type PageQuery struct {
	PageSize  int         `json:"pageSize"`
	PageNum   int         `json:"pageNum"`
	Condition interface{} `json:"condition"`
}

// 设备
type Device struct {
	Id   string `json:"id"`
	Sn   string `json:"sn"`
	Name string `json:"name"`
}
