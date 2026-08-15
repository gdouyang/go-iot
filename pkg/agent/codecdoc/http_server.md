### OnStateChecker 函数
> OnStateChecker的作用是获取在线状态，由于http协议是无状态的，状态需要实时查询
- context参数说明参考OnMessage函数

- 返回值 online, offline, unknown

```javascript
function OnStateChecker(context) {
  var resp = globe.HttpRequest({
    url: 'http://www.example.com/?deviceId='+context.GetDevice().GetId(),
    method: 'get'
  })
  if (resp.data == 'online') {
    return 'online'
  }
  return 'offline'
}
```
### OnDeviceDeploy 函数
> 设备启用/新增时调用此函数，可以在这个函数中调用第三方系统做数据同步
```javascript
function OnDeviceDeploy(context) {
}
```
### OnDeviceUnDeploy 函数
> 设备停用时调用此函数，可以在这个函数中调用第三方系统做数据同步
```javascript
function OnDeviceUnDeploy(context) {
}
```
### OnMessage 函数
> 当有消息接收时会调用OnMessage函数
```javascript
function OnMessage(context) {
}
```
- context参数说明

| 方法 | 说明 | 参数 | 返回值 |
| --- | --- | ---- | ---- |
| GetMessage | 获取消息原始数据 | - | byte数组 |
| MsgToString | 将原始数据转换成字符串 | - | 文本 |
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
| GetHeader | 获取http请求头 | (key: string) | string |
| GetUrl | 获取http url | - | string |
| GetQuery | 获取http query | (key: string) | string |
| GetForm | 获取http表单 | (key: string) | string |

### OnInvoke 函数
> 功能调用时执行，在这里实现具体的命令下方逻辑
```javascript
function OnInvoke(context) {
}
```
- context参数说明

| 方法 | 说明 | 参数 | 返回值 |
| --- | --- | ---- | ---- |
| GetMessage | 获取下发消息 | - | FuncInvoke |
| GetSession | 获取Session | - | Session |
| GetDevice | 获取设备 | - | Device |
| GetDeviceById | 通过设备id获取设备 | (deviceId: string) | Device |
| GetConfig | 获取设备配置项 | (key: string) | string |
| ReplyOk | 服务下发执行成功 | - | - |
| ReplyFail | 服务下发执行失败 | (str: string) | - |
| ReplyAsync | 异步功能回复 | (resp: {success,msg,traceId}) | - |

> OnInvoke 里 `DeviceOnline` 是空实现，不能用来上线。`GetMessage()` 用 `FunctionId`、`Data`、`DeviceId`，不是 `inputs`。

### FuncInvoke

| 字段 | 说明 | 参数 | 返回值 |
| --- | --- | ---- | ---- |
| FunctionId | 功能id | - | string |
| Data | 下发数据 | - | object |
| DeviceId | 设备id | - | string |
| TraceId | 跟踪id | - | string |

### Session对象

| 方法 | 说明 | 参数 | 返回值 |
| --- | --- | ---- | ---- |
| Disconnect | 标记设备离线 | - | - |
| GetDeviceId | 当前会话设备id | - | string |
| Response | 发送文本数据 | (data: string) | - |
| ResponseJSON | 发送json数据 | (data: string) | - |
| ResponseHeader | 设置http响应头 | (key: string, value: string) | - |
| SetStatesCode | 设置http响应status code | (code: int) | - |

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
| GetDataInt | 获取临时数据并转 int | (key: string) | number |

### globe对象

| 方法 | 说明 | 参数 | 返回值 |
| --- | --- | ---- | ---- |
| HttpRequest | 发送http请求 | HttpConfig | HttpResp |
| HttpRequestAsync | 发送http请求（异步） | HttpConfig | - |
| ToCrc16Str | 计算16进制字符串的crc16 | (str: string) | string |
| BytesToBase64 | bytes数组转base64 | (bytes) | string |
| HmacEncrypt | hmac，key 为 base64 | (data: string, key: string, type: string) type: sha1/sha256/md5 | byte[] |
| HmacEncryptBase64 | hmac 后再 base64 | 同上 | string |

#### HttpConfig

| 属性 | 说明 | 类型 |
| --- | --- | ---- |
| method | http请求方法("get", "post", "put", "delete") | string |
| url | 完整 http(s) 地址，必须带协议 | string |
| data | 下发数据 | object / string |
| headers | http请求头 | object |
| timeout | 超时秒数，默认 3 | number |
| complete | 执行完的回调(HttpRequestAsync才有效) | function |

#### HttpResp

| 属性 | 说明 | 类型 |
| --- | --- | ---- |
| status | 响应码(20x,30x,40x,50x) | int |
| message | 响应消息 | string |
| header | 响应头 | object |
| data | 响应数据 | string |

### 对接移动Onet平台
- [测试产品](./doc/onenet-lamp.json)

```javascript
// 检查在线状态
function OnStateChecker(context) {
  // unknown, online, offline;
  // 获取页面上配置的地址与apiKey
  var url= context.GetConfig('apiAddress')
  var resp = globe.HttpRequest({
    method: 'get',
    url: url + '/devices/getbyimei?imei='+ context.GetDevice().Id,
    headers: {'Authorization': getAuthorization(context)},
  })
  if (resp.status >= 200 && resp.status < 300) {
    console.log(resp.data);
    var body = JSON.parse(resp.data);
    if (body.errno != 0) {
      return "offline"
    }
    if (body.data.online) {
      return "online"
    }
  }
  return "offline"
}
function getAuthorization(context) {
  var version = "2018-10-31";
  var resourceName = "products/8001495"; // 这里需要替换具体的oneNet产品id
  var expirationTime = Number(new Date().getTime() / 1000 + 100 * 24 * 60 * 60).toFixed();
  var signatureMethod = "sha1";
  var accessKey = context.GetConfig('accessKey');

  var res = encodeURIComponent(resourceName);
  var encryptText = expirationTime + "\n" + signatureMethod + "\n" + resourceName + "\n" + version;
  var str = globe.HmacEncryptBase64(encryptText, accessKey, signatureMethod);
  var sig = encodeURIComponent(str);
  return "version=" + version + "&res=" + res + "&et=" + expirationTime+"&method="+signatureMethod+"&sign="+sig;
}
// 设备激活时同步到OneNet平台
function OnDeviceDeploy(context) {
  var url= context.GetConfig('apiAddress')
  var device = context.GetDevice()
  var auth_info = {}
  auth_info[device.Id] = device.Id
  var resp = globe.HttpRequest({
    method: 'post',
    url: url + '/devices',
    data: {
      title: device.Name,
      protocol: 'LWM2M',
      auth_info: auth_info
    },
    headers: {'Authorization': getAuthorization(context)},
  })
  console.log(resp)
  if (resp.status >= 200 && resp.status < 300) {
    console.log(resp.data);
    var body = JSON.parse(resp.data);
    if (body.errno != 0) {
      if (body.errno == 6) {// 如果OneNet平台已存在
        return;
      }
      throw new Error("同步OnNet失败 errno:"+body.error)
    }
    var device_id = body.data.device_id
    device.SetConfig("deviceId", device_id)
    return;
  }
  throw new Error("同步OnNet失败"+resp.message)
}
// 设备禁用时同步OneNet平台删除
function OnDeviceUnDeploy(context) {
  var url= context.GetConfig('apiAddress')
  var device = context.GetDevice()
  var device_id = device.GetConfig('deviceId')
  if (!device_id) {
    device_id = device.Id
  }
  var resp = globe.HttpRequest({
    method: 'delete',
    url: url + '/devices/' + device_id,
    headers: {'Authorization': getAuthorization(context)},
  })
  if (resp.status >= 200 && resp.status < 300) {
    console.log(resp.data);
    var body = JSON.parse(resp.data);
    if (body.errno != 0 && body.errno != 3) {
      throw new Error("同步OnNet失败 errno:"+body.error)
    }
    return;
  }
  throw new Error("同步OnNet失败"+resp.message)
}
/**
 * 2：设备上下线消息
 */
var ONLINE_OFFLINE = 2;
/**
 * 1：设备上线
 */
var ONLINE = "1";
/**
 * 0：设备下线
 */
var OFFLINE = "0";
// 设备报文 -> 物模型
function OnMessage(context) {
  var str = context.MsgToString();
  console.log(str)
  var data = JSON.parse(str);
  for (var i = 0; i < data.msg.length; i++) {
    var msg = data.msg[i];
    var imei = msg.imei;
    var msgType = msg.type;
    // 处理上线消息
    if (ONLINE_OFFLINE == msgType) {
      var status = msg.status;
      if (ONLINE == status) {
        context.DeviceOnline(imei)
        continue;
      } else if (OFFLINE == status) {
        context.DeviceOffline(imei)
        continue;
      }
    }
    context.SetDevice(context.GetDeviceById(imei))

    var value = msg.value;
    var entity = new MsgEntity(value);
    var msgHeader = entity.header;
    var value1 = msgHeader.msgType.value;
    if (MsgType.心跳消息.equals(value1) || MsgType.上行接入请求.equals(value1)) {
      console.log("心跳消息" + MsgType.心跳消息.equals(value1) + ",上行接入请求" + MsgType.上行接入请求.equals(value1))
      // 灯控器心中、上行接入时需要返回ack不然会断开连接
      doAck(context, imei, msgHeader);
    }
    // 心跳消息中包含了亮度、电流、电压、功率
    if (MsgType.心跳消息.equals(value1)) {
      var light = entity.getNextIntegerWord(); // 亮度 0表示关闭
      var current = entity.getNextIntegerWord(); // 电流 单位mA
      var voltage = entity.getNextIntegerWord(); // 电压 V
      var power = entity.getNextIntegerWord(); // 功率 W
      var json = {
        deviceId: imei,
        light: light,
        current: current,
        voltage: voltage,
        power: power
      };
      console.log("SaveProperties:" + JSON.stringify(json))
      context.SaveProperties(json)
    }
  }
}
// 物模型 -> 设备报文
function OnInvoke(context) {
  var message = context.GetMessage();
  if (!message.FunctionId) {
    throw new Error('只支持设备功能调用');
  }
  var functionId = message.FunctionId;
  var messageId = "1"//message.Id();
  var invoke = new FunctionInvokeUtil(context);
  var result = null;
  if (functionId == "timing") {
    result = invoke.timing();
  } else if (functionId == "switching") {
    result = invoke.switching(message);
  } else if (functionId == "dimming") {
    result = invoke.dimming(message);
  } else if (functionId == "strategy") {
    result = invoke.strategy(message);
  } else {
    throw new Error(functionId + '无效功能ID');
  }
  var resp = sendToOneNet(context, message.DeviceId, {'args': result});
  if (resp.status == 200) {
    // 发送成功后要处理回复
    context.ReplyOk()
  } else {
    context.ReplyFail(resp.message)
  }
}

function doAck(context, imei, msgHeader) {
  var value1 = msgHeader.msgType.value;
  var args = '';
  if (MsgType.上行接入请求.equals(value1)) {
    args = new FunctionInvokeUtil(context).connectResp();
  } else if (MsgType.心跳消息.equals(value1)) {
    args = new FunctionInvokeUtil(context).heartbeatResp(msgHeader);
  }
  sendToOneNetAsync(context, imei, {'args': args});
}

// 发送指令给移动平台
function sendToOneNet(context, imei, data) {
  // 获取页面上配置的地址与apiKey
  var url= context.GetConfig('apiAddress')
  return globe.HttpRequest({
    method: 'post',
    url: url + '/nbiot/execute?imei='+ imei +'&obj_id=3200&obj_inst_id=0&res_id=5505',
    data: data,
    headers: {'Authorization': getAuthorization(context)},
  })
}
function sendToOneNetAsync(context, imei, data) {
  // 获取页面上配置的地址与apiKey
  var url= context.GetConfig('apiAddress')
  globe.HttpRequestAsync({
    method: 'post',
    url: url + '/nbiot/execute?imei='+ imei +'&obj_id=3200&obj_inst_id=0&res_id=5505',
    data: data,
    headers: {'Authorization': getAuthorization(context)},
    complete: function(resp) {
      console.log(resp)
    }
  })
}

function FunctionInvokeUtil(context) {
  this.context = context
  // 开关
  this.switching = function(message) {
    var status = message.Data.status;
    if ("on" == status) {
      return CmdUtil.getCmdMsg(this.context, MsgType.开关灯调光, CmdUtil.getHexStr(10, 1), null);
    } else {
      return CmdUtil.getCmdMsg(this.context, MsgType.开关灯调光, CmdUtil.getHexStr(0, 1), null);
    }
  }
  // 调光
  this.dimming = function(message) {
    var data = message.Data.bright ? message.Data.bright : 0;
    return CmdUtil.getCmdMsg(this.context, MsgType.开关灯调光, CmdUtil.getHexStr(parseInt(data), 1), null);
  }
  // 设置策略
  this.strategy = function(message) {
    var data = message.Data.strategy
    if (data) {
      var dateStr = "";
      // 时-分-亮度,...,...："18-30-30,20-30-80,5-30-50,6-30-0"
      var strategyS = data.split(",");
      for(var i = 0; i < strategyS.length; i++) {
        var s = strategyS[i];
        var strategy = s.split("-");
        for(var j = 0; j < strategy.length; j++) {
          var t = strategy[j];
          dateStr += CmdUtil.getHexStr(parseInt(t), 1);
        }
      }
      return CmdUtil.getCmdMsg(this.context, MsgType.设置策略, dateStr, strategyS.length * 3);
    }
  }
  // 校时
  this.timing = function () {
    return this.resp(MsgType.校时);
  }
  // 接入应答
  this.connectResp = function () {
    return this.resp(MsgType.接入应答);
  }
  // 下行ACK
  this.heartbeatResp = function (msgHeader) {
    return CmdUtil.getCmdMsg(this.context, MsgType.下行ACK, msgHeader.pktNum.hex, null);
  }
  // 应答
  this.resp = function(msgType) {
    var date = new Date();
    var year = (date.getFullYear() + "").substring(2);
    var dateStr = CmdUtil.getHexStr(parseInt(year), 1);
    dateStr += CmdUtil.getHexStr(date.getMonth() + 1, 1);
    dateStr += CmdUtil.getHexStr(date.getDate(), 1);
    dateStr += CmdUtil.getHexStr(date.getHours(), 1);
    dateStr += CmdUtil.getHexStr(date.getMinutes(), 1);
    dateStr += CmdUtil.getHexStr(date.getSeconds(), 1);
    return CmdUtil.getCmdMsg(this.context, msgType, dateStr, null);
  }
}

function CmdUtil() {
}
CmdUtil.getCmdMsg = function(context, msgType, msgData, len) {
  var version = "0100";
  var msgLen = len !== null ? CmdUtil.getHexStr(len,2) : msgType.lenHex;
  var msgHead = version + CmdUtil.getPktNum(context) + msgType.codeHex + msgLen;
  var msgCrc16 = globe.ToCrc16Str(msgHead + msgData);

  return (msgHead + msgData + msgCrc16).toUpperCase();
}
CmdUtil.getHexStr = function(value, size) {
  // 十进制转十六进制，并补齐四位
  var hexStr = value.toString(16);
  while (hexStr.length < 4) {
    hexStr = "0" + hexStr;
  }
  // 前后置位
  hexStr = hexStr.substring(2, 4) + hexStr.substring(0, 2);
  // 按需返回
  return hexStr.substring(0, size * 2);
}

CmdUtil.getPktNum = function(context) {
  var pktNum = context.GetDevice().GetData("pktNum");
  if (pktNum) {
    pktNum = parseInt(pktNum, 10);
    pktNum = pktNum + 1;
  } else {
    pktNum = 0;
  }
  context.GetDevice().SetData("pktNum", pktNum);
  return CmdUtil.getHexStr(pktNum, 2);
}

function MsgEntity(text) {
  var header = text.substring(0, 16);
  var msgHeader = new MsgHeader(header);
  var length = msgHeader.msgLen.value;
  var endIndex = 16 + (length * 2);

  this.body = text.substring(16, endIndex);
  this.header = msgHeader;
  this.offset = 0;

  this.resetOffset = function () {
    this.offset = 0;
  }

  this.getNextIntegerBYTE = function () {
    return parseInt(this.getNextByte(), 16);
  }

  this.getNextWord = function () {
    var high = this.getNextByte();
    var low = this.getNextByte();
    return low + high;
  }

  this.getNextByte = function () {
    var endIndex = this.offset + 2;
    if (endIndex > this.body.length) {
        return null;
    }
    var str = this.body.substring(this.offset, endIndex);
    this.offset = endIndex;
    return str;
  }

  this.getNextIntegerWord = function () {
    return parseInt(this.getNextWord(), 16);
  }
}

function MsgHeader(header) {
  var versionStr = header.substring(0, 4);
  var pktNumStr = header.substring(4, 8); // 序列号，递增
  var msgTypeStr = header.substring(8, 12); // 消息类型
  var msgLenStr = header.substring(12, 16); // 消息体长度

  this.version = new DataTypeWord(versionStr);
  this.pktNum = new DataTypeWord(pktNumStr);
  this.msgType = new DataTypeWord(msgTypeStr);
  this.msgLen = new DataTypeWord(msgLenStr);
}

function DataTypeWord(text) {
  this.hex = text; // string

  this.highBit = text.substring(2, 4); // string
  this.lowBit = text.substring(0, 2); // string
  this.value = parseInt(this.highBit + this.lowBit, 16); // int

}

function MsgType(typeCode, desc, codeHex, lenHex) {
  this.typeCode = typeCode;
  this.typeDesc = desc;
  this.codeHex = codeHex;
  this.lenHex = lenHex;

  this.equals = function(code) {
    return this.typeCode == code;
  }
}
// 上行消息类型
MsgType.上行ACK = new MsgType(1, "上行ACK", "0100", "");
MsgType.上行NACK = new MsgType(2, "上行NACK", "0200", "");
MsgType.上行接入请求 = new MsgType(3, "上行接入请求", "0300", "");
MsgType.心跳消息 = new MsgType(4, "心跳消息", "0400", "");
MsgType.终端电参数消息应答 = new MsgType(5, "终端电参数消息应答", "0500", "");
MsgType.终端上行通信参数获取应答 = new MsgType(6, "终端上行通信参数获取应答", "0600", "");
MsgType.终端通信网络信息应答 = new MsgType(7, "终端通信网络信息应答", "0700", "");
MsgType.调光模式获取应答 = new MsgType(8, "调光模式获取应答", "0800", "");
MsgType.获取策略请求应答 = new MsgType(9, "获取策略请求应答", "0900", "");
MsgType.上行升级报文 = new MsgType(100, "上行升级报文", "6400", "");
// 下行消息类型
MsgType.下行ACK = new MsgType(1001, "下行ACK", "E903", "0200");
MsgType.下行NACK = new MsgType(1002, "下行ACK", "EA03", "");
MsgType.开关灯调光 = new MsgType(1003, "开关灯调光", "EB03", "0100");
MsgType.设置策略 = new MsgType(1004, "设置策略", "EC03", "0300");
MsgType.配置还原 = new MsgType(1005, "配置还原", "ED03", "");
MsgType.获取调光模式 = new MsgType(1006, "获取调光模式", "EE03", "");
MsgType.设置调光模式 = new MsgType(1007, "设置调光模式", "EF03", "");
MsgType.复位重启 = new MsgType(1008, "设置调光模式", "F003", "");
MsgType.设置终端上行通信参数 = new MsgType(1009, "设置终端上行通信参数", "F103", "");
MsgType.获取终端电参数信息 = new MsgType(1010, "获取终端电参数信息", "F203", "");
MsgType.获取终端上行通信参数 = new MsgType(1011, "获取终端上行通信参数", "F303", "");
MsgType.获取终端通信网络信息 = new MsgType(1012, "获取终端通信网络信息", "F403", "");
MsgType.校时 = new MsgType(1013, "校时", "F503", "F5030600");
MsgType.接入应答 = new MsgType(1014, "接入应答", "F603", "0600");
MsgType.获取策略请求 = new MsgType(1015, "获取策略请求", "F703", "");
MsgType.下行升级报文 = new MsgType(1100, "下行升级报文", "4C04", "");

```