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
	"time"

	"github.com/dop251/goja"
)

// js 全局对象，包含常用工具方法
type globe struct {
	vm        *goja.Runtime `json:"-"`
	productId string        `json:"-"`
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

// crc16
func (g *globe) ToCrc16Str(str string) string {
	d, err := util.ToCrc16Str(str)
	if err != nil {
		panic(fmt.Errorf("%v %s", err, g.getCallStack()))
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
		panic(g.vm.ToValue(err))
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
		panic(g.vm.ToValue(fmt.Errorf("unsupported signatureMethod: %s %s", signatureMethod, g.getCallStack())))
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
	result := httpResp{}
	result.setStatus(400)
	path := config["url"]
	u, err := url.ParseRequestURI(fmt.Sprintf("%v", path))
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
			core.DebugLog("", g.productId, fmt.Sprintf("headers is not object: %v", v))
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
				core.DebugLog("", g.productId, fmt.Sprintf("data parse error: %v", err))
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
	go func() {
		defer func() {
			if rec := recover(); rec != nil {
				l := fmt.Sprintf("productId: [%s] error: %v", g.productId, rec)
				logger.Errorf(l)
				core.DebugLog("", g.productId, l)
			}
		}()
		resp := g.HttpRequest(config)
		if v, ok := config["complete"]; ok {
			fn, success := goja.AssertFunction(g.vm.ToValue(v))
			if success {
				fn(goja.Undefined(), g.vm.ToValue(resp))
			} else {
				core.DebugLog("", g.productId, "HttpRequestAsync complete is not a function")
			}
		}
	}()
}
