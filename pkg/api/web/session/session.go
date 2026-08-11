package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go-iot/pkg/redis"
	"strconv"
	"time"

	logs "go-iot/pkg/logger"
)

const (
	KEY_PREFIX = "goiot:usersession:"
	// sessionIDBytes 会话 id 随机字节数（hex 后 64 字符）
	sessionIDBytes = 32
)

func getSessionId(key string) string {
	return KEY_PREFIX + key
}

// DefaultExpireSec 未指定登录 expires 时的默认 TTL（秒）。
const DefaultExpireSec = 60 * 60

// NewSession 创建会话。SessionId 使用 crypto/rand，不可从时间戳反推。
// expire 为秒；≤0 时使用 DefaultExpireSec。
func NewSession(expire int) *HttpSession {
	if expire <= 0 {
		expire = DefaultExpireSec
	}
	sessionId, err := randomSessionID()
	if err != nil {
		// 极罕见：退化为纳秒+再读一次随机，仍避免固定 md5(nanosecond)
		fallback := make([]byte, sessionIDBytes)
		_, _ = rand.Read(fallback)
		sessionId = hex.EncodeToString(fallback)
		if sessionId == "" {
			sessionId = fmt.Sprintf("%x%x", time.Now().UnixNano(), time.Now().UnixNano())
		}
		logs.Errorf("session id crypto/rand failed: %v, used fallback", err)
	}
	sesion := &HttpSession{Sessionid: sessionId, ExpireSec: expire}
	// 持久化 TTL 秒数，供后续 Get 滑动续期使用（与 Redis EXPIRE 一致）
	sesion.persistExpireSec()
	sesion.UpdateExpire()
	return sesion
}

func randomSessionID() (string, error) {
	b := make([]byte, sessionIDBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func Get(key string) *HttpSession {
	if len(key) == 0 {
		return nil
	}
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
	if sesion.ExpireSec <= 0 {
		sesion.ExpireSec = DefaultExpireSec
	}
	// 滑动续期：重置 Redis TTL
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

// TTL 返回会话有效时长（秒），至少 DefaultExpireSec。
func (s *HttpSession) TTL() int {
	if s == nil || s.ExpireSec <= 0 {
		return DefaultExpireSec
	}
	return s.ExpireSec
}

func (s *HttpSession) persistExpireSec() {
	if s == nil {
		return
	}
	client := redis.GetRedisClient()
	// 存原始秒数字符串，避免 JSON 数字类型歧义
	client.HSet(context.Background(), s.getSessionId(), "expire", strconv.Itoa(s.TTL()))
}

// UpdateExpire 将 Redis key 的 TTL 重置为 ExpireSec（滑动续期）。
func (s *HttpSession) UpdateExpire() {
	if s == nil {
		return
	}
	client := redis.GetRedisClient()
	ttl := time.Duration(s.TTL()) * time.Second
	client.Expire(context.Background(), s.getSessionId(), ttl)
}

func (s *HttpSession) SetPermission(p map[string]bool) {
	s.SetAttribute("permissions", p)
}

func (s *HttpSession) GetPermission() map[string]bool {
	permission := map[string]bool{}
	s.GetObject("permissions", &permission)
	return permission
}
