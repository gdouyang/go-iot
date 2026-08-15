# COAP_SERVER 编解码

无状态协议，一次请求一次响应。没有 `MsgToHexStr`、`GetHeader`、`GetForm`。

### OnMessage

| 方法 | 说明 | 参数 | 返回值 |
| --- | --- | ---- | ---- |
| GetMessage | 获取消息原始数据 | - | byte数组 |
| MsgToString | 将原始数据转换成字符串 | - | 文本 |
| GetUrl | 获取请求 path | - | string |
| GetQuery | 获取 query | (key: string) | string |
| DeviceOnline | 将设备上线 | (deviceId: string) | - |
| DeviceOffline | 将设备离线 | (deviceId: string) | - |
| GetSession | 获取Session | - | Session |
| GetDevice | 获取设备 | - | Device |
| SetDevice | 设置context中的设备 | (device: Device) | - |
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
function OnMessage(context) {
  var data = JSON.parse(context.MsgToString())
  var path = context.GetUrl()
  if (path == '/prop') {
    context.SaveProperties(data)
  }
  context.GetSession().ResponseJSON(JSON.stringify({ok: true}))
}
```

### OnInvoke

| 方法 | 说明 | 参数 | 返回值 |
| --- | --- | ---- | ---- |
| GetMessage | 获取下发消息 | - | FuncInvoke |
| GetSession | 获取Session | - | Session |
| GetDevice | 获取设备 | - | Device |
| GetDeviceById | 通过设备id获取设备 | (deviceId: string) | Device |
| GetConfig | 获取设备配置项 | (key: string) | string |
| ReplyOk | 服务下发执行成功 | - | - |
| ReplyFail | 服务下发执行失败 | (str: string) | - |

> OnInvoke 里 `DeviceOnline` 是空实现。`GetMessage()` 用 `FunctionId`、`Data`、`DeviceId`。

### FuncInvoke

| 字段 | 说明 | 参数 | 返回值 |
| --- | --- | ---- | ---- |
| FunctionId | 功能id | - | string |
| Data | 下发数据 | - | object |
| DeviceId | 设备id | - | string |
| TraceId | 跟踪id | - | string |

### Session

| 方法 | 说明 | 参数 | 返回值 |
| --- | --- | ---- | ---- |
| Disconnect | 标记设备离线 | - | - |
| GetDeviceId | 当前会话设备id | - | string |
| Response | 发送文本 | (data: string) | - |
| ResponseJSON | 发送 JSON（Content-Type 为 application/json） | (data: string) | - |

### Device

| 字段/方法 | 说明 | 参数 | 返回值 |
| --- | --- | ---- | ---- |
| Id | 设备id | - | string |
| Name | 设备名称 | - | string |
| GetId | 设备id（与 Id 相同） | - | string |
| GetConfig | 获取配置（先设备后产品） | (key: string) | string |
| SetConfig | 设置设备配置 | (key: string, value: string) | - |
| GetData | 获取临时数据 | (key: string) | string |
| SetData | 设置临时数据 | (key: string, value: string) | - |

### globe

| 方法 | 说明 | 参数 | 返回值 |
| --- | --- | ---- | ---- |
| ToCrc16Str | 计算16进制字符串的crc16 | (str: string) | string |
| BytesToBase64 | bytes数组转base64 | (bytes) | string |
| HmacEncrypt | hmac，key 为 base64 | (data: string, key: string, type: string) type: sha1/sha256/md5 | byte[] |
| HmacEncryptBase64 | hmac 后再 base64 | 同上 | string |
