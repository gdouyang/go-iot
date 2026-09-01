### 采集（点表）
> 平台作为 Modbus 主站。读寄存器由产品「点表/采集」完成（按组组包读），不要用物模型功能的 `expands.interval` 去轮询 OnInvoke。没有 OnConnect。

产品启用采集，且配置了采集组、点表后，运行时按组的间隔定时组包读取并声明式解码：

- 未实现 `OnMessage`：采集器直接 `SaveProperties`
- 已实现 `OnMessage`：把解码结果交给脚本，由脚本决定入库（可改、可删 key、可不报）

写线圈 / 写保持寄存器走 `OnInvoke`。`OnInvoke` 里仍可用 `Session` 额外读寄存器。

**地址基准**

- **PDU 0 起始**（`addressBase=0`）：点表填多少，报文里就是多少。适合说明书写「地址 0」。例：填 4，发出去的起始地址是 4。
- **协议 1 起始**（`addressBase=1`）：按说明书从 1 数的编号填写，发送时自动减 1。适合第一个寄存器写成 1（或 40001 对应偏移 1）的手册。例：填 1，实际读取报文地址 0。

脚本里 `Session` 的读写地址是 PDU 地址，不受点表「地址基准」影响。

**采集组**

| 字段 | 说明 |
| --- | ---- |
| id | 组标识 |
| intervalMs | 采集间隔，毫秒，最小 100 |
| report | `onPeriod` 每周期 / `onChange` 变化 / `onChangeOrPeriod` 变化或心跳 |

**点表**

| 字段 | 说明 |
| --- | ---- |
| propertyId | 对应物模型属性 |
| groupId | 所属采集组 |
| table | `HOLDING_REGISTERS` 保持 / `INPUT_REGISTERS` 输入 / `COILS` 线圈 / `DISCRETES_INPUT` 离散 |
| address | 寄存器/线圈地址，含义由地址基准决定 |
| dataType | `int16` / `uint16` / `int32` / `uint32` / `float32` / `int64` / `uint64` / `float64` / `bool` / `string` |
| scale | 缩放，解码后为 `raw * scale + offset` |
| byteOrder | 16 位 `AB`/`BA`；32/64 位 `ABCD`/`CDAB`/`BADC`/`DCBA` |

### OnMessage函数
> 采集组读完并声明式解码后调用。没有此函数时采集器直接 `SaveProperties`。有则由脚本负责入库。

| 方法 | 说明 | 参数 | 返回值 |
| --- | --- | ---- | ---- |
| GetMessage | 采集结果 | - | `{source:'collect', groupId, properties}` |
| GetSession | 获取 Session | - | Session |
| GetDevice | 获取设备 | - | Device |
| SaveProperties | 保存属性 | (data: object) | - |
| SaveEvents | 保存事件 | (eventId: string, data: object) | - |

```javascript
function OnMessage(context) {
  var msg = context.GetMessage();
  var p = msg.properties; // 已按点表解码（含 scale）
  if (p.Temperature === 3276.7) {
    delete p.Temperature;
  }
  context.SaveProperties(p);
}
```

### OnInvoke函数
> 命令下发时调用。用于写线圈、写保持寄存器等，不要用它做定时采集。

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

> `GetMessage()` 用 `FunctionId`、`Data`、`DeviceId`。

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
| Int16ToData | int16 转 hex | (num: number) | string |
| FloatToInt16Data | float 转 int16 hex | (flo: number) | string |
| FloatToUint16Data | float 转 uint16 hex | (flo: number) | string |

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
- 点表/采集（产品「点表/采集」页签，或 JSON）

```json
{
  "enabled": true,
  "addressBase": 0,
  "offlineAfterFailures": 0,
  "groups": [
    {
      "id": "g1",
      "name": "快扫",
      "intervalMs": 1000,
      "report": "onPeriod"
    }
  ],
  "points": [
    {
      "id": "p-temp",
      "propertyId": "Temperature",
      "groupId": "g1",
      "table": "HOLDING_REGISTERS",
      "address": 4,
      "dataType": "int16",
      "scale": 0.1,
      "byteOrder": "AB"
    }
  ]
}
```

- 编解码

```javascript
function OnMessage(context) {
  var msg = context.GetMessage();
  var p = msg.properties;
  context.SaveProperties(p);
}

function OnInvoke(context) {
  var message = context.GetMessage();
  var session = context.GetSession();
  if (message.FunctionId === "setTemp") {
    var hex = session.FloatToInt16Data(message.Data.Temperature * 10);
    session.WriteHoldingRegisters(4, 1, hex);
    context.ReplyOk();
  }
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
      "id": "setTemp",
      "name": "设置温度",
      "expands": {
        "readOnly": null
      },
      "description": null,
      "output": {},
      "inputs": [
        {
          "id": "Temperature",
          "name": "温度",
          "type": "float"
        }
      ],
      "async": false
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
