/**
 * mqtt脚本模板
 */
const obj = {
  tpl: `// 连接时执行
function OnConnect(context) {
}
// 收到消息时执行
function OnMessage(context) {
}
// 命令调用
function OnInvoke(context) {
}
`,
  demoCode: '',
  codeTip: [
    { caption: 'context', meta: 'common', value: 'context' },
    // OnConnect
    {
      caption: 'context.GetClientId()',
      meta: 'OnConnect',
      value: 'var clientId = context.GetClientId()'
    },
    {
      caption: 'context.GetUserName()',
      meta: 'OnConnect',
      value: 'var username = context.GetUserName()'
    },
    {
      caption: 'context.GetPassword()',
      meta: 'OnConnect',
      value: 'var pwd = context.GetPassword()'
    },
    {
      caption: 'context.DeviceOnline(deviceId)',
      meta: 'OnConnect',
      value: 'context.DeviceOnline(deviceId)'
    },
    { caption: 'context.AuthFail()', meta: 'OnConnect', value: 'context.AuthFail()' },
    // OnMessage
    {
      caption: 'context.GetMessage()',
      meta: 'OnMessage',
      value: 'var message = context.GetMessage()'
    },
    {
      caption: 'context.GetSession()',
      meta: 'OnMessage',
      value: 'var session = context.GetSession()'
    },
    {
      caption: 'context.DeviceOnline(deviceId)',
      meta: 'OnMessage',
      value: 'context.DeviceOnline(deviceId)'
    },
    {
      caption: 'context.MsgToString()',
      meta: 'OnMessage',
      value: 'var str = context.MsgToString()'
    },
    {
      caption: 'context.MsgToHexStr()',
      meta: 'OnMessage',
      value: 'var hexStr = context.MsgToHexStr()'
    },

    {
      caption: 'context.GetDevice()',
      meta: 'OnMessage',
      value: 'var deviceOper = context.GetDevice()'
    },
    {
      caption: 'context.GetDeviceById(deviceId)',
      meta: 'OnMessage',
      value: 'var deviceOper = context.GetDeviceById(deviceId)'
    },
    {
      caption: 'context.GetConfig()',
      meta: 'OnMessage',
      value: 'var value = context.GetConfig("key")'
    },
    { caption: 'context.Topic()', meta: 'OnMessage', value: 'var topic = context.Topic()' },
    { caption: 'context.ReplyOk()', meta: 'OnMessage', value: 'context.ReplyOk()' },
    { caption: 'context.ReplyFail()', meta: 'OnMessage', value: 'context.ReplyFail("resaon")' },
    {
      caption: 'context.SaveProperties()',
      meta: 'OnMessage',
      value: 'context.SaveProperties({"key":"value"})'
    },
    {
      caption: 'context.SaveEvents()',
      meta: 'OnMessage',
      value: 'context.SaveEvents("eventId", {"key":"value"})'
    },
    // deviceOper
    {
      caption: 'deviceOper.GetConfig()',
      meta: 'deviceOper',
      value: 'var value = deviceOpr.GetConfig("key")'
    },
    {
      caption: 'deviceOper.GetData("key", "defaultValue")',
      meta: 'deviceOper',
      value: 'var map = deviceOper.GetData("key", "defaultValue")'
    },
    // session
    { caption: 'session.Disconnect()', meta: 'session', value: 'session.Disconnect()' },
    {
      caption: 'session.Publish("topic", "payload")',
      meta: 'session',
      value: 'session.Publish("topic", "payload")'
    },
    {
      caption: 'session.PublishHex("topic", "hexStr")',
      meta: 'session',
      value: 'session.PublishHex("topic", "68657820737472696e67")'
    },
    // OnInvoke
    {
      caption: 'context.GetMessage()',
      meta: 'OnInvoke',
      value: 'var message = context.GetMessage()'
    },
    {
      caption: 'context.GetDevice()',
      meta: 'OnInvoke',
      value: 'var deviceOper = context.GetDevice()'
    },
    {
      caption: 'message.GetClientId()',
      meta: 'OnInvoke',
      value: 'var clientId = message.GetClientId()'
    },
    { caption: 'context.ReplyOk()', meta: 'OnInvoke', value: 'context.ReplyOk()' },
    {
      caption: 'context.ReplyFail("reason")',
      meta: 'OnInvoke',
      value: 'context.ReplyFail("reason")'
    },
    // FuncInvoke
    {
      caption: 'message.FunctionId',
      meta: 'FuncInvoke',
      value: 'var functionId = message.FunctionId;'
    },
    { caption: 'message.Data', meta: 'FuncInvoke', value: 'var data = message.Data;' }
  ]
}
obj.demoCode = `function OnConnect(context) {
  console.log("OnConnect: " + context.GetClientId())
	context.DeviceOnline(context.GetClientId())
}
function OnMessage(context) {
  console.log("OnMessage: " + context.MsgToString())
  var data = JSON.parse(context.MsgToString())
	if (data.name == 'f') {
		context.ReplyOk()
		return
	}
  context.SaveProperties(data)
}
function OnInvoke(context) {
  var msg = context.GetMessage();
  msg.
	console.log("OnInvoke: " + JSON.stringify(context.GetMessage().Data))
	context.GetSession().Publish("test", JSON.stringify(context.GetMessage().Data))
}
`
export default obj
