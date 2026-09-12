// redis相关配置与方法
package redis

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	logs "go-iot/pkg/logger"
	"go-iot/pkg/option"

	"github.com/go-redis/redis/v8"
)

// the config of redis
type RedisConfig struct {
	Addr     string
	Addrs    []string
	Password string
	DB       int
	PoolSize int
}

type ZRangeBy = redis.ZRangeBy
type Z = redis.Z

// Script/NewScript 透传 go-redis 脚本支持（Lua 原子操作用）
type Script = redis.Script

var NewScript = redis.NewScript

func (r RedisConfig) String() string {
	if len(r.Addrs) > 1 {
		return fmt.Sprintf("[cluster addrs=%v, poolSize=%v]", r.Addrs, r.PoolSize)
	}
	return fmt.Sprintf("[addr=%s, db=%v, poolSize=%v]", r.Addr, r.DB, r.PoolSize)
}

var DefaultRedisConfig RedisConfig = RedisConfig{
	Addr:     "127.0.0:6379",
	PoolSize: 100,
}

// config redis
func Config(opt *option.Options) {
	DefaultRedisConfig.Addr = opt.Redis.Addr
	// addr 逗号分隔多地址时自动启用 Cluster 模式（多节点）；单地址保持单实例
	DefaultRedisConfig.Addrs = splitAddr(opt.Redis.Addr)
	DefaultRedisConfig.Password = opt.Redis.Password
	DefaultRedisConfig.DB = opt.Redis.Db
	// 仅在配置了有效值时覆盖默认连接池大小；≤0 保持默认(100)，
	// 大并发设备上线场景每个上线都要查离线命令队列，池子太小会 connection pool timeout
	if opt.Redis.PoolSize > 0 {
		DefaultRedisConfig.PoolSize = opt.Redis.PoolSize
	}
	// Cluster 模式不支持 select db，配置了 db 时提示
	if len(DefaultRedisConfig.Addrs) > 1 && opt.Redis.Db != 0 {
		logs.Warnf("redis cluster mode does not support db selection, db=%d ignored", opt.Redis.Db)
	}
	logs.Infof("redis config: %v", DefaultRedisConfig)
	InitRedis()
}

// splitAddr 逗号分隔多节点地址（addr: "a:6379,b:6379" → 集群模式）
func splitAddr(addr string) []string {
	parts := make([]string, 0)
	for _, p := range strings.Split(addr, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}

var rdb redis.UniversalClient

const Nil = redis.Nil

// init redis client, panic can't connect to the server
func InitRedis() {
	if rdb == nil {
		var mutex sync.Mutex
		mutex.Lock()
		defer mutex.Unlock()
		for {
			if len(DefaultRedisConfig.Addrs) > 1 {
				// addr 逗号分隔多节点 → Redis Cluster 模式（水平扩展/高可用）
				rdb = redis.NewClusterClient(&redis.ClusterOptions{
					Addrs:    DefaultRedisConfig.Addrs,
					Password: DefaultRedisConfig.Password,
					PoolSize: DefaultRedisConfig.PoolSize,
				})
			} else {
				// 单实例模式（默认）：单地址或未配置
				addr := DefaultRedisConfig.Addr
				if len(DefaultRedisConfig.Addrs) == 1 {
					addr = DefaultRedisConfig.Addrs[0]
				}
				rdb = redis.NewClient(&redis.Options{
					Addr:     addr,
					Password: DefaultRedisConfig.Password,
					DB:       DefaultRedisConfig.DB,
					PoolSize: DefaultRedisConfig.PoolSize,
				})
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
			err := rdb.Ping(ctx).Err()
			cancel()
			if err == nil {
				break // 连接成功，退出循环
			}
			logs.Errorf("redis error: %v", err)
			time.Sleep(5 * time.Second) // 等待5秒后重试
		}
	}
}

func GetRedisClient() redis.UniversalClient {
	return rdb
}

// SetClientForTest 注入 Redis 客户端（单测用 miniredis 等）。返回还原函数。
// 生产代码请使用 Config / InitRedis，勿调用本函数。
func SetClientForTest(c redis.UniversalClient) (restore func()) {
	old := rdb
	rdb = c
	return func() { rdb = old }
}

func Sub(channels ...string) <-chan *redis.Message {
	client := GetRedisClient()
	sub := client.Subscribe(client.Context(), channels...)
	return sub.Channel()
}

func Pub(channel string, message interface{}) {
	client := GetRedisClient()
	client.Publish(client.Context(), channel, message).Result()
}

// ==================== Redis 分布式锁实现（支持 Redis Cluster / 单实例） ====================

var (
	// ErrLockNotHeld 锁未被当前持有者持有（可能已过期或被其他客户端获取）
	ErrLockNotHeld = errors.New("redis lock: lock not held or expired")
	// ErrLockHeld 锁已被其他客户端持有
	ErrLockHeld = errors.New("redis lock: lock already held")
)

var (
	// unlockScript 原子安全释放锁：仅当 key 对应的 value 等于期望的 token 时才执行 del。
	// 单 Key (KEYS[1]) 操作，天然兼容 Redis Cluster 模式（无 cross-slot 限制）。
	unlockScript = redis.NewScript(`
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`)

	// refreshScript 原子安全续期：仅当 key 对应的 value 等于期望的 token 时才重置过期时间。
	// 单 Key (KEYS[1]) 操作，天然兼容 Redis Cluster 模式。
	refreshScript = redis.NewScript(`
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("pexpire", KEYS[1], ARGV[2])
		else
			return 0
		end
	`)
)

const (
	defaultLockTTL   = 10 * time.Second
	defaultLockRetry = 50 * time.Millisecond
)

// LockOptions 锁的可选配置
type LockOptions struct {
	TTL           time.Duration         // 锁过期时间（默认 10s）
	RetryInterval time.Duration         // 阻塞加锁等待重试间隔（默认 50ms）
	Token         string                // 锁持有者唯一标识（若为空则自动生成随机 token）
	AutoRefresh   bool                  // 是否启用后台看门狗自动续期
	Client        redis.UniversalClient // 自定义 Redis 客户端（默认使用全局 rdb）
}

// LockOption 锁配置函数
type LockOption func(*LockOptions)

// WithTTL 指定锁的过期时间
func WithTTL(d time.Duration) LockOption {
	return func(o *LockOptions) {
		if d > 0 {
			o.TTL = d
		}
	}
}

// WithRetryInterval 指定阻塞加锁时的重试间隔
func WithRetryInterval(d time.Duration) LockOption {
	return func(o *LockOptions) {
		if d > 0 {
			o.RetryInterval = d
		}
	}
}

// WithToken 指定自定义的 token 标识
func WithToken(token string) LockOption {
	return func(o *LockOptions) {
		if token != "" {
			o.Token = token
		}
	}
}

// WithAutoRefresh 启用/关闭看门狗自动续期
func WithAutoRefresh(enabled bool) LockOption {
	return func(o *LockOptions) {
		o.AutoRefresh = enabled
	}
}

// WithClient 指定自定义的 Redis 客户端实例
func WithClient(client redis.UniversalClient) LockOption {
	return func(o *LockOptions) {
		o.Client = client
	}
}

// Lock Redis 分布式互斥锁，支持单实例模式与 Redis Cluster 集群模式。
type Lock struct {
	client      redis.UniversalClient
	key         string
	token       string
	ttl         time.Duration
	retry       time.Duration
	autoRefresh bool

	mu        sync.Mutex
	stopWD    chan struct{}
	wdRunning bool
}

// NewLock 创建一个新的分布式锁实例
func NewLock(key string, opts ...LockOption) *Lock {
	cfg := LockOptions{
		TTL:           defaultLockTTL,
		RetryInterval: defaultLockRetry,
		AutoRefresh:   false,
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.Token == "" {
		cfg.Token = newLockToken()
	}
	return &Lock{
		client:      cfg.Client,
		key:         key,
		token:       cfg.Token,
		ttl:         cfg.TTL,
		retry:       cfg.RetryInterval,
		autoRefresh: cfg.AutoRefresh,
	}
}

func (l *Lock) getClient() redis.UniversalClient {
	if l.client != nil {
		return l.client
	}
	return GetRedisClient()
}

// Key 返回锁对应的 Redis Key
func (l *Lock) Key() string {
	return l.key
}

// Token 返回当前锁持有者的唯一 Token
func (l *Lock) Token() string {
	return l.token
}

// TTL 返回锁配置的过期时间
func (l *Lock) TTL() time.Duration {
	return l.ttl
}

// TryLock 尝试获取锁一次（非阻塞）。
// 成功返回 true, nil；若锁已被他人占用返回 false, nil；若发生网络或 Redis 错误返回 false, err。
func (l *Lock) TryLock(ctx context.Context) (bool, error) {
	client := l.getClient()
	if client == nil {
		return false, errors.New("redis lock: redis client is nil, call InitRedis first")
	}
	if l.key == "" {
		return false, errors.New("redis lock: key cannot be empty")
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	ok, err := client.SetNX(ctx, l.key, l.token, l.ttl).Result()
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}

	if l.autoRefresh {
		l.startWatchdogLocked()
	}
	return true, nil
}

// Lock 阻塞尝试加锁，直到获取成功或 ctx 取消/超时。
func (l *Lock) Lock(ctx context.Context) error {
	timer := time.NewTimer(0)
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
	defer timer.Stop()

	for {
		ok, err := l.TryLock(ctx)
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
		timer.Reset(l.retry)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}
	}
}

// Unlock 释放当前持有的分布式锁。
// 使用 Lua 脚本原子比较 token，确保仅能释放自己获取的锁，防止因超时被其他客户端重新持有时误删。
// 若锁不存在或已被他人持有，返回 ErrLockNotHeld。
func (l *Lock) Unlock(ctx context.Context) error {
	client := l.getClient()
	if client == nil {
		return errors.New("redis lock: redis client is nil")
	}

	l.mu.Lock()
	l.stopWatchdogLocked()
	l.mu.Unlock()

	val, err := unlockScript.Run(ctx, client, []string{l.key}, l.token).Result()
	if err != nil {
		return err
	}
	res, _ := val.(int64)
	if res == 0 {
		return ErrLockNotHeld
	}
	return nil
}

// Refresh 手动延长锁的过期时间为初始配置的 TTL。
// 仅当当前锁依然被自身持有时才能续期成功，否则返回 ErrLockNotHeld。
func (l *Lock) Refresh(ctx context.Context) error {
	client := l.getClient()
	if client == nil {
		return errors.New("redis lock: redis client is nil")
	}

	val, err := refreshScript.Run(ctx, client, []string{l.key}, l.token, l.ttl.Milliseconds()).Result()
	if err != nil {
		return err
	}
	res, _ := val.(int64)
	if res == 0 {
		return ErrLockNotHeld
	}
	return nil
}

func (l *Lock) startWatchdogLocked() {
	if l.wdRunning {
		return
	}
	l.wdRunning = true
	stopCh := make(chan struct{})
	l.stopWD = stopCh

	interval := l.ttl / 3
	if interval < 100*time.Millisecond {
		interval = 100 * time.Millisecond
	}

	key := l.key
	token := l.token
	ttlMs := l.ttl.Milliseconds()
	client := l.getClient()

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-stopCh:
				return
			case <-ticker.C:
				rfCtx, rfCancel := context.WithTimeout(context.Background(), interval)
				val, err := refreshScript.Run(rfCtx, client, []string{key}, token, ttlMs).Result()
				rfCancel()
				if err != nil || val != int64(1) {
					// 续约失败或锁已丢失，自动停止看门狗
					return
				}
			}
		}
	}()
}

func (l *Lock) stopWatchdogLocked() {
	if l.wdRunning && l.stopWD != nil {
		close(l.stopWD)
		l.wdRunning = false
		l.stopWD = nil
	}
}

func newLockToken() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%d-%d", time.Now().UnixNano(), time.Now().Unix())
	}
	return hex.EncodeToString(b[:])
}

// TryAcquireLock 快捷方法：尝试非阻塞获取锁一次。
// 成功返回获取到的 *Lock 实例，若已被占用返回 (nil, false, nil)。
func TryAcquireLock(ctx context.Context, key string, ttl time.Duration, opts ...LockOption) (*Lock, bool, error) {
	allOpts := append([]LockOption{WithTTL(ttl)}, opts...)
	l := NewLock(key, allOpts...)
	ok, err := l.TryLock(ctx)
	if err != nil || !ok {
		return nil, ok, err
	}
	return l, true, nil
}

// AcquireLock 快捷方法：阻塞加锁直到成功或 ctx 取消/超时。
// 成功返回获取到的 *Lock 实例。
func AcquireLock(ctx context.Context, key string, ttl time.Duration, opts ...LockOption) (*Lock, error) {
	allOpts := append([]LockOption{WithTTL(ttl)}, opts...)
	l := NewLock(key, allOpts...)
	if err := l.Lock(ctx); err != nil {
		return nil, err
	}
	return l, nil
}

