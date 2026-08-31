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
    {
      caption: 'context.GetMessage()',
      meta: 'OnMessage',
      value: 'var message = context.GetMessage()'
    },
    {
      caption: 'context.MsgToString()',
      meta: 'OnMessage',
      value: 'var str = context.MsgToString()'
    },
    {
      caption: 'context.GetSession()',
      meta: 'OnMessage',
      value: 'var session = context.GetSession()'
    },
    {
      caption: 'context.DeviceOnline()',
      meta: 'OnMessage',
      value: 'context.DeviceOnline(deviceId)'
    },
    {
      caption: 'context.GetDevice()',
      meta: 'OnMessage',
      value: 'var deviceOper = context.GetDevice()'
    },
    {
      caption: 'context.GetDeviceById("id")',
      meta: 'OnMessage',
      value: 'var deviceOper = context.GetDeviceById("id")'
    },
    {
      caption: 'context.GetConfig("key")',
      meta: 'OnMessage',
      value: 'var value = context.GetConfig("key")'
    },
    {
      caption: 'context.IsTextMessage()',
      meta: 'OnMessage',
      value: 'var yes = context.IsTextMessage()'
    },
    {
      caption: 'context.IsBinaryMessage()',
      meta: 'OnMessage',
      value: 'var yes = context.IsBinaryMessage()'
    },
    {
      caption: 'context.GetHeader()',
      meta: 'OnMessage',
      value: 'var value = context.GetHeader("key")'
    },
    { caption: 'context.GetUrl()', meta: 'OnMessage', value: 'var url = context.GetUrl()' },
    {
      caption: 'context.GetQuery()',
      meta: 'OnMessage',
      value: 'var value = context.GetQuery("key")'
    },
    {
      caption: 'context.GetForm()',
      meta: 'OnMessage',
      value: 'var value = context.GetForm("key")'
    },
    {
      caption: 'context.SaveProperties({"key":"value"})',
      meta: 'OnMessage',
      value: 'context.SaveProperties({"key":"value"})'
    },
    {
      caption: 'context.SaveEvents("eventId", {"key":"value"})',
      meta: 'OnMessage',
      value: 'context.SaveEvents("eventId", {"key":"value"})'
    },
    { caption: 'context.ReplyOk()', meta: 'OnMessage', value: 'context.ReplyOk()' },
    {
      caption: 'context.ReplyFail("reason")',
      meta: 'OnMessage',
      value: 'context.ReplyFail("reason")'
    },
    // deviceOper
    {
      caption: 'deviceOper.GetConfig("key")',
      meta: 'deviceOper',
      value: 'var value = deviceOper.GetConfig("key")'
    },
    {
      caption: 'deviceOper.GetData("key")',
      meta: 'deviceOper',
      value: 'var map = deviceOper.GetData("key")'
    },
    // session
    { caption: 'session.Disconnect()', meta: 'session', value: 'session.Disconnect()' },
    { caption: 'session.SendText("text")', meta: 'session', value: 'session.SendText("text")' },
    {
      caption: 'session.SendBinary("68657820737472696e67")',
      meta: 'session',
      value: 'session.SendBinary("68657820737472696e67")'
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
    { caption: 'context.GetConfig()', meta: 'OnInvoke', value: 'context.ReplyFail("resaon")' },
    // FuncInvoke
    {
      caption: 'message.FunctionId',
      meta: 'FuncInvoke',
      value: 'var functionId = message.FunctionId;'
    },
    { caption: 'message.Data', meta: 'FuncInvoke', value: 'var data = message.Data;' }
  ]
}
obj.demoCode = `// 设备报文 -> 物模型
function OnMessage(context) {
  var session = context.GetSession();
  var payload = JSON.parse(context.MsgToString());
  var topic = context.GetUrl();
  // 根据路径来判断是什么类型
  if (topic.startsWith("/report-property")) {
    context.SaveProperties(payload)
  } else if (topic.startsWith("/fault_alarm")) {
    context.SaveEvents('fire_alarm', payload)
  } else {
    // 如果都不匹配则给客户端返回404
    session.SendText('{"status":404, "message": "uri ['+ topic +']not support"}')
  }

}

// 物模型 -> 设备报文
function OnInvoke(context) {
  var message = context.GetMessage();
  context.GetSession().SendText(JSON.stringify(message.Data))
}
`
export default obj
