package api

import (
	"encoding/base64"
	"errors"
	"go-iot/pkg/api/session"
	"go-iot/pkg/cluster"
	"go-iot/pkg/core/common"
	"go-iot/pkg/models"
	"strings"

	"github.com/beego/beego/v2/server/web"
)

func init() {
	web.Router("/", &MainController{})
}

type MainController struct {
	web.Controller
}

func (c *MainController) Get() {
	c.Data["Website"] = ""
	c.Data["Email"] = "gdouyang@foxmail.com"
	c.TplName = "index.html"
}

// base controllers
type RespController struct {
	web.Controller
}

func (c *RespController) NewSession(expire int) *session.HttpSession {
	session := session.NewSession(expire)
	c.Ctx.Request.Header.Add("x-access-token", session.Sessionid)
	c.Ctx.Output.Cookie("gsessionid", session.Sessionid, expire)
	return session
}

func (c *RespController) GetSession() *session.HttpSession {
	gsessionid := c.Ctx.Request.Header.Get("x-access-token")
	if len(gsessionid) == 0 {
		gsessionid = c.Ctx.Input.Cookie("gsessionid")
	}
	s := session.Get(gsessionid)
	return s
}

// return request param
func (c *RespController) Param(key string) string {
	return c.Ctx.Input.Param(key)
}

func (c *RespController) RespOk() error {
	return c.Ctx.Output.JSON(common.JsonRespOk(), false, false)
}

func (c *RespController) RespOkData(data interface{}) error {
	return c.Ctx.Output.JSON(common.JsonRespOkData(data), false, false)
}

func (c *RespController) RespOkClusterData(data interface{}) error {
	return c.Ctx.Output.JSON(data, false, false)
}

func (c *RespController) RespError(err error) error {
	resp := common.JsonRespError(err)
	if c.Ctx.Output.Status == 0 {
		c.Ctx.Output.Status = 400
		resp.Code = 400
	}
	return c.Ctx.Output.JSON(resp, false, false)
}

func (c *RespController) RespErr(err *common.Err) error {
	resp := common.JsonRespErr(err)
	c.Ctx.Output.Status = err.Code
	return c.Ctx.Output.JSON(resp, false, false)
}

func (c *RespController) Resp(resp common.JsonResp) error {
	c.Ctx.Output.Status = resp.Code
	return c.Ctx.Output.JSON(resp, false, false)
}

// 不是集群内部请求
func (c *RespController) isNotClusterRequest() bool {
	header := c.Ctx.Request.Header.Get(cluster.X_Cluster_Request)
	return !cluster.Enabled() || header != cluster.Token()
}

type AuthController struct {
	RespController
}

func (c *AuthController) Prepare() {
	s := c.GetSession()
	if s == nil {
		// Basic auth认证
		authorization := c.Ctx.Request.Header.Get("Authorization")
		if strings.HasPrefix(authorization, "Basic ") {
			data := strings.Replace(authorization, "Basic ", "", 1)
			by, err := base64.StdEncoding.DecodeString(data)
			if err == nil {
				split := strings.Split(string(by), ":")
				if len(split) == 2 {
					username := split[0]
					password := split[1]
					err := (&LoginController{}).login(&c.RespController, username, password, 0)
					if err != nil {
						c.Ctx.Output.Status = 401
						c.RespError(err)
						return
					}
					return
				}
			}
		}
		c.Ctx.Output.Status = 401
		c.RespError(errors.New("Unauthorized"))
		c.StopRun()
	}
}

func (c *AuthController) isForbidden(r Resource, rc ResourceAction) bool {
	session := c.GetSession()
	permission := session.GetPermission()
	if _, ok := permission[r.Id+":"+rc.Id]; !ok {
		c.Ctx.Output.Status = 403
		c.RespError(errors.New("Forbidden"))
		return true
	}
	return false
}

func (c *AuthController) Logout(ctl *AuthController) {
	sess := ctl.GetSession()
	session.Del(sess.Sessionid)
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
