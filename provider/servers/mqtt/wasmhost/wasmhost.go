/*
 * Copyright (c) 2017, MegaEase
 * All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package wasmhost

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/beego/beego/v2/core/logs"
)

const (
	// Kind is the kind of WasmHost.
	Kind          = "WasmHost"
	maxWasmResult = 9
)

var (
	resultOutOfVM   = "outOfVM"
	resultWasmError = "wasmError"
	results         = []string{resultOutOfVM, resultWasmError}
)

func wasmResultToFilterResult(r int32) string {
	if r == 0 {
		return ""
	}
	return fmt.Sprintf("wasmResult%d", r)
}

func init() {
	for i := int32(1); i <= maxWasmResult; i++ {
		results = append(results, wasmResultToFilterResult(i))
	}
	// httppipeline.Register(&WasmHost{})
}

type (
	// Spec is the spec for WasmHost
	Spec struct {
		MaxConcurrency int32             `yaml:"maxConcurrency" jsonschema:"required,minimum=1"`
		Code           string            `yaml:"code" jsonschema:"required"`
		Timeout        string            `yaml:"timeout" jsonschema:"required,format=duration"`
		Parameters     map[string]string `yaml:"parameters" jsonschema:"omitempty"`
		timeout        time.Duration
	}

	// WasmHost is the WebAssembly filter
	WasmHost struct {
		spec *Spec

		code       []byte
		dataPrefix string
		data       atomic.Value
		vmPool     atomic.Value
		chStop     chan struct{}

		numOfRequest   int64
		numOfWasmError int64
	}

	// Status is the status of WasmHost
	Status struct {
		Health         string `yaml:"health"`
		NumOfRequest   int64  `yaml:"numOfRequest"`
		NumOfWasmError int64  `yaml:"numOfWasmError"`
	}
)

func NewWasmHost(spec *Spec) *WasmHost {
	wh := &WasmHost{}
	wh.Init(spec)
	return wh
}

func (wh *WasmHost) Results() []string {
	return results
}

func (wh *WasmHost) readWasmCode() ([]byte, error) {
	return base64.StdEncoding.DecodeString(wh.spec.Code)
}

func (wh *WasmHost) loadWasmCode() error {
	code, e := wh.readWasmCode()
	if e != nil {
		logs.Error("failed to load wasm code: %v", e)
		return e
	}

	if len(wh.code) > 0 && bytes.Equal(wh.code, code) {
		return nil
	}

	p, e := NewWasmVMPool(wh, code)
	if e != nil {
		logs.Error("failed to create wasm VM pool: %v", e)
		return e
	}
	wh.code = code

	wh.vmPool.Store(p)
	return nil
}

func (wh *WasmHost) reload(spec *Spec) {
	wh.spec = spec

	wh.spec.timeout, _ = time.ParseDuration(wh.spec.Timeout)
	wh.chStop = make(chan struct{})

	wh.loadWasmCode()
}

// Init initializes WasmHost.
func (wh *WasmHost) Init(spec *Spec) {
	wh.reload(spec)
}

func (wh *WasmHost) Handle(ctx *MqttContext) (result string) {
	// we must save the pool to a local variable for later use as it will be
	// replaced when updating the wasm code
	var pool *WasmVMPool
	if p := wh.vmPool.Load(); p == nil {
		// ctx.AddTag("wasm VM pool is not initialized")
		logs.Warn("wasm VM pool is not initialized")
		return resultOutOfVM
	} else {
		pool = p.(*WasmVMPool)
	}

	// get a free wasm VM and attach the ctx to it
	vm := pool.Get()
	if vm == nil {
		// ctx.AddTag("failed to get a wasm VM")
		logs.Warn("failed to get a wasm VM")
		return resultOutOfVM
	}
	vm.ctx = ctx
	atomic.AddInt64(&wh.numOfRequest, 1)

	var wg sync.WaitGroup
	chCancelInterrupt := make(chan struct{})
	defer func() {
		close(chCancelInterrupt)
		wg.Wait()

		// the VM is not usable if there's a panic, set it to nil and a new
		// VM will be created in pool.Get later
		if e := recover(); e != nil {
			logs.Error("recovered from wasm error: %v", e)
			result = resultWasmError
			atomic.AddInt64(&wh.numOfWasmError, 1)
			vm = nil
		}

		pool.Put(vm)
	}()

	// start another goroutine to interrupt the wasm execution
	wg.Add(1)
	go func() {
		defer wg.Done()

		timer := time.NewTimer(wh.spec.timeout)

		select {
		case <-chCancelInterrupt:
			break
		case <-timer.C:
			vm.Interrupt()
			vm = nil
			break
		case <-ctx.Done():
			vm.Interrupt()
			vm = nil
			break
		}

		if !timer.Stop() {
			<-timer.C
		}
	}()

	r := vm.Run() // execute wasm code
	n, ok := r.(int32)
	if !ok || n < 0 || n > maxWasmResult {
		panic(fmt.Errorf("invalid wasm result: %v", r))
	}

	return wasmResultToFilterResult(n)
}

// Status returns Status generated by the filter.
func (wh *WasmHost) Status() interface{} {
	p := wh.vmPool.Load()
	s := &Status{}
	if p == nil {
		s.Health = "VM pool is not initialized"
	} else {
		s.Health = "ready"
	}

	s.NumOfRequest = atomic.LoadInt64(&wh.numOfRequest)
	s.NumOfWasmError = atomic.LoadInt64(&wh.numOfWasmError)
	return s
}

// Close closes WasmHost.
func (wh *WasmHost) Close() {
	close(wh.chStop)
}

// VM实例总数
func (wh *WasmHost) TotalNumbersOfVM() int32 {
	var pool *WasmVMPool
	if p := wh.vmPool.Load(); p == nil {
		return int32(0)
	} else {
		pool = p.(*WasmVMPool)
	}
	return pool.total
}