package timeseries

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTdBatchFlushBuildsMultiInsert(t *testing.T) {
	var calls atomic.Int32
	var mu sync.Mutex
	var sqls []string
	SetTdBatchInsertFn(func(sql string) error {
		calls.Add(1)
		mu.Lock()
		sqls = append(sqls, sql)
		mu.Unlock()
		return nil
	})
	t.Cleanup(func() { SetTdBatchInsertFn(nil) })

	// 直接 commit 多条
	defaultTdBatch.commit("goiot.a USING s TAGS('1') (c) VALUES(1)")
	defaultTdBatch.commit("goiot.b USING s TAGS('2') (c) VALUES(2)")
	// 等 loop 收包
	time.Sleep(50 * time.Millisecond)
	FlushTdengineBatch()

	require.GreaterOrEqual(t, calls.Load(), int32(1))
	joined := strings.Join(sqls, "\n")
	require.Contains(t, joined, "INSERT INTO ")
	// 分桶后 a/b 可能并行执行（不同 worker），两个 SQL 都要出现
	require.Contains(t, joined, "goiot.a")
	require.Contains(t, joined, "goiot.b")
}

func TestTdBatchHighConcurrencyThroughput(t *testing.T) {
	var insertedRows atomic.Int64
	SetTdBatchInsertFn(func(sql string) error {
		// 模拟 TDengine HTTP 真实写入延迟 (10ms)
		time.Sleep(10 * time.Millisecond)
		// 计算 SQL 中的 VALUES 片段数量
		count := strings.Count(sql, "USING ")
		if count == 0 {
			count = 1
		}
		insertedRows.Add(int64(count))
		return nil
	})
	t.Cleanup(func() { SetTdBatchInsertFn(nil) })

	total := 20000
	var wg sync.WaitGroup
	workers := 20
	chunk := total / workers

	start := time.Now()
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for i := 0; i < chunk; i++ {
				defaultTdBatch.commit(fmt.Sprintf("goiot.dev_%d USING s TAGS('%d') (c) VALUES(%d)", workerID*chunk+i, workerID, i))
			}
		}(w)
	}
	wg.Wait()
	FlushTdengineBatch()
	cost := time.Since(start)

	t.Logf("20,000 rows committed & flushed in %v (throughput: %.1f rows/sec)", cost, float64(total)/cost.Seconds())
	require.Equal(t, int64(total), insertedRows.Load(), "all 20000 rows must be inserted without any drop")
}

// TestTdBatchBackpressureNoDrop 验证：worker 慢 + taskCh 满（分发阻塞）时，
// loop 仍持续消费 ch，全部行最终写入无丢失（回归：flush 曾同步阻塞在 taskCh 上导致 ch 停摆丢包）。
func TestTdBatchBackpressureNoDrop(t *testing.T) {
	var insertedRows atomic.Int64
	SetTdBatchInsertFn(func(sql string) error {
		// 模拟慢后端（REST 写 50ms/批）
		time.Sleep(50 * time.Millisecond)
		count := strings.Count(sql, "USING ")
		if count == 0 {
			count = 1
		}
		insertedRows.Add(int64(count))
		return nil
	})
	t.Cleanup(func() { SetTdBatchInsertFn(nil) })

	// bulk=1：每行一个 SQL → 20000 行 = 20000 个 task，远超 taskCh(cap 1000) → 必然触发分发阻塞
	defaultTdBatch.applyConfig(0, 1, 200*time.Millisecond)
	t.Cleanup(func() { defaultTdBatch.applyConfig(0, tdBulkSizeDefault, tdFlushIntervalDefault) })

	total := 20000
	var wg sync.WaitGroup
	workers := 20
	chunk := total / workers
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for i := 0; i < chunk; i++ {
				defaultTdBatch.commit(fmt.Sprintf("goiot.dev_%d USING s TAGS('%d') (c) VALUES(%d)", workerID*chunk+i, workerID, i))
			}
		}(w)
	}
	wg.Wait()
	FlushTdengineBatch()

	require.Equal(t, int64(total), insertedRows.Load(), "all 20000 rows must be flushed even when taskCh is full")
}

// TestTdBatchResizeChan 验证：applyConfig 重建 ch（buffer-size 生效）后，
// loop 被哨兵唤醒、继续消费新 ch，数据无丢失。
func TestTdBatchResizeChan(t *testing.T) {
	var insertedRows atomic.Int64
	SetTdBatchInsertFn(func(sql string) error {
		count := strings.Count(sql, "USING ")
		if count == 0 {
			count = 1
		}
		insertedRows.Add(int64(count))
		return nil
	})
	t.Cleanup(func() { SetTdBatchInsertFn(nil) })

	// 重建 ch：默认 50000 → 1000（模拟 buffer-size: 1000 配置）
	defaultTdBatch.applyConfig(1000, 0, 0)
	t.Cleanup(func() { defaultTdBatch.applyConfig(tdBatchChanSizeDefault, 0, 0) })
	require.Equal(t, 1000, cap(defaultTdBatch.ch()))

	// 重建后 commit → loop 应已切换到新 ch 消费，Flush 收口成功即证明哨兵唤醒生效
	defaultTdBatch.commit("goiot.a USING s TAGS('1') (c) VALUES(1)")
	defaultTdBatch.commit("goiot.b USING s TAGS('2') (c) VALUES(2)")
	FlushTdengineBatch()

	// 2 行必须全部经新 ch 写入（无论合并为 1 个还是 2 个 SQL）
	require.Equal(t, int64(2), insertedRows.Load(), "both rows must be flushed via the new channel without loss")
}

// TestTdBatchPerChildOrder 验证：并发交错提交时，同一子表的行最终写入顺序 = createTime 分配顺序。
// （回归：8 worker 并发消费曾导致同子表行被不同 worker 乱序写入）
func TestTdBatchPerChildOrder(t *testing.T) {
	var mu sync.Mutex
	order := make(map[string][]int) // child → 收到的 seq 序列
	SetTdBatchInsertFn(func(sql string) error {
		// SQL: INSERT INTO goiot.dev_7 USING s TAGS('7') (c) VALUES(123)
		head := strings.TrimPrefix(sql, "INSERT INTO ")
		child := childKey(head)
		var seq int
		fmt.Sscanf(head[strings.LastIndex(head, "(")+1:strings.LastIndex(head, ")")], "%d", &seq)
		mu.Lock()
		order[child] = append(order[child], seq)
		mu.Unlock()
		return nil
	})
	t.Cleanup(func() { SetTdBatchInsertFn(nil) })

	// bulk=1：每行一个 SQL，worker 串行消费队列 → 收到顺序 = 入队顺序
	defaultTdBatch.applyConfig(0, 1, 50*time.Millisecond)
	t.Cleanup(func() { defaultTdBatch.applyConfig(0, tdBulkSizeDefault, tdFlushIntervalDefault) })

	const (
		children = 20
		total    = 20000
	)
	// seq 在 commitWithTime 的回调（commitMu 锁内）递增 → 入队顺序 = 分配顺序（真实生产路径）
	var seqAtomic atomic.Int64
	var wg sync.WaitGroup
	workers := 20
	chunk := total / workers
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < chunk; i++ {
				defaultTdBatch.commitWithTime(func(_ string) string {
					seq := seqAtomic.Add(1)
					child := fmt.Sprintf("dev_%d", seq%children)
					// 同子表行由并发 goroutine 交错提交（真实场景：多设备并发上报）
					return fmt.Sprintf("goiot.%s USING s TAGS('%d') (c) VALUES(%d)", child, seq%children, seq)
				})
			}
		}()
	}
	wg.Wait()
	FlushTdengineBatch()

	require.Len(t, order, children, "all children must be written")
	for child, seqs := range order {
		require.Equal(t, total/children, len(seqs), "child %s row count", child)
		for i := 1; i < len(seqs); i++ {
			require.Greater(t, seqs[i], seqs[i-1], "child %s must be written in ascending order", child)
		}
	}
}

// TestTdCommitWithTimeMonotonic 验证：commitWithTime 原子分配+入队，
// 并发提交时单子表收到的时间戳严格递增（回归：分配与入队分离曾导致 [101,100] 乱序）。
func TestTdCommitWithTimeMonotonic(t *testing.T) {
	var mu sync.Mutex
	var times []int64
	SetTdBatchInsertFn(func(sql string) error {
		// SQL: INSERT INTO goiot.dev_1 USING s TAGS('t') (c) VALUES('2026-08-22 12:00:00.123')
		open := strings.LastIndex(sql, "(")
		close := strings.LastIndex(sql, ")")
		raw := strings.Trim(sql[open+1:close], "' ")
		tm, err := time.ParseInLocation("2006-01-02 15:04:05.000", raw, time.Local)
		if err != nil {
			return err
		}
		mu.Lock()
		times = append(times, tm.UnixMilli())
		mu.Unlock()
		return nil
	})
	t.Cleanup(func() { SetTdBatchInsertFn(nil) })

	// 单子表 + bulk=1：全部行进同一 worker 队列串行写入 → 收到顺序 = 入队顺序
	defaultTdBatch.applyConfig(0, 1, 50*time.Millisecond)
	t.Cleanup(func() { defaultTdBatch.applyConfig(0, tdBulkSizeDefault, tdFlushIntervalDefault) })

	const total = 5000
	var wg sync.WaitGroup
	workers := 20
	chunk := total / workers
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < chunk; i++ {
				defaultTdBatch.commitWithTime(func(ct string) string {
					return fmt.Sprintf("goiot.dev_1 USING s TAGS('t') (c) VALUES('%s')", ct)
				})
			}
		}()
	}
	wg.Wait()
	FlushTdengineBatch()

	require.Equal(t, total, len(times), "all rows must arrive")
	for i := 1; i < len(times); i++ {
		require.Greater(t, times[i], times[i-1], "timestamps must be strictly monotonic")
	}
}
