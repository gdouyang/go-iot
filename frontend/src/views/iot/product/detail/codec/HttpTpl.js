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
      caption: 'context.GetHeader("key")',
      meta: 'OnMessage',
      value: 'var value = context.GetHeader("key")'
    },
    { caption: 'context.GetUrl()', meta: 'OnMessage', value: 'var url = context.GetUrl()' },
    {
      caption: 'context.GetQuery("key")',
      meta: 'OnMessage',
      value: 'var value = context.GetQuery("key")'
    },
    {
      caption: 'context.GetForm("key")',
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
    {
      caption: 'context.HttpRequest({method:"", url:"", data:{}, header:{}})',
      meta: 'OnMessage',
      value: 'var resp = context.HttpRequest({method:"", url:"", data:{}, header:{}})'
    },
    { caption: 'context.ReplyOk()', meta: 'OnMessage', value: 'context.ReplyOk()' },
    { caption: 'context.ReplyFail()', meta: 'OnMessage', value: 'context.ReplyFail("resaon")' },
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
    { caption: 'session.Response()', meta: 'session', value: 'session.Response("text")' },
    { caption: 'session.ResponseJSON("{}")', meta: 'session', value: 'session.ResponseJSON("{}")' },
    {
      caption: 'session.ResponseHeader("key", "value")',
      meta: 'session',
      value: 'session.ResponseHeader("key", "value")'
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
    {
      caption: 'context.HttpRequest({method:"", url:"", data:{}, header:{}})',
      meta: 'OnInvoke',
      value: 'var resp = context.HttpRequest({method:"", url:"", data:{}, header:{}})'
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
obj.demoCode = `// 检查在线状态
function OnStateChecker(context) {
  // unknown, online, offline;
  return "online"
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
  var data = JSON.parse(str);
  var msg = data.msg;
  var imei = msg.imei;
  var msgType = msg.type;
  // 处理上线消息
  if (ONLINE_OFFLINE == msgType) {
    var status = msg.status;
    if (ONLINE == status) {
      context.DeviceOnline(imei)
      return;
    } else if (OFFLINE == status) {
      // context.DeviceOnline(imei)
      return;
    }
  }

  var value = msg.value;
  var entity = new MsgEntity(value);
  var msgHeader = entity.header;
  var value1 = msgHeader.msgType.value;
  if (MsgType.心跳消息.equals(value1) || MsgType.上行接入请求.equals(value1)) {
    // 灯控器心中、上行接入时需要返回ack不然会断开连接
    var devOper = context.GetDeviceById(imei)
    if (devOper) {
      doAck(context, imei, msgHeader);
    }
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
    context.SaveProperties(json)
  }
}
// 物模型 -> 设备报文
function OnInvoke(context) {
  var message = context.GetMessage();
  if (!message.FunctionId) {
    throw new Error('只支持设备功能调用');
  }
  var functionId = message.FunctionId;
  var messageId = "1"//message.getMessageId();
  var invoke = new FunctionInvokeUtil();
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
  var resp = sendToOneNet(message.DeviceId, {'args': result});
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
    args = CmdUtil.resp(MsgType.接入应答);
  } else if (MsgType.心跳消息.equals(value1)) {
    args = CmdUtil.getCmdMsg(MsgType.下行ACK, msgHeader.pktNum.hex, null);
  }
  sendToOneNet(context, imei, {'args': args});
}

// 发送指令给移动平台
function sendToOneNet(context, imei, data) {
  // 获取页面上配置的地址与apiKey
  var url= context.GetConfig('apiAddress')
  var apiKey = context.GetConfig('apiKey')
  return context.HttpRequest({
    method: 'post',
    url: url + '/nbiot/execute?imei='+ imei +'&obj_id=3200&obj_inst_id=0&res_id=5505',
    data: data,
    headers: {'api-key': apiKey},
  })
}

function FunctionInvokeUtil() {
  // 开关
  this.switching = function(message) {
    var data = message.getInput("status").orElse("off");
    if ("on" == data) {
      return CmdUtil.getCmdMsg(MsgType.开关灯调光, CmdUtil.getHexStr(10, 1), null);
    } else {
      return CmdUtil.getCmdMsg(MsgType.开关灯调光, CmdUtil.getHexStr(0, 1), null);
    }
  }
  // 调光
  this.dimming = function(message) {
    var data = message.getInput("bright").orElse(0);
    return CmdUtil.getCmdMsg(MsgType.开关灯调光, CmdUtil.getHexStr(parseInt(data), 1), null);
  }
  // 设置策略
  this.strategy = function(message) {
    var data = message.getInput("strategy").orElse("");
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
      return CmdUtil.getCmdMsg(MsgType.设置策略, dateStr, strategyS.length * 3);
    }
  }
  // 校时
  this.timing = function () {
    return CmdUtil.resp(MsgType.校时);
  }
}

function CmdUtil() {
}
CmdUtil.resp = function(msgType) {
  var date = globe.formatDate(new Date().getTime(), "yy MM dd HH mm ss").split(' ');
  var dateStr = '';
  for(var i = 0; i < date.length; i++) {
    dateStr += CmdUtil.getHexStr(parseInt(date[i]), 1);
  }
  print(dateStr);
  return CmdUtil.getCmdMsg(msgType, dateStr, null);
}

CmdUtil.getCmdMsg = function(msgType, msgData, len) {
  var version = "0100";
  var msgLen = len !== null ? CmdUtil.getHexStr(len,2) : msgType.lenHex;
  var msgHead = version + CmdUtil.getPktNum() + msgType.codeHex + msgLen;
  var msgCrc16 = globe.toCrc16(msgHead + msgData);

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

CmdUtil.pktNum = 0;
CmdUtil.getPktNum = function() {
  CmdUtil.pktNum++;
  return CmdUtil.getHexStr(CmdUtil.pktNum, 2);
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
    if (endIndex > this.body.length()) {
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
`
export default obj
