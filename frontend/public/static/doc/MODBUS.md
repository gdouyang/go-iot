
### OnInvoke函数
> 当有进行命令下发调用OnInvoke函数

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
| Int16ToData | int16 转 hex | (num: number) | string |
| FloatToInt16Data | float 转 int16 hex | (flo: number) | string |
| FloatToUint16Data | float 转 uint16 hex | (flo: number) | string |

> 没有 OnConnect / OnMessage。轮询靠物模型功能的 `expands.interval`（秒）触发 OnInvoke。`GetMessage()` 用 `FunctionId`、`Data`、`DeviceId`。

### Session对象

| 方法 | 说明 | 参数 | 返回值 |
| --- | --- | ---- | ---- |
| Disconnect | 断开连接 | - | - |
| GetDeviceId | 当前会话设备id | - | string |
| ReadDiscreteInputs | 读取离散量输入 | (startingAddress: number, length: number) | Response |
| ReadCoils | 读取线圈 | (startingAddress: number, length: number) | Response |
| ReadInputRegisters | 读输入寄存器 | (startingAddress: number, length: number) | Response |
| ReadHoldingRegisters | 读保持寄存器 | (startingAddress: number, length: number) | Response |
| WriteCoils | 写线圈（data 为 hex） | (startingAddress: number, length: number, data: string) | - |
| WriteHoldingRegisters | 写保持寄存器（length==1 单寄存器，>1 多寄存器；data 为 hex） | (startingAddress: number, length: number, data: string) | - |

### Response

| 方法 | 说明 | 参数 | 返回值 |
| --- | --- | ---- | ---- |
| GetMessage | 获取原始返回数据 | - | byte数组 |
| MsgToString | 消息转成文本 | - | string |
| MsgToHexStr | 消息转成16进制字符串 | - | string |
| MsgToUint16 | 转 16 位无符号整型 | - | number |
| MsgToUint32 | 转 32 位无符号整型 | - | number |
| MsgToUint64 | 转 64 位无符号整型 | - | number |
| MsgToInt16 | 转 16 位有符号整型 | - | number |
| MsgToInt32 | 转 32 位有符号整型 | - | number |
| MsgToInt64 | 转 64 位有符号整型 | - | number |
| MsgToBool | 转布尔（`(data[0] & 1) > 0`） | - | boolean |

### 样例
- 编解码

```javascript
// 物模型 -> 设备报文
function OnInvoke(context) {
  var message = context.GetMessage();
  var session = context.GetSession();
  var resp = session.ReadHoldingRegisters(4003, 1);
  var dv = new DataView(new ArrayBuffer(2));
  var data = resp.GetMessage();
  dv.setUint8(0, data[0]);
  dv.setUint8(1, data[1]);
  var temp = dv.getInt16(0);
  console.log(temp);
  context.SaveProperties({"Temperature": temp / 10});
}
```
- 物模型

```json
{
  "events": [],
  "properties": [
    {
      "id": "Temperature",
      "name": "Temperature",
      "expands": {
        "readOnly": null
      },
      "description": "Temperature x 10 (np. 10,5 st.C to 105)",
      "scale": 2,
      "unit": null,
      "type": "float"
    }
  ],
  "functions": [
    {
      "id": "getTemp",
      "name": "定时获取温度",
      "expands": {
        "readOnly": null,
        "interval": "1"
      },
      "description": null,
      "output": {},
      "inputs": [],
      "async": true
    }
  ]
}
```

#### 配置ModbusPal
> https://sourceforge.net/p/modbuspal/discussion/899955/thread/72cf35ee/cd1f/attachment/ModbusPal.jar

- 添加模拟设备
![modbus_addmockdevice](img/modbus_addmockdevice.png)
- 添加寄存器
![modbus_addregister](img/modbus_addregister.png)
- 自动生成值
![modbus_addvaluegen](img/modbus_addvaluegen.png)
- 绑定值生成器
![modbus_bindvaluegen](img/modbus_bindvaluegen.png)
- 启动
![modbus_run](img/modbus_run.png)