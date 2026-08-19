package codec

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-iot/pkg/core"
	"go-iot/pkg/logger"
	"go-iot/pkg/option"
)

func newBenchCodec(b *testing.B, productId, script string, poolSize int) *ScriptCodec {
	b.Helper()
	logger.InitNop()
	pool, err := NewVmPool1(script, poolSize, productId)
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { pool.Close() })
	return &ScriptCodec{script: script, productId: productId, pool: pool}
}

func enableScriptHTTPBench(b *testing.B) {
	b.Helper()
	prev := option.Global
	b.Cleanup(func() { option.Global = prev })
	option.Global = &option.Options{
		Script: option.Script{HTTPEnabled: true, HTTPBlockPrivate: false},
	}
}

func BenchmarkOnMessageConsoleLog(b *testing.B) {
	c := newBenchCodec(b, "p-bench-log", `function OnMessage(context){ console.log("x") }`, 8)
	ctx := &core.BaseContext{ProductId: "p-bench-log", DeviceId: "dev-1"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := c.OnMessage(ctx); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkOnMessageConsoleLogParallel(b *testing.B) {
	c := newBenchCodec(b, "p-bench-log-p", `function OnMessage(context){ console.log("x") }`, 8)
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		var n int
		for pb.Next() {
			ctx := &core.BaseContext{ProductId: "p-bench-log-p", DeviceId: fmt.Sprintf("dev-%d", n)}
			if err := c.OnMessage(ctx); err != nil {
				b.Fatal(err)
			}
			n++
		}
	})
}

func BenchmarkOnMessageHttpRequest(b *testing.B) {
	enableScriptHTTPBench(b)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	b.Cleanup(srv.Close)
	script := `function OnMessage(context){ globe.HttpRequest({url:"` + srv.URL + `",method:"GET"}) }`
	c := newBenchCodec(b, "p-bench-http", script, 8)
	ctx := &core.BaseContext{ProductId: "p-bench-http", DeviceId: "dev-1"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := c.OnMessage(ctx); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkOnMessageHttpRequestParallel(b *testing.B) {
	enableScriptHTTPBench(b)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	b.Cleanup(srv.Close)
	script := `function OnMessage(context){ globe.HttpRequest({url:"` + srv.URL + `",method:"GET"}) }`
	c := newBenchCodec(b, "p-bench-http-p", script, 8)
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		var n int
		for pb.Next() {
			ctx := &core.BaseContext{ProductId: "p-bench-http-p", DeviceId: fmt.Sprintf("dev-%d", n)}
			if err := c.OnMessage(ctx); err != nil {
				b.Fatal(err)
			}
			n++
		}
	})
}

func BenchmarkOnMessageHttpRequestAsyncFire(b *testing.B) {
	enableScriptHTTPBench(b)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	b.Cleanup(srv.Close)
	script := `
function OnMessage(context){
  globe.HttpRequestAsync({url:"` + srv.URL + `",method:"GET",complete:function(resp){}})
}`
	c := newBenchCodec(b, "p-bench-async-fire", script, 8)
	ctx := &core.BaseContext{ProductId: "p-bench-async-fire", DeviceId: "dev-1"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := c.OnMessage(ctx); err != nil {
			b.Fatal(err)
		}
	}
}
