package api

import (
	"crypto/md5"
	"errors"
	"fmt"
	"go-iot/models"
	"sync"
	"time"

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

// session manager
var defaultSessionManager = &sessionManager{}

type sessionManager struct {
	m sync.Map
}

func (s *sessionManager) Get(key string) *HttpSession {
	val, ok := s.m.Load(key)
	if ok {
		return val.(*HttpSession)
	}
	return nil
}

func (s *sessionManager) Put(session *HttpSession) {
	s.m.Store(session.sessionid, session)
}

func (s *sessionManager) NewSession() *HttpSession {
	sesion := &HttpSession{m: map[string]interface{}{}}
	val := fmt.Sprintf("%d", time.Now().Nanosecond())
	data := []byte(val)
	has := md5.Sum(data)
	sesion.sessionid = fmt.Sprintf("%x", has) //将[]byte转成16进制
	s.Put(sesion)
	return sesion
}

func (s *sessionManager) Del(key string) {
	s.m.Delete(key)
}

func (s *sessionManager) Login(ctl *web.Controller, u *models.User) *HttpSession {
	session := s.NewSession()
	session.Put("user", u)
	ctl.Ctx.Output.Cookie("gsessionid", session.sessionid)
	return session
}

func (s *sessionManager) Logout(ctl *AuthController) {
	session := ctl.GetSession()
	defaultSessionManager.Del(session.sessionid)
}

type HttpSession struct {
	sync.RWMutex
	sessionid string
	m         map[string]interface{}
}

func (s *HttpSession) Get(key string) interface{} {
	s.RLock()
	defer s.RUnlock()
	v := s.m[key]
	return v
}

func (s *HttpSession) Put(key string, value interface{}) {
	s.Lock()
	defer s.Unlock()
	s.m[key] = value
}

func (s *HttpSession) SetPermission(p map[string]bool) {
	s.Put("permissions", p)
}

func (s *HttpSession) GetPermission() map[string]bool {
	p := s.Get("permissions")
	if p == nil {
		return map[string]bool{}
	}
	return p.(map[string]bool)
}

// base controllers
type RespController struct {
	web.Controller
}

func (c *RespController) RespOk() error {
	return c.Ctx.Output.JSON(models.JsonRespOk(), false, false)
}

func (c *RespController) RespOkData(data interface{}) error {
	return c.Ctx.Output.JSON(models.JsonRespOkData(data), false, false)
}

func (c *RespController) RespError(err error) error {
	resp := models.JsonRespError(err)
	if c.Ctx.Output.Status == 0 {
		c.Ctx.Output.Status = 400
		resp.Code = 400
	}
	return c.Ctx.Output.JSON(resp, false, false)
}

func (c *RespController) Resp(resp models.JsonResp) error {
	c.Ctx.Output.Status = resp.Code
	return c.Ctx.Output.JSON(resp, false, false)
}

type AuthController struct {
	RespController
}

func (c *AuthController) Prepare() {
	s := c.GetSession()
	if s == nil {
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

func (c *AuthController) GetSession() *HttpSession {
	gsessionid := c.Ctx.Input.Cookie("gsessionid")
	s := defaultSessionManager.Get(gsessionid)
	return s
}

func (c *AuthController) GetCurrentUser() *models.User {
	s := c.GetSession()
	if s == nil {
		return nil
	}
	val := s.Get("user")
	if val == nil {
		return nil
	}
	return val.(*models.User)
}
