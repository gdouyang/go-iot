import _ from 'lodash-es'

var variables =
  'Array|Boolean|Date|Function|Iterator|Number|Object|RegExp|String|' + // Constructors
  'ArrayBuffer|Float32Array|Float64Array|Int16Array|Int32Array|Int8Array|' +
  'Uint16Array|Uint32Array|Uint8Array|Uint8ClampedArray|' +
  'Error|EvalError|InternalError|RangeError|ReferenceError|StopIteration|' + // Errors
  'SyntaxError|TypeError|URIError|' +
  'isNaN|parseFloat|parseInt|' +
  'JSON|Math|' + // Other
  'this|arguments|'
var tips = []
variables.split('|').forEach((item) => {
  tips.push({ caption: item, meta: 'keyword', value: item, completerId: 'keywordCompleter' })
})
var keys =
  'const|yield|import|get|set|' +
  'break|case|catch|continue|default|delete|do|else|finally|for|function|' +
  'if|in|of|instanceof|new|return|switch|throw|try|typeof|var|while|with'
keys.split('|').forEach((item) => {
  tips.push({ caption: item, meta: 'keyword', value: item, completerId: 'keywordCompleter' })
})

export function getKeywordCompleters() {
  return tips
}

function _getCompletions(type) {
  var list = []
  if (type === 'MODBUS') {
    list = [
      { caption: 'context', meta: 'common', value: 'context' },
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
      { caption: 'context.GetConfig()', meta: 'OnInvoke', value: 'context.ReplyFail("resaon")' }
    ]
  } else {
    list = [
      {
        caption: 'context',
        meta: 'common',
        value: 'context',
        remark: '上下文，可以获取到消息、session、设备等数据'
      },
      // OnMessage
      {
        caption: 'context.GetMessage()',
        meta: 'OnMessage',
        value: 'var message = context.GetMessage()',
        remark: '获取消息信息'
      },
      {
        caption: 'context.MsgToString()',
        meta: 'OnMessage',
        value: 'var str = context.MsgToString()',
        remark: '返回消息的字符串格式'
      },
      {
        caption: 'context.GetSession()',
        meta: 'OnMessage',
        value: 'var session = context.GetSession()',
        remark: '获取设备Session'
      },
      {
        caption: 'context.DeviceOnline()',
        meta: 'OnConnect',
        value: 'context.DeviceOnline(deviceId)',
        remark: '使设备上线'
      },
      {
        caption: 'context.GetDevice()',
        meta: 'OnMessage',
        value: 'var deviceOper = context.GetDevice()',
        remark: '获取当前设备（没上线时返回null）'
      },
      {
        caption: 'context.SetDevice()',
        meta: 'OnMessage',
        value: 'context.SetDevice()',
        remark:
          '设置context中的Device，对于无状态连接可以先使用GetDeviceById，然后调用SetDevice来设置'
      },
      {
        caption: 'context.GetDeviceById()',
        meta: 'OnMessage',
        value: 'var deviceOper = context.GetDeviceById("id")',
        remark: '获取指定设备'
      },
      {
        caption: 'context.GetConfig()',
        meta: 'OnMessage',
        value: 'var value = context.GetConfig("key")',
        remark: '获取当前设备的配置'
      },
      {
        caption: 'context.SaveProperties()',
        meta: 'OnMessage',
        value: 'context.SaveProperties({"key":"value"})',
        remark: '保存物模型属性数据'
      },
      {
        caption: 'context.SaveEvents()',
        meta: 'OnMessage',
        value: 'context.SaveEvents("eventId", {"key":"value"})',
        remark: '保存物模型事件数据'
      },
      {
        caption: 'context.ReplyOk()',
        meta: 'OnMessage',
        value: 'context.ReplyOk()',
        remark: '指令下发成功'
      },
      {
        caption: 'context.ReplyFail()',
        meta: 'OnMessage',
        value: 'context.ReplyFail("resaon")',
        remark: '指令失败'
      },
      // deviceOper
      {
        caption: 'deviceOper.GetConfig()',
        meta: 'deviceOper',
        value: 'var value = deviceOpr.GetConfig("key")',
        remark: '获取设备配置'
      },
      {
        caption: 'deviceOper.GetData()()',
        meta: 'deviceOper',
        value: 'var map = deviceOpr.GetData()()'
      },
      // session
      {
        caption: 'session.Disconnect()',
        meta: 'session',
        value: 'session.Disconnect()',
        remark: '断开连接'
      },
      // OnInvoke
      {
        caption: 'context.GetMessage()',
        meta: 'OnInvoke',
        value: 'var message = context.GetMessage()',
        remark: '获取下发信息'
      },
      {
        caption: 'context.GetDevice()',
        meta: 'OnInvoke',
        value: 'var deviceOper = context.GetDevice()'
      },
      {
        caption: 'context.ReplyOk()',
        meta: 'OnInvoke',
        value: 'context.ReplyOk()',
        remark: '指令下发成功'
      },
      {
        caption: 'context.ReplyFail()',
        meta: 'OnInvoke',
        value: 'context.ReplyFail("resaon")',
        remark: '指令失败'
      },
      // FuncInvoke
      {
        caption: 'message.FunctionId',
        meta: 'FuncInvoke',
        value: 'var functionId = message.FunctionId;',
        remark: '物模型功能id'
      },
      {
        caption: 'message.Data',
        meta: 'FuncInvoke',
        value: 'var data = message.Data;',
        remark: '下发的数据'
      }
    ]
    if (type === 'MQTT_BROKER' || type === 'MQTT_CLIENT') {
      // OnConnect
      list = list.concat([
        {
          caption: 'context.GetClientId()',
          meta: 'OnConnect',
          value: 'var clientId = context.GetClientId()',
          remark: '获取Mqtt的clentId'
        },
        {
          caption: 'context.GetUserName()',
          meta: 'OnConnect',
          value: 'var username = context.GetUserName()',
          remark: '获取Mqtt的用户名'
        },
        {
          caption: 'context.GetPassword()',
          meta: 'OnConnect',
          value: 'var pwd = context.GetPassword()',
          remark: '获取Mqtt的密码'
        },
        {
          caption: 'context.AuthFail()',
          meta: 'OnConnect',
          value: 'context.AuthFail()',
          remark: '自定义认证失败, 认证失败会断开连接'
        },
        {
          caption: 'context.Topic()',
          meta: 'OnMessage',
          value: 'var topic = context.Topic()',
          remark: '获取消息的topic字符串'
        },
        {
          caption: 'context.MsgToHexStr()',
          meta: 'OnMessage',
          value: 'var hexStr = context.MsgToHexStr()',
          remark: '获取消息内容转成16进制字符串, 当消息为byte数组时可以使用'
        },
        {
          caption: 'session.Publish()',
          meta: 'session',
          value: 'session.Publish("topic", "payload")',
          remark: '发布消息给指定topic, 内容为字符串'
        },
        {
          caption: 'session.PublishHex()',
          meta: 'session',
          value: 'session.PublishHex("topic", "68657820737472696e67")',
          remark: '发布消息给指定topic, 内容为16进制字符串, 会转成byte数组'
        }
      ])
    } else if (type === 'WEBSOCKET_SERVER' || type === 'HTTP_SERVER') {
      list = list.concat([
        {
          caption: 'context.DeviceOffline()',
          meta: 'OnMessage',
          value: 'context.DeviceOffline(deviceId)',
          remark: '修改设备状态为离线'
        },
        {
          caption: 'context.GetHeader()',
          meta: 'OnMessage',
          value: 'var value = context.GetHeader("key")',
          remark: '获取http头'
        },
        {
          caption: 'context.GetUrl()',
          meta: 'OnMessage',
          value: 'var url = context.GetUrl()',
          remark: '获取请求url'
        },
        {
          caption: 'context.GetQuery()',
          meta: 'OnMessage',
          value: 'var value = context.GetQuery("key")',
          remark: '获取url中的queryString, www.domain.com?abc=1'
        },
        {
          caption: 'context.GetForm()',
          meta: 'OnMessage',
          value: 'var value = context.GetForm("key")',
          remark: '获取表单提交的数据'
        }
      ])
      if (type === 'WEBSOCKET_SERVER') {
        list = list.concat([
          {
            caption: 'context.IsTextMessage()',
            meta: 'OnMessage',
            value: 'var yes = context.IsTextMessage()',
            remark: '是否为文本类型的消息（true,false）'
          },
          {
            caption: 'context.IsBinaryMessage()',
            meta: 'OnMessage',
            value: 'var yes = context.IsBinaryMessage()',
            remark: '是否为二进制类型的消息（true,false）'
          },
          {
            caption: 'session.SendText()',
            meta: 'session',
            value: 'session.SendText("text")',
            remark: '发送文本数据'
          },
          {
            caption: 'session.SendBinary()',
            meta: 'session',
            value: 'session.SendBinary("68657820737472696e67")',
            remark: '发送二进制数据, 内容为16进制字符串, 会转成byte数组'
          }
        ])
      } else {
        list = list.concat([
          {
            caption: 'globe.HttpRequest()',
            meta: 'globe',
            value: `var resp = globe.HttpRequest({
    method: "post", 
    url: "", 
    data: {}, 
    headers: {}
  })`,
            remark: `发送http请求（异步）
var resp = globe.HttpRequest({
  method: "post", 
  url: "", 
  data: {}, 
  headers: {}
})
console.log(resp)
`
          },
          {
            caption: 'globe.HttpRequestAsync()',
            meta: 'globe',
            value: `globe.HttpRequestAsync({
    method: "post", 
    url: "", 
    data: {}, 
    headers: {},
    complete: function(resp) {
      console.log(resp)
    }
  })`,
            remark: `发送http请求（异步）
globe.HttpRequestAsync({
  method: "post", 
  url: "", 
  data: {}, 
  headers: {},
  complete: function(resp) {
    console.log(resp)
  }
})`
          },
          {
            caption: 'session.ResponseHeader()',
            meta: 'session',
            value: 'session.ResponseHeader("key", "value")',
            remark: '设置响应头'
          },
          {
            caption: 'session.Response()',
            meta: 'session',
            value: 'session.Response("text")',
            remark: '响应文本数据'
          },
          {
            caption: 'session.ResponseJSON()',
            meta: 'session',
            value: 'session.ResponseJSON("{}")',
            remark: '响应Json数据'
          },
          {
            caption: 'session.SetStatesCode()',
            meta: 'session',
            value: 'session.SetStatesCode(200)',
            remark: '设置http响应states code'
          }
        ])
      }
    } else if (type === 'TCP_SERVER' || type === 'TCP_CLIENT') {
      list = list.concat([
        {
          caption: 'session.Send()',
          meta: 'session',
          value: 'session.Send("text")',
          remark: '发送文本数据'
        },
        {
          caption: 'session.SendHex()',
          meta: 'session',
          value: 'session.SendHex("68657820737472696e67")',
          remark: '发送二进制数据, 内容为16进制字符串, 会转成byte数组'
        }
      ])
    }
  }
  list = list.concat([
    {
      caption: 'JSON.parse',
      meta: 'keyword',
      value: 'JSON.parse(jstr);',
      remark: '解析JSON格式字符串为对象'
    },
    {
      caption: 'JSON.stringify',
      meta: 'keyword',
      value: 'JSON.stringify(object);',
      remark: '把对象序列化为JSON格式字符'
    }
  ])
  return list
}

export function getCompletions(type) {
  var list = _getCompletions(type)
  list.forEach((item) => {
    item.completerId = 'goiotkeywordCompleter'
  })
  return list // .concat(tips)
}

export function addCompletions(editor, datas) {
  if (!editor.completers) {
    return
  }
  // 加入自定义语法提示
  var list = []
  editor.completers.forEach((item) => {
    if (item.id !== 'keywordCompleter') {
      list.push(item)
    }
  })
  var s1 = getKeywordCompleters()
  list.push({
    id: 'keywordCompleter',
    getCompletions: function (editor, session, pos, prefix, callback) {
      console.log(prefix)
      prefix = _.trim(prefix)
      if (!prefix || prefix === '.' || prefix.length === 0) {
        return callback(null, [])
      } else {
        return callback(null, s1)
      }
    }
  })
  list.push({
    id: 'goiotkeywordCompleter',
    identifierRegexps: [/[a-zA-Z_0-9.]/],
    getCompletions: function (editor, session, pos, prefix, callback) {
      console.log(prefix)
      prefix = _.trim(prefix)
      if (!prefix || prefix === '.' || prefix.length === 0) {
        return callback(null, [])
      } else {
        return callback(null, datas)
      }
    },
    getDocTooltip: function (item) {
      if (item.remark) {
        item.docHTML = ['<b>', item.caption, '</b>', '<hr></hr>', item.remark].join('')
      }
    }
  })
  editor.completers = list
}
