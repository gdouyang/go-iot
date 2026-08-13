package timeseries

import (
	"strings"
	"sync"
	"sync/atomic"
	"time"

	logs "go-iot/pkg/logger"
)

const (
	// 默认值；运行时可由 ConfigTdengine 覆盖 bulk / flush
	tdBatchChanSizeDefault  = 20000
	tdBulkSizeDefault       = 500
	tdFlushIntervalDefault  = time.Second
)

// tdBatchSaver 将多条 INSERT 片段合并为一次 REST 提交。
// SQL 形态：
//
//	INSERT INTO
//	  child1 USING stable TAGS(...) (cols) VALUES (...),
//	  child2 USING stable TAGS(...) (cols) VALUES (...);
type tdBatchSaver struct {
	ch             chan string
	buf            []string
	lastCommitTime int64
	mu             sync.Mutex // 保护 flush 与测试同步
	bulkSize       int
	flushInterval  time.Duration
	// flushHook 测试可注入
	insertFn func(sql string) error
}

var defaultTdBatch = &tdBatchSaver{
	ch:             make(chan string, tdBatchChanSizeDefault),
	buf:            make([]string, 0, tdBulkSizeDefault),
	lastCommitTime: time.Now().UnixMilli(),
	bulkSize:       tdBulkSizeDefault,
	flushInterval:  tdFlushIntervalDefault,
}

func (s *tdBatchSaver) applyConfig(bulkSize int, flushInterval time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if bulkSize > 0 {
		s.bulkSize = bulkSize
	}
	if flushInterval > 0 {
		s.flushInterval = flushInterval
	}
}

// 保证 createTime 毫秒全局递增，降低同设备并发写冲突
var tdLastCreateMs atomic.Int64

func init() {
	go defaultTdBatch.loop()
}

// nextTdCreateTime 生成单调递增的 createTime 字符串。
func nextTdCreateTime() string {
	now := time.Now().UnixMilli()
	for {
		prev := tdLastCreateMs.Load()
		next := now
		if next <= prev {
			next = prev + 1
		}
		if tdLastCreateMs.CompareAndSwap(prev, next) {
			return time.UnixMilli(next).In(time.Local).Format(timeformt)
		}
	}
}

// FlushTdengineBatch 立刻刷出缓冲（单测 / 优雅停机可用）。
func FlushTdengineBatch() {
	// 先把 channel 中已入队片段并入 buf，再 flush；再短暂等待收口 loop 竞态
	for round := 0; round < 3; round++ {
		for {
			select {
			case frag := <-defaultTdBatch.ch:
				defaultTdBatch.mu.Lock()
				defaultTdBatch.buf = append(defaultTdBatch.buf, frag)
				defaultTdBatch.mu.Unlock()
			default:
				goto flushed
			}
		}
	flushed:
		defaultTdBatch.flush()
		if round < 2 {
			time.Sleep(15 * time.Millisecond)
		}
	}
}

func (s *tdBatchSaver) commit(fragment string) {
	if fragment == "" {
		return
	}
	select {
	case s.ch <- fragment:
	default:
		logs.Warnf("tdengine batch chan full (cap=%d), drop 1 row", cap(s.ch))
	}
}

func (s *tdBatchSaver) loop() {
	// 用较短 tick 轮询配置中的 flush 间隔
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.mu.Lock()
			intervalMs := s.flushInterval.Milliseconds()
			now := time.Now().UnixMilli()
			need := len(s.buf) > 0 && now-s.lastCommitTime >= intervalMs
			s.mu.Unlock()
			if need {
				s.flush()
			}
		case frag := <-s.ch:
			s.mu.Lock()
			s.buf = append(s.buf, frag)
			n := len(s.buf)
			bulk := s.bulkSize
			if bulk <= 0 {
				bulk = tdBulkSizeDefault
			}
			intervalMs := s.flushInterval.Milliseconds()
			now := time.Now().UnixMilli()
			need := n >= bulk || now-s.lastCommitTime >= intervalMs
			s.mu.Unlock()
			if need {
				s.flush()
			}
		}
	}
}

func (s *tdBatchSaver) flush() {
	s.mu.Lock()
	if len(s.buf) == 0 {
		s.mu.Unlock()
		return
	}
	// 复制并清空
	batch := make([]string, len(s.buf))
	copy(batch, s.buf)
	s.buf = s.buf[:0]
	s.lastCommitTime = time.Now().UnixMilli()
	s.mu.Unlock()

	bulk := s.bulkSize
	if bulk <= 0 {
		bulk = tdBulkSizeDefault
	}
	// 按 USING stable 分组，避免不同 super table 混在一条 INSERT 里出问题
	groups := groupFragmentsByStable(batch)
	for _, frags := range groups {
		for i := 0; i < len(frags); i += bulk {
			end := i + bulk
			if end > len(frags) {
				end = len(frags)
			}
			sql := buildMultiInsertSQL(frags[i:end])
			start := time.Now()
			var err error
			if s.insertFn != nil {
				err = s.insertFn(sql)
			} else {
				err = defaultTdInsert(sql)
			}
			cost := time.Since(start)
			if err != nil {
				logs.Errorf("tdengine batch insert error (rows=%d cost=%s): %v", end-i, cost, err)
				continue
			}
			if cost > time.Second {
				logs.Warnf("tdengine batch insert slow: rows=%d cost=%s", end-i, cost)
			} else {
				logs.Debugf("tdengine batch insert ok: rows=%d cost=%s", end-i, cost)
			}
		}
	}
}

// groupFragmentsByStable 按 "USING <stable>" 分组。
func groupFragmentsByStable(frags []string) map[string][]string {
	out := make(map[string][]string)
	for _, f := range frags {
		key := "_default"
		if i := strings.Index(f, " USING "); i >= 0 {
			rest := f[i+len(" USING "):]
			if j := strings.Index(rest, " "); j >= 0 {
				key = rest[:j]
			} else {
				key = rest
			}
		}
		out[key] = append(out[key], f)
	}
	return out
}

func buildMultiInsertSQL(fragments []string) string {
	if len(fragments) == 0 {
		return ""
	}
	if len(fragments) == 1 {
		return "INSERT INTO " + fragments[0] + ";"
	}
	var sb strings.Builder
	// 预估容量
	sb.Grow(32 + len(fragments)*64)
	sb.WriteString("INSERT INTO ")
	for i, f := range fragments {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(f)
	}
	sb.WriteByte(';')
	return sb.String()
}

// defaultTdInsert 走共享 HTTP 客户端提交
func defaultTdInsert(sql string) error {
	var t TdengineTimeSeries
	return t.insert(sql)
}

// SetTdBatchInsertFn 仅供测试替换插入实现。
func SetTdBatchInsertFn(fn func(sql string) error) {
	defaultTdBatch.mu.Lock()
	defaultTdBatch.insertFn = fn
	defaultTdBatch.mu.Unlock()
}

// TdBatchStats 测试用：当前缓冲与队列长度。
func TdBatchStats() (buffered, queued int) {
	defaultTdBatch.mu.Lock()
	buffered = len(defaultTdBatch.buf)
	defaultTdBatch.mu.Unlock()
	queued = len(defaultTdBatch.ch)
	return buffered, queued
}
