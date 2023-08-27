package web

import (
	"encoding/json"
	"fmt"
	"go-iot/pkg/api/web/session"
	"go-iot/pkg/cluster"
	"go-iot/pkg/core/common"
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

func (c *RespController) Init(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	c.Request = r
	c.ResponseWriter = w
}

func (c *RespController) Prepare() {
}

// expire sec default 1 hour
func (c *RespController) NewSession(expire int) *session.HttpSession {
	if expire <= 0 {
		expire = 60 * 60
	}
	session := session.NewSession(expire)
	c.Request.Header.Add("x-access-token", session.Sessionid)
	gsessionid := fmt.Sprintf("gsessionid=%s; Expires=%s; Max-Age=%d", session.Sessionid, time.Now().Add(time.Duration(expire)*time.Second).UTC().Format(time.RFC1123), expire)
	c.ResponseWriter.Header().Add("Set-Cookie", gsessionid)
	return session
}

func (c *RespController) GetSession() *session.HttpSession {
	gsessionid := c.Request.Header.Get("x-access-token")
	if len(gsessionid) == 0 {
		cookie, _ := c.Request.Cookie("gsessionid")
		if cookie != nil {
			gsessionid = cookie.Value
		}
	}
	s := session.Get(gsessionid)
	return s
}

// return request path value
func (c *RespController) Param(key string) string {
	return chi.URLParam(c.Request, key)
}

// request param
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

func (c *RespController) WriteHeader(statusCode int) {
	c.ResponseWriter.WriteHeader(statusCode)
}

func (c *RespController) StopRun() {
	panic(http.ErrAbortHandler)
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
