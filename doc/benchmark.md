#### 压力测试
##### 主机配置：
| 标题 | 配置 |
| --- | --- |
| CPU | Intel(R) Xeon(R) CPU E5-2689 0 @ 2.60GHz   2.60 GHz |
| RAM | 32 GB（三星 DDR3 1600MHz）|
| 硬盘 | 三星SSD 970 EVO 500GB |
| 操作系统 | Windows 10 专业版 |

##### 虚拟机配置：使用两个Linux虚拟机，一个server(12G)，一个压测client(8G)
| 标题 | 配置 |
| --- | --- |
| CPU | 4C |
| RAM | 12G, 8G |
| 硬盘 | 50G |
| 操作系统 | CentOS Linux release 7.8.2003 (Core) |


- 使用docker运行Redis和elasticsearch，Reids使用容器默认配置，elasticsearch内存设置为6G
```
docker run -d --name elasticsearchv7 -p 9200:9200 -p 9300:9300 -e "discovery.type=single-node" -e "ES_JAVA_OPTS=-Xms6024m -Xmx6024m" elasticsearch:7.17.7
```

使用`ulimit -n 655350`修改最大文件句柄数或在`vim /etc/security/limits.conf`中追加以下配置
```
* soft nofile 655350
* hard nofile 655350
```
客户端
```
# 设置最大句柄
ulimit -n 655350
# 增加虚拟网卡
sudo ifconfig enp0s3:1 192.168.31.51 up
sudo ifconfig enp0s3:2 192.168.31.52 up

# 运行
java -jar device-simulator.jar mqtt.address=192.168.31.197 mqtt.port=9010 mqtt.limit=10000 mqtt.eventLimit=10000 mqtt.start=1111 mqtt.batchSize=1000 mqtt.binds=192.168.31.50,192.168.31.51,192.168.31.52
```
> 设备模拟器 https://gitee.com/jetlinks/device-simulator/tree/dev-1.0/

- MQTT Broker测试一万设备连接，每隔1秒上报5个属性
物模型
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
      "unit": null,
      "type": "float"
    },
    {
      "id": "light",
      "name": "亮度",
      "expands": {
        "readOnly": null
      },
      "description": null,
      "unit": null,
      "type": "int"
    },
    {
      "id": "humidity",
      "name": "湿度",
      "expands": {
        "readOnly": null
      },
      "description": null,
      "unit": null,
      "type": "long"
    },
    {
      "id": "current",
      "name": "电流",
      "expands": {
        "readOnly": null
      },
      "description": null,
      "unit": null,
      "type": "int"
    },
    {
      "id": "voltage",
      "name": "电压",
      "expands": {
        "readOnly": null
      },
      "description": null,
      "scale": 2,
      "unit": null,
      "type": "double"
    }
  ],
  "functions": []
}
```
编解码脚本
```javascript
function OnConnect(context) {
  context.DeviceOnline(context.GetClientId())
}

function OnMessage(context) {
  console.log("OnMessage: " + context.MsgToString())
  var data = JSON.parse(context.MsgToString())
    if (data.name == 'reply') {
        context.ReplyOk()
        return
    }
  var topic = context.Topic()
  if (topic == '/report-property') {
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

##### 一万设备连接，一万设备上报数据（每秒）
```
create mqtt client: 9000 ok
create mqtt client: 10000 ok

max : 1571ms
min : 525ms
avg : 956ms

> 5000ms : 0(0.00%)
> 2000ms : 0(0.00%)
> 1000ms : 2935(29.35%)
> 500ms : 7065(70.65%)
> 200ms : 0(0.00%)
> 100ms : 0(0.00%)
> 10ms : 0(0.00%)

# CPU、内存使用情况
top - 15:29:57 up 10:34,  2 users,  load average: 4.69, 1.75, 0.83
Tasks: 140 total,   1 running, 139 sleeping,   0 stopped,   0 zombie
%Cpu(s): 67.4 us,  3.1 sy,  0.0 ni, 19.5 id,  0.4 wa,  0.0 hi,  9.7 si,  0.0 st
KiB Mem : 83.2/6831120  [|||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||         ]
KiB Swap:  0.6/3145724 [|                                                            ]

  PID USER      PR  NI    VIRT    RES    SHR S  %CPU %MEM     TIME+ COMMAND                                                                      
13100 root      20   0 1023088 299000  12652 S 245.8  4.4   3:38.39 go-iot                                                                       
 3297 1000      20   0 7474404   3.8g  38236 S  41.2 58.3  14:30.41 java

# go-iot运行状态
Count	Profile
422	allocs
0	block
0	cmdline
20020	goroutine
422	heap
0	mutex
0	profile
8	threadcreate
0	trace
```
- 测试结果

> ES写入速度 6000/s，压力主要在ES中，持续写入`2540000`条数据无丢失，但有延迟，当请求结束后2分钟go协程恢复正常

##### 十万设备连接（由于配置原因只测试了连接没有测试上报）
```
create mqtt client: 98000 ok
create mqtt client: 99000 ok
create mqtt client: 100000 ok

max : 3644ms
min : 488ms
avg : 727ms

> 5000ms : 0(0.00%)
> 2000ms : 2010(2.01%)
> 1000ms : 6335(6.33%)
> 500ms : 90038(90.04%)
> 200ms : 1617(1.62%)
> 100ms : 0(0.00%)
> 10ms : 0(0.00%)

# CPU、内存使用情况
%Cpu(s):  4.8 us, 10.9 sy,  0.0 ni, 63.2 id,  0.0 wa,  0.0 hi, 21.1 si,  0.0 st
KiB Mem : 79.3/12184576 [||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||                    ]
KiB Swap:  0.0/3145724  [                                                                                                    ]

  PID USER      PR  NI    VIRT    RES    SHR S  %CPU %MEM     TIME+ COMMAND                                                                                                                                                                
 3072 root      20   0 2213244   1.2g   9928 S  96.0 10.4   3:41.42 go-iot

# go-iot运行状态
Count	Profile
210	allocs
0	block
0	cmdline
200019	goroutine
210	heap
0	mutex
0	profile
9	threadcreate
0	trace

```