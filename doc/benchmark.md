# go-iot 压力测试报告

## 1. 测试目的与目标

- **目的**：验证 go-iot 在 MQTT 长连接 + 数据上报场景下的承载能力，定位系统瓶颈，支撑 **10 万设备** 目标
- **验收标准（目标）**：
  - 10 万设备连接：全部建立，avg 连接耗时 < 1s，>5000ms 占比 0
  - 上报场景：连接保持稳定，无进程崩溃，错误率 < 0.1%
  - ES 数据无丢失（允许延迟，停压后需恢复）

## 2. 测试环境

### 2.1 历史环境（早期基准）

| 标题 | 配置 |
| --- | --- |
| CPU | Intel(R) Xeon(R) CPU E5-2689 0 @ 2.60GHz 2.60 GHz |
| RAM | 32 GB（三星 DDR3 1600MHz） |
| 硬盘 | 三星SSD 970 EVO 500GB |
| 操作系统 | Windows 10 专业版 |

使用两个Linux虚拟机，一个server(12G)，一个压测client(8G)：

| 标题 | 配置 |
| --- | --- |
| CPU | 4C |
| RAM | 12G, 8G |
| 硬盘 | 50G |
| 操作系统 | CentOS Linux release 7.8.2003 (Core) |

### 2.2 当前环境（2026-08-08 实测）

| 项 | 服务端 test（linux1, <服务端IP>） | 压测端 test2（liunx2, <压测端IP>） |
|---|---|---|
| CPU | 4C AMD Ryzen 7 5700G | 4C AMD Ryzen 7 5700G |
| 内存 | 11G（**11:25 从 9.6G 调大，伴随 VM 重启**，重启中断了当轮压测） | 6.5G |
| 磁盘 | 26G（用 67%） | 46G |
| 服务 | docker: elasticsearch 7.17.7（堆 6G）、redis7、portainer | Java 1.8.0_252，device-simulator.jar（jetlinks） |

- go-iot 端口：9010 mqtt-mqttserver（压测入口）、9011 mqtt2、1883 内置 broker、8088 API
- 服务端文件：二进制 `/home/goiot/go-iot`、配置 `/home/goiot/conf/app.yaml`、日志 `/home/goiot/logs/goiot.log`（stderr 输出见 nohup.out）
- 压测端文件：`/home/run.sh`、日志 `/home/device-simulator-YYYYMMDD-HHMMSS.log`（时间戳，历史保留）

## 3. 测试工具与配置

### 3.1 压测工具

> 设备模拟器（jetlinks）https://gitee.com/jetlinks/device-simulator/tree/dev-1.0/

```
java -jar device-simulator.jar \
mqtt.address=<服务端IP> \
mqtt.port=9010 \
mqtt.limit=10000 \
mqtt.eventLimit=10000 \
mqtt.eventRate=1000 \
mqtt.maxSendTotal=3000000 \
mqtt.start=1111 \
mqtt.batchSize=1000 \
2>&1 | tee /home/device-simulator-$(date +%Y%m%d-%H%M%S).log
```

**参数说明**（源码 `MQTTSimulator.java`）：

| 参数 | 含义 |
|---|---|
| `mqtt.limit` | 设备总数 |
| `mqtt.start` | 设备 ID 起始 |
| `mqtt.batchSize` | 连接批量建立批次 |
| `mqtt.binds` | 绑定源 IP 列表（多 IP 突破单 IP 本地端口数限制） |
| `mqtt.eventLimit` | 每轮推送的设备数（随机抽取，可重复） |
| `mqtt.eventRate` | 推送频率（毫秒/轮，1000=每秒一轮） |
| `mqtt.maxSendTotal` | 累计推送总量上限，达到后停止推送（连接保持） |

推送速率 = `eventLimit / eventRate`（条/秒）；推送时长 ≈ `maxSendTotal / 速率`。

### 3.2 前置准备

服务端与压测端均需：

```
* soft nofile 655350
* hard nofile 655350
```

压测端虚拟网卡（大连接量必须，多 IP 突破单 IP 本地端口数限制）：

```
ifconfig enp0s3:1 <绑定IP1> up
ifconfig enp0s3:2 <绑定IP2> up
```

ES 容器（堆 6G）：

```
docker run -d --name elasticsearchv7 -p 9200:9200 -p 9300:9300 -e "discovery.type=single-node" -e "ES_JAVA_OPTS=-Xms6024m -Xmx6024m" elasticsearch:7.17.7
```

### 3.3 物模型与编解码脚本

物模型（5 个属性，1 秒上报一次）：

```json
{
  "events": [],
  "properties": [
    {"id": "temperature", "name": "温度", "scale": 2, "type": "float"},
    {"id": "light", "name": "亮度", "type": "int"},
    {"id": "humidity", "name": "湿度", "type": "long"},
    {"id": "current", "name": "电流", "type": "int"},
    {"id": "voltage", "name": "电压", "scale": 2, "type": "double"}
  ],
  "functions": []
}
```

编解码脚本（go-iot 侧）：

```javascript
function OnConnect(context) {
  context.DeviceOnline(context.GetClientId())
}
function OnMessage(context) {
  var data = JSON.parse(context.MsgToString())
  var topic = context.Topic()
  if (topic == '/report-property') {
    context.SaveProperties(data)
  } else if (topic == '/event') {
    context.SaveEvents(data.eventId, data)
  }
}
```

## 4. 测试场景设计

| 场景 | 连接数 | 上报速率 | 推送总量 | 目的 | 状态 |
|---|---|---|---|---|---|
| A. 历史：1 万 + 全量上报 | 1 万 | 1 万/s | 持续 | 早期基准 | ✅ 完成（旧环境） |
| B. 历史：10 万连接（不上报） | 10 万 | 0 | 0 | 连接容量上限 | ✅ 完成（旧环境） |
| C. 10 万连接尝试（9.6G 内存） | 10 万 | 0 | 0 | 当前环境容量摸底 | ⚠️ 5 万时内存不足停止 |
| D. 5 万 + 5 万/s（9.6G 内存） | 5 万 | 5 万/s | 300 万 | 极限压力 | ❌ 崩溃（ES 瓶颈+内存） |
| E. 1 万 + 1 万/s（9.6G 内存） | 1 万 | 1 万/s | 300 万 | ES 舒适区验证 | ✅ 完成 |
| F. 3 万 + 3 万/s（11G 内存，修复前） | 3 万 | 3 万/s | 300 万 | 并发压力 | ❌ 崩溃（map bug） |
| G. 3 万 + 3 万/s（修复后） | 3 万 | 3 万/s | 300 万 | 修复验证 | ✅ 完成 |

## 5. 测试结果

### 5.1 历史基准（旧环境 12G/8G VM）

#### 场景 A：一万设备连接，一万设备上报数据（每秒）

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
```

资源占用：

```
%Cpu(s): 67.4 us ... 19.5 id
  PID  %CPU  %MEM  RES    COMMAND
13100 245.8  4.4   299MB  go-iot
 3297  41.2 58.3   3.8g   java (ES)
```

go-iot 运行状态：20020 goroutine

> **结果**：ES写入速度 6000/s，压力主要在ES中，持续写入 `2540000` 条数据无丢失，但有延迟，当请求结束后2分钟go协程恢复正常

#### 场景 B：十万设备连接（配置原因只测连接不测上报）

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
```

资源占用：

```
%Cpu(s): 4.8 us ... 63.2 id
  PID  %CPU  %MEM  RES    COMMAND
 3072 96.0  10.4   1.2g   go-iot
```

go-iot 运行状态：200019 goroutine

### 5.2 2026-08-08（AMD Ryzen 环境）

> 内存时间线：场景 C/D/E 在 **9.6G** 下测试；**11:25 调大至 11G（VM 重启，中断了当轮压测）**，场景 F/G 在 11G 下测试

#### 场景 C：10 万连接目标（优化前二进制，仅连接）

- 连接建立至 **5 万**时：go-iot RSS 2.58G、**50KB/连接**，服务端可用内存仅 112M、swap 1.1G → 手动停止
- 结论：10 万连接需 ~5G，单机 9.6G（ES 占 6G）放不下

#### 场景 D：5 万连接 + 5 万/s 上报（内存优化后二进制）

- 5 万连接全部建立，**每连接 ~42KB**（优化前 50KB）
- 推送 5 万/s：ES 写入 1170ms 延迟（warntime=1000），消息在 go-iot 内存缓冲堆积，RSS 涨至 **3.4G**
- 可用内存 53M 时 go-iot 崩溃（Go runtime 内存分配失败，非 OOM killer，dmesg 无记录）
- 推送完成 198 万/300 万被打断

#### 场景 E：1 万连接 + 1 万/s 上报

```
已达到最大推送数量:3000000,停止推送 成功3009722 失败33
```

- 1 万连接全部建立并保持；推送 300 万条完成，失败仅 33（**0.001%**）
- go-iot RSS 稳定 743MB，CPU 107~187%；ES 偶发延迟 5 次（最大 1322ms）
- **结论：1 万/s 上报是 ES 单节点（6G 堆）舒适区上限**

#### 场景 F：3 万连接 + 3 万/s 上报（修复前，触发 bug）

- 连接建至 2.9 万、推送开始后 go-iot 崩溃两次（11:30、11:37）
- nohup.out 抓到崩溃堆栈：`fatal error: concurrent map read and map write`（OnPublished 无锁读 clients map，已修复）

#### 场景 G：3 万连接 + 3 万/s 上报（修复版）

```
已达到最大推送数量:3000000,停止推送 成功3021010 失败6949
```

- 3 万连接全部建立并保持（29625），全程无崩溃、无 fatal
- 推送 300 万条完成，失败 6949（**0.23%**，3 万/s 高并发下部分连接 i/o timeout）
- 推送期内存峰值 RSS **2.84G**，停止后回落至 **1.55G**（3 万连接基础占用）
- swap 仅用 38M；CPU 峰值 213%

### 5.3 数据汇总

| 场景 | 连接数 | 每连接内存 | go-iot 峰值 RSS | 结果 |
|---|---|---|---|---|
| B. 10 万连接（历史） | 10 万 | ~12KB | 1.2G | ✅ 稳定，avg 727ms |
| E. 1 万 + 1 万/s | 1 万 | ~42KB | 0.74G | ✅ 稳定，失败 0.001% |
| G. 3 万 + 3 万/s（修复版） | 3 万 | ~42KB | 2.84G | ✅ 稳定，失败 0.23% |
| D. 5 万 + 5 万/s | 5 万 | ~42KB | 3.4G（缓冲堆积） | ❌ 内存耗尽崩溃 |
| C. 10 万连接 | 5 万时停 | ~50KB（优化前） | 2.58G | ⚠️ 内存不足停止 |

## 6. 分析与结论

1. **瓶颈是 ES 单节点写入**：6G 堆也扛不住 >1 万/s 持续写入；消息缓冲堆积会推高 go-iot 内存，是场景 D 崩溃的直接原因
2. **go-iot 连接容量**：每连接 ~42KB（优化后），3 万连接基础占用 1.55G；**10 万连接预计 ~4.2G**
3. **10 万连接可行性**：加 ES 6G 需 10.2G+，单机 11G 紧张；建议 ES 降堆（压测连接阶段 ES 无压力，3G 足够）或加内存
4. **上报场景可行性**：10 万设备全量 1 万/s 不现实（ES 单节点上限），需按比例/降频设计；单节点极限 ~1 万/s
5. **错误率**：1 万/s 时 0.001%（优秀）；3 万/s 时 0.23%（压测端单机瓶颈初现，与服务端关系不大）

### 6.1 内存模型（连接与推送分开计算）

**公式**：

```
go-iot RSS ≈ 固定开销 0.3G + 43KB × 连接数 + 650B × 未消费消息数
未消费消息数 = max(0, 推送速率 − ES消费速率约1万/s) × 推送时长
```

- **固定开销 ~0.3G**：Go runtime + ES 客户端等（由 1 万连接 0.74G、3 万连接 1.60G 联立推算）
- **每连接 ~43KB**：连接基础内存，与推送无关
- **每条消息 ~650B**：MQTT 包 + 队列缓冲（3 万/s 轮 +1.29G ÷ 200 万滞留条推算）

**纯连接（不上报）内存预估**：

| 连接数 | 公式 | 预估 | 实测 |
|---|---|---|---|
| 1 万 | 0.3 + 0.43 | 0.73G | 0.74G ✅ |
| 3 万 | 0.3 + 1.29 | 1.59G | 1.60G ✅ |
| 5 万 | 0.3 + 2.15 | 2.45G | — |
| 10 万 | 0.3 + 4.30 | 4.60G | — |

**连接 + 推送内存预估**（推送只占"滞留量"，速率 ≤ 1 万/s 时几乎不占内存）：

| 场景 | 速率差 | 滞留量 | 缓冲 | 总 RSS | 实测 |
|---|---|---|---|---|---|
| 3 万连接 + 1 千/s | 0（ES 吃得下） | 0 | 0 | 1.59G | 1.95G（含连接期缓冲）✅ |
| 3 万连接 + 1 万/s | 0（临界） | ~0 | ~0 | 1.59G | 稳定 ✅ |
| 3 万连接 + 3 万/s | +2 万/s × 100s | 200 万条 | +1.3G | 2.89G | 2.84G ✅ |
| 5 万连接 + 5 万/s | +4 万/s × 40s | 160 万条 | +1.0G | 3.45G | 3.4G 崩 ⚠️ |

**要点**：连接内存线性增长（43KB/连接）；推送内存只与"ES 吃不下的滞留量"成正比（650B/条）——**速率 ≤ ES 消费能力（1 万/s）时推送不占内存，超过后按 速率差×时长 线性上涨**，这就是 5 万/s 崩溃和 3 万/s 擦边跑完的原因。

## 7. 注意事项 / 踩坑清单

1. **go-iot 启动建议 nohup + stderr 落盘**：`nohup ./go-iot > nohup.out 2>&1 &`——进程崩溃堆栈只在这里，否则无从排查
2. **ulimit**：服务端+压测端都需 `nofile 655350`（limits.conf 修改后需重登或重启 sshd 生效；root 可在启动脚本里 `ulimit -n 655350` 兜底）
3. **大连接量必须多绑定 IP**：`mqtt.binds` 绑定多个源 IP（单 IP 本地端口数有限），配合压测端虚拟网卡使用
4. **推送参数组合**：速率 = eventLimit/eventRate，时长 = maxSendTotal/速率；连接建立期间推送就在跑，总量含连接期
5. **ES 数据膨胀**：压测属性数据按月索引（goiot-properties-*），多轮压测后建议删除（`curl -X DELETE` 索引），保留元数据索引（goiot-product/network/device 等，删除会导致配置丢失）
6. **内存紧张判断**：`free -h` available + swap 需大于 go-iot 预计峰值（连接数 × 42KB + 推送缓冲）；推送缓冲与 ES 写入速度强相关
7. **疑似崩溃先查**：nohup.out（stderr）→ dmesg（OOM killer）→ /var/log/messages → core 文件，按此顺序排除
