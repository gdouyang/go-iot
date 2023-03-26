package session

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"go-iot/pkg/core/redis"
	"time"

	"github.com/beego/beego/v2/core/logs"
)

const (
	KEY_PREFIX = "goiot-usersession:"
)

func getSessionId(key string) string {
	return KEY_PREFIX + key
}
func NewSession() *HttpSession {
	sesion := &HttpSession{}
	val := fmt.Sprintf("%d", time.Now().Nanosecond())
	data := []byte(val)
	has := md5.Sum(data)
	sesion.Sessionid = fmt.Sprintf("%x", has) //将[]byte转成16进制
	Put(sesion)
	return sesion
}

func Get(key string) *HttpSession {
	client := redis.GetRedisClient()
	sessionid := getSessionId(key)
	data, err := client.Exists(context.Background(), sessionid).Result()
	if err != nil {
		logs.Error(err)
		return nil
	}
	if data < 1 {
		return nil
	}
	sesion := &HttpSession{Sessionid: key}
	client.Expire(context.Background(), sessionid, time.Duration(1)*time.Hour)
	return sesion
}

func Put(session *HttpSession) {
	client := redis.GetRedisClient()
	client.HSet(context.Background(), session.getSessionId(), map[string]string{})
	client.Expire(context.Background(), session.getSessionId(), time.Duration(1)*time.Hour)
}

func Del(key string) {
	client := redis.GetRedisClient()
	client.Del(context.Background(), getSessionId(key))
}

type HttpSession struct {
	Sessionid string
}

func (s *HttpSession) getSessionId() string {
	return getSessionId(s.Sessionid)
}

func (s *HttpSession) Get(key string, data interface{}) bool {
	client := redis.GetRedisClient()
	v, err := client.HGet(context.Background(), s.getSessionId(), key).Result()
	if err == nil && len(v) > 0 {
		json.Unmarshal([]byte(v), data)
		return true
	}
	return false
}

func (s *HttpSession) Put(key string, value interface{}) {
	client := redis.GetRedisClient()
	data, err := json.Marshal(value)
	if err != nil {
		logs.Error(err)
	}
	client.HSet(context.Background(), s.getSessionId(), key, string(data))
}

func (s *HttpSession) SetPermission(p map[string]bool) {
	s.Put("permissions", p)
}

func (s *HttpSession) GetPermission() map[string]bool {
	permission := map[string]bool{}
	s.Get("permissions", &permission)
	return permission
}
