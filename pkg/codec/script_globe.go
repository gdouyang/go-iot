package codec

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"go-iot/pkg/util"
	"hash"
	"strings"

	"github.com/dop251/goja"
)

// js 全局对象，包含常用工具方法
type globe struct {
	vm *goja.Runtime `json:"-"`
}

func (g *globe) getCallStack() string {
	stacks := g.vm.CaptureCallStack(10, nil)
	sb := strings.Builder{}

	for _, v := range stacks {
		sb.WriteString(v.FuncName())
		sb.WriteString(v.Position().String())
		sb.WriteString("\n")
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
