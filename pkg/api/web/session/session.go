package session

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"go-iot/pkg/redis"
	"strconv"
	"time"

	logs "go-iot/pkg/logger"
)

const (
	KEY_PREFIX = "goiot:usersession:"
)

func getSessionId(key string) string {
	return KEY_PREFIX + key
}

// expire sec
func NewSession(expire int) *HttpSession {
	val := fmt.Sprintf("%d", time.Now().Nanosecond())
	data := []byte(val)
	has := md5.Sum(data)
	//将[]byte转成16进制
	sessionId := fmt.Sprintf("%x", has)
	sesion := &HttpSession{Sessionid: sessionId, ExpireSec: expire}
	sesion.UpdateExpire()
	return sesion
}

func Get(key string) *HttpSession {
	client := redis.GetRedisClient()
	sessionid := getSessionId(key)
	data, err := client.Exists(context.Background(), sessionid).Result()
	if err != nil {
		logs.Errorf("get http session key: %s, error: %v", key, err)
		return nil
	}
	if data < 1 {
		return nil
	}
	sesion := &HttpSession{Sessionid: key}
	str := sesion.GetPrimitive("expire")
	if len(str) > 0 {
		expire, _ := strconv.Atoi(str)
		sesion.ExpireSec = expire
	}
	sesion.UpdateExpire()
	return sesion
}

func Del(key string) {
	client := redis.GetRedisClient()
	client.Del(context.Background(), getSessionId(key))
}

type HttpSession struct {
	Sessionid string
	ExpireSec int // 过期时间秒
}

func (s *HttpSession) getSessionId() string {
	return getSessionId(s.Sessionid)
}

// 获取对象类型数据
func (s *HttpSession) GetObject(key string, data interface{}) bool {
	client := redis.GetRedisClient()
	v, err := client.HGet(context.Background(), s.getSessionId(), key).Result()
	if err == nil && len(v) > 0 {
		json.Unmarshal([]byte(v), data)
		return true
	}
	return false
}

// 获取原始类型数据
func (s *HttpSession) GetPrimitive(key string) string {
	client := redis.GetRedisClient()
	v, err := client.HGet(context.Background(), s.getSessionId(), key).Result()
	if err == nil && len(v) > 0 {
		return string(v)
	}
	return ""
}

func (s *HttpSession) SetAttribute(key string, value interface{}) {
	client := redis.GetRedisClient()
	data, err := json.Marshal(value)
	if err != nil {
		logs.Errorf("put http session key: %s, value: %v, error: %v", key, value, err)
	}
	client.HSet(context.Background(), s.getSessionId(), key, string(data))
}

func (s *HttpSession) UpdateExpire() {
	client := redis.GetRedisClient()
	expire := time.Duration(1) * time.Hour
	if s.ExpireSec > 0 {
		expire = time.Duration(s.ExpireSec) * time.Second
	}
	client.Expire(context.Background(), s.getSessionId(), expire)
}

func (s *HttpSession) SetPermission(p map[string]bool) {
	s.SetAttribute("permissions", p)
}

func (s *HttpSession) GetPermission() map[string]bool {
	permission := map[string]bool{}
	s.GetObject("permissions", &permission)
	return permission
}
