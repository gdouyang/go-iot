package web

import (
	"encoding/json"
	"fmt"
	"go-iot/pkg/api/web/session"
	"go-iot/pkg/cluster"
	"go-iot/pkg/common"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

type ControllerInterface interface {
	Init(http.ResponseWriter, *http.Request)
	Prepare()
}

// base controllers
type RespController struct {
	Request        *http.Request
	ResponseWriter http.ResponseWriter
}

func NewController(w http.ResponseWriter, r *http.Request) *RespController {
	ctl := &RespController{}
	ctl.Init(w, r)
	ctl.Prepare()
	return ctl
}

func (c *RespController) Init(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	c.Request = r
	c.ResponseWriter = w
}

func (c *RespController) Prepare() {
}

// SessionCookieName 管理端会话 Cookie 名。
const SessionCookieName = "gsessionid"

// expire sec default 1 hour
func (c *RespController) NewSession(expire int) *session.HttpSession {
	if expire <= 0 {
		expire = session.DefaultExpireSec
	}
	s := session.NewSession(expire)
	c.Request.Header.Set("x-access-token", s.Sessionid)
	// 登录时清掉历史 Path 上的旧 gsessionid，再写统一 Path=/，避免双 cookie
	c.writeSessionCookie(s)
	return s
}

// sessionTokenFromRequest 优先读请求头 token（前端 pinia 默认 key 为 Authorization）。
// 顺序：x-access-token → Authorization（非 Basic）→ 空。
func (c *RespController) sessionTokenFromRequest() string {
	if c == nil || c.Request == nil {
		return ""
	}
	if t := strings.TrimSpace(c.Request.Header.Get("x-access-token")); t != "" {
		return t
	}
	auth := strings.TrimSpace(c.Request.Header.Get("Authorization"))
	if auth == "" {
		return ""
	}
	lower := strings.ToLower(auth)
	// Basic 留给 AuthController.Prepare 做账号密码登录
	if strings.HasPrefix(lower, "basic ") {
		return ""
	}
	if len(auth) > 7 && strings.EqualFold(auth[:7], "bearer ") {
		return strings.TrimSpace(auth[7:])
	}
	// 前端常把 sessionId 直接放在 Authorization 值里
	return auth
}

// expireSessionCookie 删除指定 Path 上的 gsessionid（须与当初 Set 的 Path 一致才能删掉）。
func (c *RespController) expireSessionCookie(path string) {
	if c == nil || c.ResponseWriter == nil {
		return
	}
	// 过期时间用固定 epoch，兼容旧浏览器
	const expired = "Thu, 01 Jan 1970 00:00:00 GMT"
	var cookie string
	if path == "" {
		// 无 Path：匹配「未写 Path」的历史 cookie
		cookie = fmt.Sprintf("%s=; Max-Age=0; Expires=%s; HttpOnly; SameSite=Lax", SessionCookieName, expired)
	} else {
		cookie = fmt.Sprintf("%s=; Path=%s; Max-Age=0; Expires=%s; HttpOnly; SameSite=Lax",
			SessionCookieName, path, expired)
	}
	// 必须用 Add：多个 Set-Cookie 不能 Set 互相覆盖
	c.ResponseWriter.Header().Add("Set-Cookie", cookie)
}

// ClearSessionCookies 清除常见 Path 上的 gsessionid，避免浏览器残留两个同名 cookie。
func (c *RespController) ClearSessionCookies() {
	// 历史版本可能写过 /api 或未写 Path
	for _, p := range []string{"/", "/api", ""} {
		c.expireSessionCookie(p)
	}
}

// writeSessionCookie 写入/刷新 gsessionid（Path=/），并顺带清掉其它 Path 上的旧副本。
func (c *RespController) writeSessionCookie(s *session.HttpSession) {
	if c == nil || c.ResponseWriter == nil || s == nil || len(s.Sessionid) == 0 {
		return
	}
	// 先清 /api、无 Path 旧 cookie，避免与 Path=/ 并存
	c.expireSessionCookie("/api")
	c.expireSessionCookie("")

	ttl := s.TTL()
	expires := time.Now().Add(time.Duration(ttl) * time.Second).UTC().Format(time.RFC1123)
	cookie := fmt.Sprintf("%s=%s; Path=/; Expires=%s; Max-Age=%d; HttpOnly; SameSite=Lax",
		SessionCookieName, s.Sessionid, expires, ttl)
	c.ResponseWriter.Header().Add("Set-Cookie", cookie)
}

// GetSession 解析当前会话。Header token 优先于 Cookie，避免「新 token + 旧 cookie」用错会话。
func (c *RespController) GetSession() *session.HttpSession {
	// 1) 请求头（x-access-token / Authorization）
	if tok := c.sessionTokenFromRequest(); tok != "" {
		if s := session.Get(tok); s != nil {
			c.writeSessionCookie(s)
			return s
		}
	}
	// 2) 可能存在多个同名 cookie（不同 Path），逐个尝试直到 Redis 命中
	if c.Request != nil {
		for _, ck := range c.Request.Cookies() {
			if ck.Name != SessionCookieName || ck.Value == "" {
				continue
			}
			if s := session.Get(ck.Value); s != nil {
				// 命中后统一写回 Path=/，并清掉其它 Path 副本
				c.writeSessionCookie(s)
				return s
			}
		}
	}
	return nil
}

// return request path value
func (c *RespController) Param(key string) string {
	return chi.URLParam(c.Request, key)
}

// request param from form
func (c *RespController) Query(key string) string {
	return c.Request.Form.Get(key)
}

func (c *RespController) QueryMust(key string) string {
	value := c.Query(key)
	if len(value) == 0 {
		panic(common.Err{Code: http.StatusBadRequest, Message: fmt.Sprintf("request param '%s' is not present", key)})
	}
	return value
}

func (c *RespController) BindJSON(obj interface{}) error {
	return json.NewDecoder(c.Request.Body).Decode(obj)
}

func (c *RespController) JSON(data any) error {
	c.ResponseWriter.Header().Add("Content-Type", "application/json; charset=utf-8")
	if resp, ok := data.(common.JsonResp); ok {
		c.WriteHeader(resp.Code)
	} else if resp, ok := data.(*common.JsonResp); ok {
		c.WriteHeader(resp.Code)
	}
	var content []byte
	var err error
	content, err = json.Marshal(data)
	if err != nil {
		http.Error(c.ResponseWriter, err.Error(), http.StatusInternalServerError)
		return err
	}
	_, err = c.ResponseWriter.Write(content)
	return err
}

func (c *RespController) RespOk() error {
	data := common.JsonRespOk()
	return c.JSON(data)
}

func (c *RespController) RespOkMsg(msg string) error {
	data := common.JsonRespOk()
	data.Msg = msg
	return c.JSON(data)
}

func (c *RespController) RespOkData(data interface{}) error {
	return c.JSON(common.JsonRespOkData(data))
}

func (c *RespController) RespError(err error) error {
	resp := common.JsonRespError(err)
	return c.JSON(resp)
}

// param '%s' is not persent
func (c *RespController) RespErrorParam(key string) error {
	return c.RespError(fmt.Errorf("param '%s' is not persent", key))
}

func (c *RespController) RespErr(err *common.Err) error {
	resp := common.JsonRespErr(err)
	return c.JSON(resp)
}

func (c *RespController) Resp(resp common.JsonResp) error {
	return c.JSON(resp)
}

// 不是集群内部请求
func (c *RespController) IsNotClusterRequest() bool {
	header := c.Request.Header.Get(cluster.X_Cluster_Request)
	return !cluster.Enabled() || header != cluster.Token()
}

// 设置http响应states code
func (c *RespController) WriteHeader(statusCode int) {
	c.ResponseWriter.WriteHeader(statusCode)
}

// 设置ResponseHeader
func (c *RespController) HeaderSet(key, value string) {
	c.ResponseWriter.Header().Set(key, value)
}

func (c *RespController) StopRun() {
	panic(http.ErrAbortHandler)
}

func (c *RespController) FormFile(key string) (multipart.File, *multipart.FileHeader, error) {
	return c.Request.FormFile(key)
}

func (ctl *RespController) Download(file string, filename ...string) {
	// check get file error, file not found or other error.
	if _, err := os.Stat(file); err != nil {
		http.ServeFile(ctl.ResponseWriter, ctl.Request, file)
		return
	}

	var fName string
	if len(filename) > 0 && filename[0] != "" {
		fName = filename[0]
	} else {
		fName = filepath.Base(file)
	}
	// https://tools.ietf.org/html/rfc6266#section-4.3
	fn := url.PathEscape(fName)
	if fName == fn {
		fn = "filename=" + fn
	} else {
		/**
		  The parameters "filename" and "filename*" differ only in that
		  "filename*" uses the encoding defined in [RFC5987], allowing the use
		  of characters not present in the ISO-8859-1 character set
		  ([ISO-8859-1]).
		*/
		fn = "filename=" + fName + "; filename*=utf-8''" + fn
	}

	if strings.HasSuffix(fName, ".png") {
		ctl.ResponseWriter.Header().Add("Content-Type", "image/png")
	} else if strings.HasSuffix(fName, ".jpg") || strings.HasSuffix(fName, ".jpeg") {
		ctl.ResponseWriter.Header().Add("Content-Type", "image/jpg")
	} else if strings.HasSuffix(fName, ".gif") {
		ctl.ResponseWriter.Header().Add("Content-Type", "image/gif")
	} else {
		ctl.ResponseWriter.Header().Add("Content-Disposition", "attachment; "+fn)
		ctl.ResponseWriter.Header().Add("Content-Description", "File Transfer")
		ctl.ResponseWriter.Header().Add("Content-Type", "application/octet-stream")
		ctl.ResponseWriter.Header().Add("Content-Transfer-Encoding", "binary")
		ctl.ResponseWriter.Header().Add("Expires", "0")
		ctl.ResponseWriter.Header().Add("Cache-Control", "must-revalidate")
		ctl.ResponseWriter.Header().Add("Pragma", "public")
	}
	http.ServeFile(ctl.ResponseWriter, ctl.Request, file)
}

func (ctl *RespController) Redirect(path string) {
	http.RedirectHandler(path+"/", http.StatusMovedPermanently).ServeHTTP(ctl.ResponseWriter, ctl.Request)
}
