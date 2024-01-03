package core

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"go-iot/pkg/core/util"
	"hash"
)

// js 全局对象，包含常用工具方法
type globe struct {
}

// crc16
func (g globe) ToCrc16Str(str string) string {
	d := util.ToCrc16Str(str)
	return d
}

// 将字节数组转换为 Base64 字符串
func (g globe) BytesToBase64(bytes []byte) string {
	signature := base64.StdEncoding.EncodeToString(bytes)
	return signature
}

func (g globe) HmacEncryptBase64(data, key, signatureMethod string) string {
	v := g.HmacEncrypt(data, key, signatureMethod)
	return g.BytesToBase64(v)
}

// signatureMethod支持sha1, sha256, md5
func (g globe) HmacEncrypt(data, key, signatureMethod string) []byte {
	// 解码 Base64 编码的密钥
	signinKey, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		panic(err)
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
		panic("unsupported signatureMethod: " + signatureMethod)
	}

	// 更新 Hmac 实例的数据
	hmacInstance.Write([]byte(data))
	// 返回加密结果的字节数组
	return hmacInstance.Sum(nil)
}
