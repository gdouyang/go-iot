package codec

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go-iot/pkg/core"
	"go-iot/pkg/eventbus"
	"go-iot/pkg/logger"
	"go-iot/pkg/option"

	"github.com/stretchr/testify/require"
)

func enableScriptHTTP(t *testing.T) {
	t.Helper()
	prev := option.Global
	t.Cleanup(func() { option.Global = prev })
	option.Global = &option.Options{
		Script: option.Script{HTTPEnabled: true, HTTPBlockPrivate: false},
	}
}

func newTestScriptCodec(t *testing.T, productId, script string, poolSize int) *ScriptCodec {
	t.Helper()
	pool, err := NewVmPool1(script, poolSize, productId)
	require.NoError(t, err)
	t.Cleanup(func() { pool.Close() })
	return &ScriptCodec{script: script, productId: productId, pool: pool}
}

func subscribeDebugData(t *testing.T, productId string) <-chan *eventbus.DebugMessage {
	t.Helper()
	ch := make(chan *eventbus.DebugMessage, 64)
	pattern := "/device/" + productId + "/*/debug"
	handler := func(msg eventbus.Message) {
		if d, ok := msg.(*eventbus.DebugMessage); ok {
			select {
			case ch <- d:
			default:
			}
		}
	}
	eventbus.Subscribe(pattern, handler)
	t.Cleanup(func() { eventbus.UnSubscribe(pattern, handler) })
	return ch
}

func waitDebugData(t *testing.T, ch <-chan *eventbus.DebugMessage, want string, timeout time.Duration) *eventbus.DebugMessage {
	t.Helper()
	deadline := time.After(timeout)
	for {
		select {
		case msg := <-ch:
			if msg.Data == want {
				return msg
			}
		case <-deadline:
			t.Fatalf("timeout waiting debug data %q", want)
			return nil
		}
	}
}

func TestHttpRequestAsyncCompleteConcurrentDeviceId(t *testing.T) {
	logger.InitNop()
	enableScriptHTTP(t)

	var inflight atomic.Int32
	var maxInflight atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := inflight.Add(1)
		for {
			old := maxInflight.Load()
			if n <= old || maxInflight.CompareAndSwap(old, n) {
				break
			}
		}
		time.Sleep(30 * time.Millisecond)
		inflight.Add(-1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(srv.Close)

	productId := "p-http-async-conc"
	script := `
function OnMessage(context) {
  globe.HttpRequestAsync({
    url: "` + srv.URL + `",
    method: "GET",
    complete: function(resp) {
      console.log("async-done")
    }
  })
}
`
	c := newTestScriptCodec(t, productId, script, 4)
	logs := subscribeDebugData(t, productId)

	const n = 12
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := "dev-async-" + string(rune('a'+i))
			err := c.OnMessage(&core.BaseContext{ProductId: productId, DeviceId: id})
			require.NoError(t, err)
		}(i)
	}
	wg.Wait()

	got := map[string]int{}
	deadline := time.After(5 * time.Second)
	for len(got) < n {
		select {
		case msg := <-logs:
			if msg.Data != "async-done" {
				continue
			}
			got[msg.DeviceId]++
		case <-deadline:
			t.Fatalf("got %d/%d async completes: %v", len(got), n, got)
		}
	}
	require.Len(t, got, n)
	for i := 0; i < n; i++ {
		id := "dev-async-" + string(rune('a'+i))
		require.Equal(t, 1, got[id], "device %s", id)
	}
	require.Greater(t, maxInflight.Load(), int32(1), "expected overlapping HTTP requests")
}

func TestHttpRequestSyncConcurrentDeviceId(t *testing.T) {
	logger.InitNop()
	enableScriptHTTP(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(20 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(srv.Close)

	productId := "p-http-sync-conc"
	script := `
function OnMessage(context) {
  var resp = globe.HttpRequest({
    url: "` + srv.URL + `",
    method: "GET"
  })
  console.log("sync-done")
}
`
	c := newTestScriptCodec(t, productId, script, 4)
	logs := subscribeDebugData(t, productId)

	const n = 8
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := "dev-sync-" + string(rune('a'+i))
			err := c.OnMessage(&core.BaseContext{ProductId: productId, DeviceId: id})
			require.NoError(t, err)
		}(i)
	}
	wg.Wait()

	got := map[string]int{}
	deadline := time.After(5 * time.Second)
	for len(got) < n {
		select {
		case msg := <-logs:
			if msg.Data != "sync-done" {
				continue
			}
			got[msg.DeviceId]++
		case <-deadline:
			t.Fatalf("got %d/%d sync logs: %v", len(got), n, got)
		}
	}
	require.Len(t, got, n)
}

func TestHttpRequestAsyncCompleteOverlapsNextInvoke(t *testing.T) {
	logger.InitNop()
	enableScriptHTTP(t)

	release := make(chan struct{})
	started := make(chan struct{}, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case started <- struct{}{}:
		default:
		}
		if r.URL.Path == "/slow" {
			<-release
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(srv.Close)

	productId := "p-http-overlap"
	script := `
function OnMessage(context) {
  var url = context.DeviceId === "dev-a" ? "` + srv.URL + `/slow" : "` + srv.URL + `/fast"
  if (context.DeviceId === "dev-a") {
    globe.HttpRequestAsync({
      url: url,
      method: "GET",
      complete: function(resp) {
        console.log("async-a")
      }
    })
    return
  }
  globe.HttpRequest({ url: url, method: "GET" })
  console.log("sync-b")
}
`
	// 池大小 1，强制 complete 与下一轮 OnMessage 复用同一 VM
	c := newTestScriptCodec(t, productId, script, 1)
	logs := subscribeDebugData(t, productId)

	require.NoError(t, c.OnMessage(&core.BaseContext{ProductId: productId, DeviceId: "dev-a"}))
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("slow request not started")
	}

	require.NoError(t, c.OnMessage(&core.BaseContext{ProductId: productId, DeviceId: "dev-b"}))
	syncB := waitDebugData(t, logs, "sync-b", 3*time.Second)
	require.Equal(t, "dev-b", syncB.DeviceId)

	close(release)
	asyncA := waitDebugData(t, logs, "async-a", 3*time.Second)
	require.Equal(t, "dev-a", asyncA.DeviceId)
}

func TestHttpRequestThenAsyncInSameInvoke(t *testing.T) {
	logger.InitNop()
	enableScriptHTTP(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(srv.Close)

	productId := "p-http-same-invoke"
	script := `
function OnMessage(context) {
  globe.HttpRequestAsync({
    url: "` + srv.URL + `",
    method: "GET",
    complete: function(resp) {
      console.log("async-same")
    }
  })
  globe.HttpRequest({
    url: "` + srv.URL + `",
    method: "GET"
  })
  console.log("sync-same")
}
`
	c := newTestScriptCodec(t, productId, script, 1)
	logs := subscribeDebugData(t, productId)

	require.NoError(t, c.OnMessage(&core.BaseContext{ProductId: productId, DeviceId: "dev-same"}))
	syncMsg := waitDebugData(t, logs, "sync-same", 3*time.Second)
	require.Equal(t, "dev-same", syncMsg.DeviceId)
	asyncMsg := waitDebugData(t, logs, "async-same", 3*time.Second)
	require.Equal(t, "dev-same", asyncMsg.DeviceId)
}
