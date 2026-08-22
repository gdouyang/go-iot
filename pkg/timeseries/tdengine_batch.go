package timeseries

import (
	"hash/fnv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	logs "go-iot/pkg/logger"
)

const (
	// 默认值；运行时可由 ConfigTdengine 覆盖 buffer / bulk / flush
	tdBatchChanSizeDefault = 50000
	tdBulkSizeDefault      = 1000
	tdFlushIntervalDefault = 200 * time.Millisecond
	tdWorkerCountDefault   = 8
	// tdChanSentinel 用于唤醒阻塞在旧 ch select 上的 loop（ch 容量重建时）
	tdChanSentinel = "\x00tdengine-chan-resized"
)

type tdInsertTask struct {
	sql      string
	rowCount int
	insertFn func(sql string) error
}

// tdBatchSaver 将多条 INSERT 片段合并为一次 REST 提交。
// SQL 形态：
//
//	INSERT INTO
//	  child1 USING stable TAGS(...) (cols) VALUES (...),
//	  child2 USING stable TAGS(...) (cols) VALUES (...);
type tdBatchSaver struct {
	chPtr          atomic.Pointer[chan string] // 提交队列；容量重建时原子切换（并发安全）
	buf            []string
	lastCommitTime int64
	mu             sync.Mutex // 保护 buf / lastCommitTime 与测试同步
	bulkSize       int
	flushInterval  time.Duration
	// flushHook 测试可注入
	insertFn func(sql string) error

	taskChs  []chan tdInsertTask // 按子表 hash 分桶的 worker 队列：同子表永远进同一队列（FIFO 保序）
	inFlight sync.WaitGroup
	flushCh  chan struct{} // flush 请求信号（cap=1 合并，flusher 忙时丢信号由后续请求兜底）
	flushMu  sync.Mutex    // flush 串行化：flusher 与 FlushTdengineBatch 互斥，保证 inFlight.Add 不与 Wait 并发
	commitMu sync.Mutex    // 保证 createTime 分配与入队原子（ch 内顺序 = 分配顺序）
}

// ch 返回当前提交队列（原子读取）。
func (s *tdBatchSaver) ch() chan string {
	return *s.chPtr.Load()
}

var defaultTdBatch = newTdBatchSaver(tdBatchChanSizeDefault, tdBulkSizeDefault, tdFlushIntervalDefault, tdWorkerCountDefault)

func newTdBatchSaver(chanSize, bulkSize int, flushInterval time.Duration, workers int) *tdBatchSaver {
	ch := make(chan string, chanSize)
	s := &tdBatchSaver{
		buf:            make([]string, 0, bulkSize),
		lastCommitTime: time.Now().UnixMilli(),
		bulkSize:       bulkSize,
		flushInterval:  flushInterval,
		flushCh:        make(chan struct{}, 1),
	}
	s.chPtr.Store(&ch)
	s.taskChs = make([]chan tdInsertTask, workers)
	for i := 0; i < workers; i++ {
		s.taskChs[i] = make(chan tdInsertTask, 1000)
		go s.workerLoop(i)
	}
	go s.loop()
	go s.flusher()
	return s
}

func (s *tdBatchSaver) workerLoop(i int) {
	for task := range s.taskChs[i] {		start := time.Now()
		var err error
		if task.insertFn != nil {
			err = task.insertFn(task.sql)
		} else {
			err = defaultTdInsert(task.sql)
		}
		cost := time.Since(start)
		if err != nil {
			logs.Errorf("tdengine batch insert error (rows=%d cost=%s): %v", task.rowCount, cost, err)
		} else if cost > time.Second {
			logs.Warnf("tdengine batch insert slow: rows=%d cost=%s", task.rowCount, cost)
		} else {
			logs.Debugf("tdengine batch insert ok: rows=%d cost=%s", task.rowCount, cost)
		}
		s.inFlight.Done()
	}
}

// applyConfig 更新批量写入参数；bufferSize>0 且与当前 ch 容量不同时重建 ch（仅启动早期调用，安全）。
func (s *tdBatchSaver) applyConfig(bufferSize, bulkSize int, flushInterval time.Duration) {
	s.mu.Lock()
	if bulkSize > 0 {
		s.bulkSize = bulkSize
	}
	if flushInterval > 0 {
		s.flushInterval = flushInterval
	}
	if bufferSize > 0 && bufferSize != cap(s.ch()) {
		old := s.ch()
		newCh := make(chan string, bufferSize)
		s.chPtr.Store(&newCh)
		// 唤醒可能在旧 ch 上 select 的 loop，避免其永久阻塞在旧 ch
		// （配置仅启动期加载，此时 ch 为空，不会阻塞；正常 fragment 不可能以 \x00 开头）
		old <- tdChanSentinel
	}
	s.mu.Unlock()
}

// 保证 createTime 毫秒全局递增，降低同设备并发写冲突
var tdLastCreateMs atomic.Int64

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

// FlushTdengineBatch 立刻刷出缓冲并等待所有在途写入完成（单测 / 优雅停机可用）。
// 全程持有 flushMu：保证 flusher 不会并发 flush（即 inFlight.Add 不会与下面的 Wait 并发）。
func FlushTdengineBatch() {
	defaultTdBatch.flushMu.Lock()
	defer defaultTdBatch.flushMu.Unlock()
	// 先把 channel 中已入队片段并入 buf，再 flush；再等待在途 worker 处理完
	for round := 0; round < 3; round++ {
		for {
			select {
			case frag := <-defaultTdBatch.ch():
				defaultTdBatch.mu.Lock()
				defaultTdBatch.buf = append(defaultTdBatch.buf, frag)
				defaultTdBatch.mu.Unlock()
			default:
				goto flushed
			}
		}
	flushed:
		defaultTdBatch.flushLocked()
		defaultTdBatch.inFlight.Wait()
		if round < 2 {
			time.Sleep(15 * time.Millisecond)
		}
	}
}

// commitWithTime 在 commitMu 保护下原子完成 createTime 分配 + fragment 构造 + 入队，
// 保证 ch 内顺序 = createTime 分配顺序（否则同子表行可能以乱序时间戳写入 TDengine）。
// 返回分配的 createTime。
func (s *tdBatchSaver) commitWithTime(build func(createTime string) string) string {
	s.commitMu.Lock()
	defer s.commitMu.Unlock()
	ct := nextTdCreateTime()
	frag := build(ct)
	s.commit(frag)
	return ct
}

// commit 非阻塞入队；队列满时丢行并告警。
func (s *tdBatchSaver) commit(fragment string) {
	if fragment == "" {
		return
	}
	ch := s.ch()
	select {
	case ch <- fragment:
	default:
		logs.Warnf("tdengine batch chan full (cap=%d), drop 1 row", cap(ch))
	}
}

// requestFlush 非阻塞提交 flush 请求（信号 cap=1 合并）。
// loop 只负责消费 ch→buf，绝不在此阻塞，因此 ch 永不因分发慢而停摆。
func (s *tdBatchSaver) requestFlush() {
	select {
	case s.flushCh <- struct{}{}:
	default:
	}
}

// flusher 串行消费 flush 请求；taskCh 满时在此阻塞形成背压（有界并发），不影响 loop 消费 ch。
func (s *tdBatchSaver) flusher() {
	for range s.flushCh {
		s.flush()
	}
}

func (s *tdBatchSaver) loop() {
	// 用较短 tick 轮询配置中的 flush 间隔
	ticker := time.NewTicker(50 * time.Millisecond)
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
				s.requestFlush()
			}
		case frag := <-s.ch():
			if frag == tdChanSentinel {
				// ch 已重建，重新 select 新 s.ch
				continue
			}
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
				s.requestFlush()
			}
		}
	}
}

func (s *tdBatchSaver) flush() {
	s.flushMu.Lock()
	defer s.flushMu.Unlock()
	s.flushLocked()
}

// flushLocked 调用方需持有 flushMu。
func (s *tdBatchSaver) flushLocked() {
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
	fn := s.insertFn
	s.mu.Unlock()

	bulk := s.bulkSize
	if bulk <= 0 {
		bulk = tdBulkSizeDefault
	}
	// 按 USING stable 分组，避免不同 super table 混在一条 INSERT 里出问题
	groups := groupFragmentsByStable(batch)
	for _, frags := range groups {
		// 二级：按子表分组后按 hash 分桶；同子表行永远进同一桶（worker 串行消费 ⇒ 写入保序）
		buckets := make([][]string, len(s.taskChs))
		for child, childFrags := range groupFragmentsByChild(frags) {
			b := int(hashChild(child) % uint32(len(s.taskChs)))
			// 同一 child 的行在桶内连续追加（child 内顺序保持），可跨 child 合并成批量 SQL
			buckets[b] = append(buckets[b], childFrags...)
		}
		for b, items := range buckets {
			if len(items) == 0 {
				continue
			}
			for i := 0; i < len(items); i += bulk {
				end := i + bulk
				if end > len(items) {
					end = len(items)
				}
				sql := buildMultiInsertSQL(items[i:end])
				task := tdInsertTask{
					sql:      sql,
					rowCount: end - i,
					insertFn: fn,
				}
				s.inFlight.Add(1)
				s.taskChs[b] <- task
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

// groupFragmentsByChild 按子表名分组（组内保持原顺序 = createTime 递增顺序）。
func groupFragmentsByChild(frags []string) map[string][]string {
	out := make(map[string][]string)
	for _, f := range frags {
		key := childKey(f)
		out[key] = append(out[key], f)
	}
	return out
}

// childKey 从 fragment（db.child USING ...）提取子表名；子表名本身可含 "."（不受影响）。
func childKey(f string) string {
	head := f
	if i := strings.Index(f, " USING "); i >= 0 {
		head = f[:i]
	}
	if i := strings.Index(head, "."); i >= 0 {
		head = head[i+1:]
	}
	return head
}

// hashChild 子表名 → 固定 worker 分桶（FNV-1a）。
func hashChild(child string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(child))
	return h.Sum32()
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
	queued = len(defaultTdBatch.ch())
	return buffered, queued
}
