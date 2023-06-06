package api

var (
	QueryAction  = ResourceAction{Id: "query", Name: "查询"}
	CretaeAction = ResourceAction{Id: "add", Name: "新增"}
	SaveAction   = ResourceAction{Id: "save", Name: "保存"}
	DeleteAction = ResourceAction{Id: "delete", Name: "删除"}
	ImportAction = ResourceAction{Id: "import", Name: "批量导入"}
)

var resources []Resource

// 注册菜单资源
func regResource(r Resource) {
	resources = append(resources, r)
}

// 权限控制资源
type Resource struct {
	Id     string
	Name   string
	Action []ResourceAction
}

// 资源动作
type ResourceAction struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}
