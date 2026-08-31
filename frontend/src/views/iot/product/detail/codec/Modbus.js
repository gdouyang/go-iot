/**
 * websocket脚本模板
 */
const obj = {
  tpl: `function OnConnect(context) {
}
function OnMessage(context) {
}
function OnInvoke(context) {
}
`,
  demoCode: '',
  codeTip: [
    { caption: 'context', meta: 'common', value: 'context' },
    // OnMessage
    // { caption: 'context.GetMessage()', meta: 'OnMessage', value: 'var message = context.GetMessage()' },
    // { caption: 'context.MsgToString()', meta: 'OnMessage', value: 'var str = context.MsgToString()' },
    // { caption: 'context.MsgToHexStr()', meta: 'OnMessage', value: 'var hexStr = context.MsgToHexStr()' },
    // { caption: 'context.GetSession()', meta: 'OnMessage', value: 'var session = context.GetSession()' },
    // { caption: 'context.DeviceOnline()', meta: 'OnMessage', value: 'context.DeviceOnline(deviceId)' },
    // { caption: 'context.GetDevice()', meta: 'OnMessage', value: 'var deviceOper = context.GetDevice()' },
    // { caption: 'context.GetDeviceById()', meta: 'OnMessage', value: 'var deviceOper = context.GetDeviceById("id")' },
    // { caption: 'context.GetConfig()', meta: 'OnMessage', value: 'var value = context.GetConfig("key")' },
    // { caption: 'context.SaveProperties()', meta: 'OnMessage', value: 'context.SaveProperties({"key":"value"})' },
    // { caption: 'context.SaveEvents()', meta: 'OnMessage', value: 'context.SaveEvents("eventId", {"key":"value"})' },
    // { caption: 'context.ReplyOk()', meta: 'OnMessage', value: 'context.ReplyOk()' },
    // { caption: 'context.GetConfig()', meta: 'OnMessage', value: 'context.ReplyFail("resaon")' },
    // deviceOper
    {
      caption: 'deviceOper.GetConfig()',
      meta: 'deviceOper',
      value: 'var value = deviceOpr.GetConfig("key")'
    },
    {
      caption: 'deviceOper.GetData()()',
      meta: 'deviceOper',
      value: 'var map = deviceOpr.GetData()()'
    },
    // session
    {
      caption: 'session.ReadDiscreteInputs()',
      meta: 'session',
      value: 'var resp = session.ReadDiscreteInputs(startingAddress, length)'
    },
    {
      caption: 'session.ReadCoils()',
      meta: 'session',
      value: 'var resp = session.ReadCoils(startingAddress, length)'
    },
    {
      caption: 'session.ReadInputRegisters()',
      meta: 'session',
      value: 'var resp = session.ReadInputRegisters(startingAddress, length)'
    },
    {
      caption: 'session.ReadHoldingRegisters()',
      meta: 'session',
      value: 'var resp = session.ReadHoldingRegisters(startingAddress, length)'
    },

    { caption: 'resp.GetMessage()', meta: 'resp', value: 'var message = resp.GetMessage()' },
    { caption: 'resp.MsgToString()', meta: 'resp', value: 'var str = resp.MsgToString()' },
    { caption: 'resp.MsgToHexStr()', meta: 'resp', value: 'var hexStr = resp.MsgToHexStr()' },
    { caption: 'resp.MsgToUint16()', meta: 'resp', value: 'var num = resp.MsgToUint16()' },
    { caption: 'resp.MsgToUint32()', meta: 'resp', value: 'var num = resp.MsgToUint32()' },
    { caption: 'resp.MsgToUint64()', meta: 'resp', value: 'var num = resp.MsgToUint64()' },
    { caption: 'resp.MsgToBool()', meta: 'resp', value: 'var boo = resp.MsgToBool()' },

    {
      caption: 'session.WriteCoils()',
      meta: 'session',
      value: 'session.WriteCoils(startingAddress, length, hexStr)'
    },
    {
      caption: 'session.WriteHoldingRegisters()',
      meta: 'session',
      value: 'session.WriteHoldingRegisters(startingAddress, length, hexStr)'
    },
    // OnInvoke
    {
      caption: 'context.GetMessage()',
      meta: 'OnInvoke',
      value: 'var message = context.GetMessage()'
    },
    {
      caption: 'context.GetSession()',
      meta: 'OnInvoke',
      value: 'var session = context.GetSession()'
    },
    {
      caption: 'context.GetDevice()',
      meta: 'OnInvoke',
      value: 'var deviceOper = context.GetDevice()'
    },
    {
      caption: 'context.SaveProperties()',
      meta: 'OnInvoke',
      value: 'context.SaveProperties({"key":"value"})'
    },
    {
      caption: 'context.SaveEvents()',
      meta: 'OnInvoke',
      value: 'context.SaveEvents("eventId", {"key":"value"})'
    },
    { caption: 'context.ReplyOk()', meta: 'OnInvoke', value: 'context.ReplyOk()' },
    {
      caption: 'context.ReplyFail("reason")',
      meta: 'OnInvoke',
      value: 'context.ReplyFail("reason")'
    }
  ]
}
obj.demoCode = `
// 物模型 -> 设备报文
function OnInvoke(context) {
}
`
export default obj
