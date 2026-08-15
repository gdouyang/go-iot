Go-IoT MQTT Broker是一个公用的Mqtt Broker（默认端口为1883），所有的设备都可以连接到这个Broker，设备的clientId将是设备id。

### OnConnect函数
> 当有客户端连接是会调用OnConnect函数，在OnConnect函数中可以对客户端进行账号密码的校验，默认情况下客户端的clientId将是设备id，当设置username与password时将校验是否与配置中的一致
- context参数说明

| 方法 | 说明 | 参数 | 返回值 |
| --- | --- | ---- | ---- |
| GetClientId | 获取mqtt clientId | - | string |
| GetUserName | 获取mqtt 用户名 | - | string |
| GetPassword | 获取mqtt 密码（仅 OnConnect 有） | - | string |
| GetDeviceById | 通过设备id获取设备 | (deviceId: string) | Device |
| DeviceOnline | 将设备上线 | (deviceId: string) | - |
| AuthFail | 认证失败 | - | - |

> OnConnect 的 `GetSession()` / `GetMessage()` 为 nil。认证前还没有设备，不要用 `context.GetConfig`，用 `GetDeviceById(GetClientId()).GetConfig(...)`。clientId 就是设备 id。

```javascript
// 系统默认会根据用户名和密码来认证，如果不满足可写OnConnect来自行判断
// 当mqtt客户端连接到Broker时可以在这里判断用户名和密码是否正确
function OnConnect(context) {
  var device = context.GetDeviceById(context.GetClientId())
  if (device && context.GetUserName() == device.GetConfig("username") && context.GetPassword() == device.GetConfig("password")) {
    context.DeviceOnline(context.GetClientId())
    return
  }
  context.AuthFail()
}
```
### OnMessage函数
> 当有消息接收时会调用OnMessage函数
- context参数说明

| 方法 | 说明 | 参数 | 返回值 |
| --- | --- | ---- | ---- |
| GetMessage | 获取消息原始数据 | - | byte数组 |
| MsgToString | 将原始数据转换成字符串 | - | 文本 |
| MsgToHexStr | 将原始数据转换成16进制字符串 | - | 16进制字符串 |
| Topic | 获取消息Topic | - | string |
| MessageID | 获取 MQTT messageId | - | number |
| GetClientId | 获取 mqtt clientId | - | string |
| GetUserName | 获取 mqtt 用户名 | - | string |
| DeviceOnline | 将设备上线 | (deviceId: string) | - |
| GetSession | 获取Session | - | Session |
| GetDevice | 获取设备 | - | Device |
| GetDeviceById | 通过设备id获取设备 | (deviceId: string) | Device |
| GetProduct | 获取产品 | - | Product |
| GetConfig | 获取设备配置项 | (key: string) | string |
| SaveProperties | 保存属性 | (data: object) | - |
| SaveEvents | 保存事件 | (eventId: string, data: object) | - |
| ReplyOk | 服务下发执行成功 | - | - |
| ReplyFail | 服务下发执行失败 | (str: string) | - |
| ReplyAsync | 异步功能回复 | (resp: {success,msg,traceId}) | - |
| KeepAlive | 刷新设备在线超时 | (deviceId: string) | - |

```javascript
// 当客户端向Broker推送数据时，执行OnMessage函数
function OnMessage(context) {
  console.log("OnMessage topic: " + context.Topic())
  console.log("OnMessage msg: " + context.MsgToString())
  var data = JSON.parse(context.MsgToString())
  var topic = context.Topic()
  if (topic == '/reply') {
    context.ReplyOk()
  } else if (topic == '/prop') {
    context.SaveProperties(data)
  } else if (topic == '/event') {
    context.SaveEvents(data.eventId, data)
  }
}
```

### OnInvoke函数
> 当有进行命令下发调用OnInvoke函数
- context参数说明

| 方法 | 说明 | 参数 | 返回值 |
| --- | --- | ---- | ---- |
| GetMessage | 获取下发消息 | - | FuncInvoke |
| GetSession | 获取Session | - | Session |
| GetDevice | 获取设备 | - | Device |
| GetDeviceById | 通过设备id获取设备 | (deviceId: string) | Device |
| GetConfig | 获取设备配置项 | (key: string) | string |
| SaveProperties | 保存属性 | (data: object) | - |
| SaveEvents | 保存事件 | (eventId: string, data: object) | - |
| ReplyOk | 服务下发执行成功 | - | - |
| ReplyFail | 服务下发执行失败 | (str: string) | - |
| ReplyAsync | 异步功能回复 | (resp: {success,msg,traceId}) | - |

> OnInvoke 里 `DeviceOnline` 是空实现。`GetMessage()` 用 `FunctionId`、`Data`、`DeviceId`，不是 `inputs`。

- FuncInvoke

| 字段 | 说明 | 参数 | 返回值 |
| --- | --- | ---- | ---- |
| FunctionId | 功能id | - | string |
| Data | 下发数据 | - | object |
| DeviceId | 设备id | - | string |
| TraceId | 跟踪id | - | string |

```javascript
function OnInvoke(context) {
  var data = JSON.stringify(context.GetMessage().Data)
	console.log("OnInvoke: " + data)
  var session = context.GetSession()
  // 向客户端发送文本信息
	session.Publish("/invoke", data)
}
```

### Session对象

| 方法 | 说明 | 参数 | 返回值 |
| --- | --- | ---- | ---- |
| Disconnect | 断开连接 | - | - |
| GetDeviceId | 当前会话设备id | - | string |
| Publish | 发送文本数据 | (topic: string, data: string) | - |
| PublishHex | 将16进制文本数据转换成byte发送 | (topic: string, data: string) | - |

### Device对象

| 字段/方法 | 说明 | 参数 | 返回值 |
| --- | --- | ---- | ---- |
| Id | 设备id | - | string |
| Name | 设备名称 | - | string |
| GetId | 设备id（与 Id 相同） | - | string |
| GetConfig | 获取配置（先设备后产品） | (key: string) | string |
| SetConfig | 设置设备配置 | (key: string, value: string) | - |
| GetData | 获取临时数据 | (key: string) | string |
| SetData | 设置临时数据 | (key: string, value: string) | - |

### globe对象

| 方法 | 说明 | 参数 | 返回值 |
| --- | --- | ---- | ---- |
| ToCrc16Str | 计算16进制字符串的crc16 | (str: string) | string |
| BytesToBase64 | bytes数组转base64 | (bytes) | string |
| HmacEncrypt | hmac，key 为 base64 | (data: string, key: string, type: string) type: sha1/sha256/md5 | byte[] |
| HmacEncryptBase64 | hmac 后再 base64 | 同上 | string |

### 样例
```json
{
  "events": [],
  "properties": [
    {
      "id": "temperature",
      "name": "温度",
      "expands": {
        "readOnly": null
      },
      "description": null,
      "scale": 2,
      "unit": "°C",
      "type": "float"
    }
  ],
  "functions": []
}
```
```javascript
// 系统默认会根据用户名和密码来认证，如果不满足可写OnConnect来自行判断
// 当mqtt客户端连接到Broker时可以在这里判断用户名和密码是否正确
function OnConnect(context) {
  var device = context.GetDeviceById(context.GetClientId())
  if (device && context.GetUserName() == device.GetConfig("username") && context.GetPassword() == device.GetConfig("password")) {
    context.DeviceOnline(context.GetClientId())
    return
  }
  context.AuthFail()
}

function OnMessage(context) {
  console.log("OnMessage: " + context.MsgToString())
  var data = JSON.parse(context.MsgToString())
  if (data.name == 'reply') {
		context.ReplyOk()
		return
  }
  var topic = context.Topic()
  if (topic == '/prop') {
    context.SaveProperties(data)
  } else if (topic == '/event') {
    context.SaveEvents(data.eventId, data)
  }
}

function OnInvoke(context) {
  var data = JSON.stringify(context.GetMessage().Data)
	console.log("OnInvoke: " + data)
  var session = context.GetSession()
  // 向Broker发送文本信息
	session.Publish("/invoke", data)
}
```