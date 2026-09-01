package api

import (
	"encoding/base64"
	"errors"
	"go-iot/pkg/api/web"
	"go-iot/pkg/api/web/session"
	"go-iot/pkg/models"
	"net/http"
	"strings"
)

var (
	QueryAction  = ResourceAction{Id: "query", Name: "查询"}
	CreateAction = ResourceAction{Id: "add", Name: "新增"}
	SaveAction   = ResourceAction{Id: "save", Name: "保存"}
	DeleteAction = ResourceAction{Id: "delete", Name: "删除"}
	ImportAction = ResourceAction{Id: "import", Name: "批量导入"}
)

var Resources []Resource

// 注册菜单资源
func RegResource(r Resource) {
	Resources = append(Resources, r)
}

// 权限控制资源
type Resource struct {
	Id     string
	Name   string
	// Sort 菜单/权限展示顺序（升序，越小越靠前）。侧栏与角色授权树共用。
	Sort   int32
	Action []ResourceAction
}

// 资源动作
type ResourceAction struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

func NewAuthController(w http.ResponseWriter, r *http.Request) *AuthController {
	ctl := AuthController{}
	ctl.Init(w, r)
	ctl.Prepare()
	return &ctl
}

type AuthController struct {
	web.RespController
}

func (c *AuthController) Prepare() {
	s := c.GetSession()
	if s == nil {
		// Basic auth认证
		authorization := c.Request.Header.Get("Authorization")
		if strings.HasPrefix(authorization, "Basic ") {
			data := strings.Replace(authorization, "Basic ", "", 1)
			by, err := base64.StdEncoding.DecodeString(data)
			if err == nil {
				split := strings.Split(string(by), ":")
				if len(split) == 2 {
					username := split[0]
					password := split[1]
					err := login(&c.RespController, username, password, 0)
					if err != nil {
						c.WriteHeader(http.StatusUnauthorized)
						c.RespError(err)
						return
					}
					return
				}
			}
		}
		c.WriteHeader(http.StatusUnauthorized)
		c.RespError(errors.New("Unauthorized"))
		c.StopRun()
	}
}

func (c *AuthController) isForbidden(r Resource, rc ResourceAction) bool {
	session := c.GetSession()
	if session == nil {
		c.WriteHeader(http.StatusUnauthorized)
		c.RespError(errors.New("Unauthorized"))
		return true
	}
	permission := session.GetPermission()
	if _, ok := permission[r.Id+":"+rc.Id]; !ok {
		c.WriteHeader(http.StatusForbidden)
		c.RespError(errors.New("Forbidden"))
		return true
	}
	return false
}

func (c *AuthController) Logout() {
	// 删除 Redis 中本请求能识别到的会话（header / 各 Path cookie）
	if sess := c.GetSession(); sess != nil {
		session.Del(sess.Sessionid)
	}
	if c.Request != nil {
		for _, ck := range c.Request.Cookies() {
			if ck.Name == web.SessionCookieName && ck.Value != "" {
				session.Del(ck.Value)
			}
		}
	}
	// 清除浏览器里可能残留的多个 gsessionid
	c.ClearSessionCookies()
	c.RespOk()
}

func (c *AuthController) GetCurrentUser() *models.User {
	s := c.GetSession()
	if s == nil {
		return nil
	}
	user := models.User{}
	succ := s.GetObject("user", &user)
	if succ {
		return &user
	}
	return nil
}
