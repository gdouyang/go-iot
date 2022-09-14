package codec

type ScriptCodec struct {
	Script string
}

// 设备连接时
func (codec *ScriptCodec) OnConnect(ctx Context) error {
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

//
func (codec *ScriptCodec) OnStateChecker(ctx Context) error {
	return nil
}
