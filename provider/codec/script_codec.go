package codec

import (
	"github.com/robertkrimen/otto"
)

// productId
var m = map[string]*ScriptCodec{}

func NewScriptCodec(productId, script string, session *Session) error {
	vm := otto.New()
	_, err := vm.Run(script)
	m[productId] = &ScriptCodec{
		script:  script,
		vm:      vm,
		session: session,
	}

	return err
}

func GetScriptCodec(productId string) *ScriptCodec {
	codec := m[productId]
	return codec
}

// js脚本编解码
type ScriptCodec struct {
	script  string
	vm      *otto.Otto
	session *Session
}

// 设备连接时
func (codec *ScriptCodec) OnConnect(ctx Context) error {
	// codec.vm.Call()
	return nil
}

// 设备解码
func (codec *ScriptCodec) Decode(ctx Context) error {
	return nil
}

// 编码
func (codec *ScriptCodec) Encode(ctx Context) error {
	return nil
}

// 设备新增
func (codec *ScriptCodec) OnDeviceCreate(ctx Context) error {
	return nil
}

// 设备删除
func (codec *ScriptCodec) OnDeviceDelete(ctx Context) error {
	return nil
}

// 设备修改
func (codec *ScriptCodec) OnDeviceUpdate(ctx Context) error {
	return nil
}

func (codec *ScriptCodec) OnStateChecker(ctx Context) error {
	return nil
}
