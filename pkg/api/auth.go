package api

import (
	"encoding/base64"
	"errors"
	"go-iot/pkg/api/web"
	"go-iot/pkg/api/web/session"
	"go-iot/pkg/license"
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
	Id   string
	Name string
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

// isLicenseExemptPath 判断请求路径是否属于 License 检查豁免的白名单路径
func isLicenseExemptPath(path string) bool {
	cleanPath := strings.TrimPrefix(path, web.APIPrefix)
	if !strings.HasPrefix(cleanPath, "/") {
		cleanPath = "/" + cleanPath
	}

	exemptPaths := []string{
		"/login",
		"/logout",
		"/user-info",
		"/license/status",
		"/system/license/status",
		"/system/license/info",
		"/system/license/upload",
	}

	for _, p := range exemptPaths {
		if cleanPath == p || strings.HasPrefix(cleanPath, p+"/") {
			return true
		}
	}
	return false
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
					// Basic auth 成功后继续向下进行 License 检查
					s = c.GetSession()
				}
			}
		}
		if s == nil {
			c.WriteHeader(http.StatusUnauthorized)
			c.RespError(errors.New("Unauthorized"))
			c.StopRun()
			return
		}
	}

	// License 授权状态拦截：未授权或已过期时锁定非白名单业务接口
	if license.Default().IsRequireRedirect() {
		if c.Request != nil && !isLicenseExemptPath(c.Request.URL.Path) {
			c.WriteHeader(http.StatusForbidden)
			c.RespError(errors.New("系统尚未获得有效 License 授权或授权已过期，业务接口已锁定，请联系管理员导入授权证书"))
			c.StopRun()
			return
		}
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
