package eventbus

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCollectDebugSeesLivePublish(t *testing.T) {
	pid := "collect-p1"
	go func() {
		time.Sleep(30 * time.Millisecond)
		PublishDebug(NewDebugMessage("info", "d1", pid, "hello-1"))
		PublishDebug(NewDebugMessage("debug", "d2", pid, "hello-2"))
	}()
	got := CollectDebug(pid, "", 10, 400*time.Millisecond)
	require.GreaterOrEqual(t, len(got), 2)
	texts := []string{got[0].Data, got[1].Data}
	require.Contains(t, texts, "hello-1")
	require.Contains(t, texts, "hello-2")
}

func TestCollectDebugStopsAtLimit(t *testing.T) {
	pid := "collect-p2"
	go func() {
		time.Sleep(20 * time.Millisecond)
		for i := 0; i < 8; i++ {
			PublishDebug(NewDebugMessage("debug", "d", pid, "x"))
		}
	}()
	got := CollectDebug(pid, "", 3, time.Second)
	require.Len(t, got, 3)
}

func TestListenStopDrains(t *testing.T) {
	pid := "collect-p3"
	l := Listen(GetDebugTopic(pid, "*"), 8)
	PublishDebug(NewDebugMessage("warn", "d", pid, "late"))
	got := l.Stop()
	require.NotEmpty(t, got)
	d, ok := got[0].(*DebugMessage)
	require.True(t, ok)
	require.Equal(t, "late", d.Data)
	require.Empty(t, bus.m[GetDebugTopic(pid, "*")])
}

func TestTwoCollectorsSamePattern(t *testing.T) {
	topic := "/test/concurrent/collector"
	l1 := Listen(topic, 8)
	l2 := Listen(topic, 8)
	require.Len(t, bus.m[topic], 2)
	l1.Stop()
	require.Len(t, bus.m[topic], 1, "Stop of l1 should leave l2 intact")
	l2.Stop()
	require.Empty(t, bus.m[topic])
}
