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

func (s *sessionManager) Login(ctl *web.Controller, u *models.User) {
	session := s.NewSession()
	session.Put("user", u)
	ctl.Ctx.Output.Cookie("gsessionid", session.sessionid)
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

type AuthController struct {
	web.Controller
}

func (c *AuthController) Prepare() {
	s := c.GetSession()
	if s == nil {
		c.Ctx.Output.Status = 401
		c.Data["json"] = models.JsonRespError(errors.New("Unauthorized"))
		c.ServeJSON()
		c.StopRun()
	}
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
