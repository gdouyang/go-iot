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
	"bufio"
	"bytes"
	"encoding/binary"
	"math/rand"
	"net/http"
	"net/textproto"
	"strings"
	"time"

	"github.com/beego/beego/v2/core/logs"
	"github.com/bytecodealliance/wasmtime-go"
)

// helper functions

const wasmMemory = "memory"

func (vm *WasmVM) readDataFromWasm(addr int32) []byte {
	mem := vm.inst.GetExport(vm.store, wasmMemory).Memory().UnsafeData(vm.store)
	size := int32(binary.LittleEndian.Uint32(mem[addr:]))
	data := make([]byte, size)
	copy(data, mem[addr+4:])
	return data
}

func (vm *WasmVM) writeDataToWasm(data []byte) int32 {
	mem := vm.inst.GetExport(vm.store, wasmMemory).Memory().UnsafeData(vm.store)

	vaddr, e := vm.fnAlloc.Call(vm.store, len(data)+4)
	if e != nil {
		panic(e)
	}
	addr := vaddr.(int32)

	binary.LittleEndian.PutUint32(mem[addr:], uint32(len(data)))
	copy(mem[addr+4:], data)

	return addr
}

func (vm *WasmVM) readStringFromWasm(addr int32) string {
	mem := vm.inst.GetExport(vm.store, wasmMemory).Memory().UnsafeData(vm.store)
	size := int32(binary.LittleEndian.Uint32(mem[addr:]))
	data := make([]byte, size-1)
	copy(data, mem[addr+4:])
	return string(data)
}

// a string is serialized as 4 byte length + content + trailing zero
func (vm *WasmVM) writeStringToWasm(s string) int32 {
	mem := vm.inst.GetExport(vm.store, wasmMemory).Memory().UnsafeData(vm.store)

	vaddr, e := vm.fnAlloc.Call(vm.store, len(s)+4+1)
	if e != nil {
		panic(e)
	}
	addr := vaddr.(int32)

	binary.LittleEndian.PutUint32(mem[addr:], uint32(len(s)+1))
	copy(mem[addr+4:], s)
	mem[addr+4+int32(len(s))] = 0

	return addr
}

func (vm *WasmVM) writeStringArrayToWasm(strs []string) int32 {
	size := 4 // 4 is sizeof(int32)
	for _, s := range strs {
		size += len(s) + 4 + 1
	}

	mem := vm.inst.GetExport(vm.store, wasmMemory).Memory().UnsafeData(vm.store)
	vaddr, e := vm.fnAlloc.Call(vm.store, int32(size))
	if e != nil {
		panic(e)
	}
	addr := vaddr.(int32)
	pos := int(addr)

	binary.LittleEndian.PutUint32(mem[pos:], uint32(len(strs)))
	pos += 4

	for _, s := range strs {
		binary.LittleEndian.PutUint32(mem[pos:], uint32(len(s)+1))
		pos += 4
		copy(mem[pos:], []byte(s))
		pos += len(s)
		mem[pos] = 0
		pos++
	}

	return addr
}

func (vm *WasmVM) writeHeaderToWasm(h http.Header) int32 {
	var buf bytes.Buffer
	h.Write(&buf)
	return vm.writeStringToWasm(buf.String())
}

func (vm *WasmVM) readHeaderFromWasm(addr int32) http.Header {
	str := vm.readStringFromWasm(addr) + "\r\n"
	r := textproto.NewReader(bufio.NewReader(strings.NewReader(str)))
	h, e := r.ReadMIMEHeader()
	if e != nil {
		panic(e)
	}
	return http.Header(h)
}

func (vm *WasmVM) readClusterKeyFromWasm(addr int32) string {
	key := vm.readStringFromWasm(addr)
	return vm.host.dataPrefix + key
}

// request functions

func (vm *WasmVM) hostGetClientID() int32 {
	v := vm.ctx.ClientID()
	return vm.writeStringToWasm(v)
}

func (vm *WasmVM) hostGetUserName() int32 {
	v := vm.ctx.UserName()
	return vm.writeStringToWasm(v)
}

func (vm *WasmVM) hostLog(level int32, addr int32) {
	msg := vm.readStringFromWasm(addr)
	switch level {
	case 0:
		logs.Debug(msg)
	case 1:
		logs.Info(msg)
	case 2:
		logs.Warn(msg)
	case 3:
		logs.Error(msg)
	}
}

func (vm *WasmVM) hostGetUnixTimeInMs() int64 {
	return time.Now().UnixNano() / 1e6
}

func (vm *WasmVM) hostRand() float64 {
	return rand.Float64()
}

// importHostFuncs imports host functions into wasm so that user-developed wasm
// code can call these functions to interoperate with host.
func (vm *WasmVM) importHostFuncs(linker *wasmtime.Linker) {
	defineFunc := func(name string, fn interface{}) {
		if e := linker.DefineFunc(vm.store, "easegress", name, fn); e != nil {
			panic(e) // should never happen
		}
	}

	// request functions
	defineFunc("host_req_get_client_id", vm.hostGetClientID)
	defineFunc("host_req_get_user_name", vm.hostGetUserName)

	defineFunc("host_log", vm.hostLog)
	defineFunc("host_get_unix_time_in_ms", vm.hostGetUnixTimeInMs)
	defineFunc("host_rand", vm.hostRand)
}
