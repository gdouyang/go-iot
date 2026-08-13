package timeseries

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTdBatchFlushBuildsMultiInsert(t *testing.T) {
	var calls atomic.Int32
	var lastSQL string
	SetTdBatchInsertFn(func(sql string) error {
		calls.Add(1)
		lastSQL = sql
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
	require.Contains(t, lastSQL, "INSERT INTO ")
	require.Contains(t, lastSQL, "goiot.a")
	require.Contains(t, lastSQL, "goiot.b")
}
