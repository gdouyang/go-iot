package api

import (
	"encoding/json"
	"go-iot/models"
	"go-iot/models/base"

	"github.com/beego/beego/v2/core/logs"
)

var (
	QueryAction  = ResourceAction{Id: "query", Name: "查询"}
	CretaeAction = ResourceAction{Id: "add", Name: "新增"}
	SaveAction   = ResourceAction{Id: "save", Name: "保存"}
	DeleteAction = ResourceAction{Id: "delete", Name: "删除"}
	ImportAction = ResourceAction{Id: "import", Name: "批量导入"}
)

func init() {
	models.OnDbInit(func() {
		for _, r := range resources {
			var m models.MenuResource
			m.Code = r.Id
			m.Name = r.Name
			ac, err := json.Marshal(r.Action)
			if err != nil {
				logs.Error(err)
			}
			m.Action = string(ac)
			old, err := base.GetMenuResourceByCode(m.Code)
			if err != nil {
				logs.Error(err)
				continue
			}
			if old != nil {
				m.Id = old.Id
				base.UpdateMenuResource(&m)
			} else {
				base.AddMenuResource(&m)
			}
		}
		logs.Info("menu resource inited")
	})
}

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
