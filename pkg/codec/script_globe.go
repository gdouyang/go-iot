package codec

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"go-iot/pkg/core"
	"go-iot/pkg/logger"
	"go-iot/pkg/util"
	"hash"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dop251/goja"
)

// js 全局对象，包含常用工具方法
type globe struct {
	vm         *goja.Runtime `json:"-"`
	productId  string        `json:"-"`
	deviceId   string        `json:"-"`
	deviceIdFn func() string `json:"-"`
	// mu 串行同一 VM 上的脚本执行：FuncInvoke 与 HttpRequestAsync complete
	// 不能并行进入 goja（Runtime 非线程安全），也避免 complete 读到下一轮 deviceId。
	mu sync.Mutex `json:"-"`
}

func (g *globe) currentDeviceId() string {
	if g == nil {
		return ""
	}
	if g.deviceIdFn != nil {
		if id := g.deviceIdFn(); id != "" {
			return id
		}
	}
	return g.deviceId
}

func (g *globe) bindDevice(param interface{}) func() {
	if g == nil {
		return func() {}
	}
	prev := g.deviceId
	prevFn := g.deviceIdFn
	g.deviceId = extractDeviceId(param)
	g.deviceIdFn = func() string { return extractDeviceId(param) }
	return func() {
		g.deviceId = prev
		g.deviceIdFn = prevFn
	}
}

func (g *globe) withDeviceId(deviceId string) func() {
	if g == nil {
		return func() {}
	}
	prev := g.deviceId
	prevFn := g.deviceIdFn
	g.deviceId = deviceId
	if deviceId == "" {
		g.deviceIdFn = nil
	} else {
		id := deviceId
		g.deviceIdFn = func() string { return id }
	}
	return func() {
		g.deviceId = prev
		g.deviceIdFn = prevFn
	}
}

func (g *globe) getCallStack() string {
	stacks := g.vm.CaptureCallStack(10, nil)
	sb := strings.Builder{}

	for _, v := range stacks {
		if v.Position().Line > 0 {
			sb.WriteString(v.FuncName())
			sb.WriteString(v.Position().String())
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

// throw 将 Go error 转为 JS 异常（goja 约定：宿主 panic NewGoError ≈ JS throw）。
// 仅用于脚本可调用的宿主 API，不要用于业务/IO 主路径。
func (g *globe) throw(err error) {
	if err == nil {
		return
	}
	panic(g.vm.NewGoError(fmt.Errorf("%w\n%s", err, g.getCallStack())))
}

// crc16
func (g *globe) ToCrc16Str(str string) string {
	d, err := util.ToCrc16Str(str)
	if err != nil {
		g.throw(err)
	}
	return d
}

// 将字节数组转换为 Base64 字符串
func (g *globe) BytesToBase64(bytes []byte) string {
	signature := base64.StdEncoding.EncodeToString(bytes)
	return signature
}

func (g *globe) HmacEncryptBase64(data, key, signatureMethod string) string {
	v := g.HmacEncrypt(data, key, signatureMethod)
	return g.BytesToBase64(v)
}

// signatureMethod支持sha1, sha256, md5
func (g *globe) HmacEncrypt(data, key, signatureMethod string) []byte {
	// 解码 Base64 编码的密钥
	signinKey, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		g.throw(err)
	}

	// 创建 Hmac 实例，指定签名算法和密钥
	var hmacInstance hash.Hash
	if signatureMethod == "sha1" {
		hmacInstance = hmac.New(sha1.New, signinKey)
	} else if signatureMethod == "sha256" {
		hmacInstance = hmac.New(sha256.New, signinKey)
	} else if signatureMethod == "md5" {
		hmacInstance = hmac.New(md5.New, signinKey)
	} else {
		g.throw(fmt.Errorf("unsupported signatureMethod: %s", signatureMethod))
	}

	// 更新 Hmac 实例的数据
	hmacInstance.Write([]byte(data))
	// 返回加密结果的字节数组
	return hmacInstance.Sum(nil)
}

type httpResp map[string]any

func (h httpResp) setStatus(code int) {
	h["status"] = code
}

func (h httpResp) setMessage(msg string) {
	h["message"] = msg
}

func (h httpResp) setData(msg string) {
	h["data"] = msg
}

func (h httpResp) setHeader(header map[string]string) {
	h["header"] = header
}

// http请求，使编解码脚本有发送http的能力
func (g *globe) HttpRequest(config map[string]any) map[string]any {
	return g.doHttpRequest(config, g.currentDeviceId())
}

func (g *globe) doHttpRequest(config map[string]any, deviceId string) map[string]any {
	result := httpResp{}
	result.setStatus(400)
	path := config["url"]
	urlStr := fmt.Sprintf("%v", path)
	if err := checkScriptHTTPAllowed(urlStr); err != nil {
		logger.Warnf("script HttpRequest denied: %v", err)
		core.DebugLog("warn", deviceId, g.productId, fmt.Sprintf("HttpRequest denied: %v", err))
		result.setMessage(err.Error())
		return result
	}
	u, err := url.ParseRequestURI(urlStr)
	if err != nil {
		logger.Errorf(err.Error())
		result.setMessage(err.Error())
		return result
	}
	method := strings.ToUpper(fmt.Sprintf("%v", config["method"]))
	timeout := time.Second * 3
	if v, ok := config["timeout"]; ok {
		seconds, err := strconv.Atoi(fmt.Sprintf("%v", v))
		if err == nil {
			timeout = time.Second * time.Duration(seconds)
		}
	}
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.DisableKeepAlives = true
	client := http.Client{
		Transport: t,
		Timeout:   timeout,
	}
	var req *http.Request = &http.Request{
		Method: method,
		URL:    u,
		Header: map[string][]string{},
	}
	if method == "POST" || method == "PUT" {
		req.Header.Add("Content-Type", "application/json; charset=utf-8")
	}
	if v, ok := config["headers"]; ok {
		h, ok := v.(map[string]any)
		if !ok {
			logger.Warnf("headers is not object: %v", v)
			core.DebugLog("warn", deviceId, g.productId, fmt.Sprintf("headers is not object: %v", v))
			h = map[string]any{}
		}
		for key, value := range h {
			req.Header.Add(key, fmt.Sprintf("%v", value))
		}
	}
	if data, ok := config["data"]; ok {
		if body, ok := data.(map[string]any); ok {
			b, err := json.Marshal(body)
			if err != nil {
				logger.Errorf("data parse error: %v", err)
				core.DebugLog("error", deviceId, g.productId, fmt.Sprintf("data parse error: %v", err))
				result.setMessage(err.Error())
				return result
			}
			req.Body = io.NopCloser(strings.NewReader(string(b)))
		} else {
			req.Body = io.NopCloser(strings.NewReader(fmt.Sprintf("%v", data)))
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		logger.Warnf(err.Error())
		result.setMessage(err.Error())
		return result
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Warnf(err.Error())
		result.setMessage(err.Error())
		return result
	}
	header := map[string]string{}
	if resp.Header != nil {
		for key := range resp.Header {
			header[key] = resp.Header.Get(key)
		}
	}
	result.setStatus(resp.StatusCode)
	result.setData(string(b))
	result.setHeader(header)
	if resp.StatusCode >= 300 {
		result.setMessage(string(b))
	}
	return result
}

// http请求异步
func (g *globe) HttpRequestAsync(config map[string]interface{}) {
	deviceId := g.currentDeviceId()
	go func() {
		defer func() {
			if rec := recover(); rec != nil {
				l := fmt.Sprintf("productId: [%s] error: %v", g.productId, rec)
				logger.Errorf(l)
				core.DebugLog("error", deviceId, g.productId, l)
			}
		}()
		resp := g.doHttpRequest(config, deviceId)
		v, ok := config["complete"]
		if !ok {
			return
		}
		// complete 必须在创建它的原 VM 上跑，并与 FuncInvoke 互斥。
		g.mu.Lock()
		defer g.mu.Unlock()
		restore := g.withDeviceId(deviceId)
		defer restore()
		fn, success := goja.AssertFunction(g.vm.ToValue(v))
		if success {
			fn(goja.Undefined(), g.vm.ToValue(resp))
		} else {
			core.DebugLog("warn", deviceId, g.productId, "HttpRequestAsync complete is not a function")
		}
	}()
}
