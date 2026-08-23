# go-iot 压力测试报告

> 基准版本：2026-08-23（S1：1 万连接；S2：3 万连接；S3：5 万连接，均为 1 万条/s × 51 分钟 × 3000 万条）

## 1. 测试目的与目标

- **目的**：验证 go-iot 在 MQTT 长连接 + 数据上报场景下的承载能力，定位系统瓶颈，支撑 **10 万设备** 目标
- **验收标准（目标）**：
  - 10 万设备连接：全部建立，avg 连接耗时 < 1s，>5000ms 占比 0
  - 上报场景：连接保持稳定，无进程崩溃，错误率 < 0.1%
  - ES 数据无丢失（允许延迟，停压后需恢复）

## 2. 测试环境

宿主机：Windows 10 专业版 + VirtualBox，AMD Ryzen 7 5700G（8C16T），32G 内存

| 项 | 服务端 linux1（192.168.31.197） | 压测端 linux2（192.168.31.198） |
|---|---|---|
| CPU | 4C | 4C |
| 内存 | 13G | 6.5G |
| 磁盘 | 26G | 46G |
| 系统 | CentOS 7.8 | CentOS 7.8 |
| 服务 | docker: elasticsearch 7.17.7（堆 6G）、redis7、portainer | Java 1.8.0_252，device-simulator.jar（jetlinks） |

- go-iot 端口：9010 mqtt-mqttserver（压测入口）、9011 mqtt2、1883 内置 broker、8088 API
- 服务端文件：二进制 `/home/goiot/go-iot`、配置 `/home/goiot/conf/app.yaml`、日志 `/home/goiot/logs/goiot.log`（stderr 输出见 nohup.out）
- 压测端文件：`/home/run.sh`、日志 `/home/device-simulator-YYYYMMDD-HHMMSS.log`（时间戳，历史保留）

## 3. 测试工具与配置

### 3.1 压测脚本（run.sh 当前值）

> 设备模拟器（jetlinks）https://gitee.com/jetlinks/device-simulator/tree/dev-1.0/

```bash
#!/usr/bin/env bash
# 先清理残留进程/连接，再启动新一轮压测
ulimit -n 655350
sysctl -w net.ipv4.tcp_tw_reuse=1 > /dev/null 2>&1

# 清理残留压测进程（kill 后最多等 10s，强杀兜底）
# 等待旧连接释放（ESTABLISHED 到 9010 清零或超时 30s）

java -jar device-simulator.jar \
mqtt.address=192.168.31.197 \
mqtt.port=9010 \
mqtt.limit=10000 \
mqtt.eventLimit=10000 \
mqtt.eventRate=1000 \
mqtt.maxSendTotal=30000000 \
mqtt.start=1111 \
mqtt.batchSize=1000 \
mqtt.binds=192.168.31.50,192.168.31.51,192.168.31.52 \
mqtt.timeout=300 \
2>&1 | tee /home/device-simulator-$(date +%Y%m%d-%H%M%S).log
```

> **java 参数行必须 `\` 续行，否则 batchSize/binds 不生效**

**参数说明**（源码 `MQTTSimulator.java`）：

| 参数 | 含义 |
|---|---|
| `mqtt.limit` | 设备总数（连接数） |
| `mqtt.start` | 设备 ID 起始 |
| `mqtt.batchSize` | 连接批量建立批次 |
| `mqtt.binds` | 绑定源 IP 列表（多 IP 突破单 IP 本地端口数限制） |
| `mqtt.eventLimit` | 每轮推送的设备数（随机抽取，可重复） |
| `mqtt.eventRate` | 推送频率（毫秒/轮，1000=每秒一轮） |
| `mqtt.maxSendTotal` | 累计推送总量上限，达到后停止推送；**进程不会自动退出，需手动 kill** |
| `mqtt.timeout` | 连接超时（秒），大连接量建议调大 |

推送速率 = `eventLimit / eventRate`（条/秒）；推送时长 ≈ `maxSendTotal / 速率`。

### 3.2 前置准备

服务端与压测端均需：

```
* soft nofile 655350
* hard nofile 655350
```

压测端虚拟网卡（多 IP 突破单 IP 本地端口数限制）：

```bash
ifconfig enp0s3:1 192.168.31.50 up
ifconfig enp0s3:2 192.168.31.51 up
ifconfig enp0s3:3 192.168.31.52 up
```

> **虚拟 IP 重启即失（非持久化）**：已写入 `/etc/rc.local` 自动恢复（幂等）。VM 重建需重做；排查时先 `ifconfig enp0s3:1 | grep inet` 确认 IP 在

ES 容器（堆 6G）：

```bash
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
| S1 | 1 万 | 1 万/s | 3000 万 | 稳态承载验证：ES 单节点消费上限贴边长跑 | ✅ 完成 |
| S2 | 3 万 | 1 万/s | 3000 万 | 连接规模扩展验证：3 万连接 + ES 上限速率 | ✅ 完成 |
| S3 | 5 万 | 1 万/s | 3000 万 | 连接规模扩展验证：5 万连接 + 连接风暴考验 | ✅ 完成 |
| S4（待测） | 10 万 | ≤1000/s | — | 目标场景：10 万长连接 + 低频上报 | 待执行 |

## 5. 测试结果（2026-08-23）

### 5.1 场景 S1 总览

| 项 | 结果 |
|---|---|
| 时长 | 11:19:51 ~ 12:10:52，约 **51 分钟** |
| 连接 | 10000/10000 成功，全程 established 恒为 10000，**0 掉线** |
| 推送 | **成功 3000 万，失败 0**（无 reset / assign / 超时） |
| 速率 | 恒定 10000 条/s，每秒整 1 万无波动 |
| 进程崩溃 | 无（nohup.out 干净） |

### 5.2 连接阶段（压测端日志）

```
create mqtt client: 10000 ok

max : 3008ms
min : 581ms
avg : 1728ms

> 5000ms : 0(0.00%)
> 2000ms : 3149(31.49%)
```

### 5.3 服务端资源指标（linux1，go-iot）

| 指标 | 数值 | 备注 |
|---|---|---|
| RSS | 925~969 MB，**50 分钟零增长** | 与内存模型预估吻合（见 §6.1） |
| CPU | ~135%（约 1.35 核 / 4C） | 稳定 |
| 线程数 | 14 | — |
| 主机内存 | available 5.2~5.3G，**swap 用量 0** | 充足 |
| load average | 0.22 / 0.44 / 1.22 | 很低 |

**带宽**（enp0s3，约 20s 双采样）：

| 方向 | 速率 | 包速率 | 说明 |
|---|---|---|---|
| 入站 RX | **~1.51 MB/s（12.1 Mbps）** | ~9600 pkt/s | 与 1 万条/s 对应，折合 ~150B/条（MQTT+TCP 开销） |
| 出站 TX | ~0.58 MB/s（4.6 Mbps） | ~9600 pkt/s | PUBACK 等 |

千兆网卡利用率 ~1.6%，**带宽非瓶颈**。

### 5.4 ES 写入

- 全程仅 2 条延迟 WARN（1395ms / 1932ms，warntime=1000），量级健康
- 属性索引 `goiot-properties-mqttserver-202608` 落库增速 **~1.06 万 docs/s**（40.4M→43.9M），与推送速率吻合，无积压
- 压测端 Java：CPU 57.4%、RSS 561MB、1023 线程、TX ~1.84 MB/s——单机发压余量充足

### 5.5 场景 S2：3 万连接 + 1 万条/s（12:47~13:39，51 分钟）

| 项 | 结果 |
|---|---|
| 推送 | **3000 万成功 / 0 失败**，恒定 1 万条/s |
| 连接 | 30000 全程保持零掉线；avg 1072ms / max 3009ms，>5000ms 为 0 |
| go-iot RSS | **全程 2.19~2.34G 稳定**（模型预估 2.28G ✅），停推后回落至 1.78G |
| CPU | 稳态 80~106%（约 1 核/4C），load 峰值 2.97 后回落 |
| 主机内存 | available 3.7~4.4G，swap 未动用 |
| ES 延迟 WARN | 约 19 条（1.0~2.2s），呈「间歇尖峰+自行回落」形态，无持续恶化、无堆积 |

**与 S1 的对比发现**：本轮 3 万连接建立耗时（avg 1072ms）反而优于 S1 的 1 万连接（1728ms）——说明连接耗时主要受压测端/网络瞬时状态影响，与服务端连接规模无关。

### 5.6 场景 S3：5 万连接 + 1 万条/s（14:16~15:08，51 分钟）

监控采样详见 devops `压测监控-20260823-5万连接.md`。

| 项 | 结果 |
|---|---|
| 推送 | **3000 万成功 / 0 失败**，恒定 1 万条/s |
| 连接 | 49928 全程保持（72 台为风暴期重连失败残留）；建连统计被风暴事件污染（avg 2802ms / max 48887ms / >5000ms 7.51%），干净基线参考 S1/S2 |
| go-iot RSS | 稳态 **3.48~3.85G**（模型预估 3.6G ✅），尾段短暂跳升 4.60G 后停推即消化；停推后回落 2.68G |
| CPU | 稳态 90~105%，load 峰值 4~5（4C 饱和）后回落 |
| 主机内存 | available 最低 1.7G（尾段），收尾恢复 3.7G |
| ES 延迟 WARN | 本轮累计 34 条，两次尖峰（最大 **12.7s**）——5 万连接下 ES 余量已耗尽 |

#### ⚠️ 关键事件：连接风暴导致临时端口耗尽（14:17）

- **现象**：连接建至 3.4 万时被服务端批量拒连（`CONNECTION_REFUSED_NOT_AUTHORIZED`），压测端自动重连刷屏
- **根因链**：连接风暴 → 每设备上线触发 ES/Redis 认证查询，go-iot 对 ES 高频短连接 → TIME_WAIT 堆积至 ~4.5 万 → 本地临时端口池（`ip_local_port_range` 默认 32768~60999，约 2.8 万个）耗尽 → 新出站连接 `cannot assign requested address` → 认证失败拒连
- **自愈**：TIME_WAIT 60s 寿命到期老化后端口释放，重连全部成功，未影响后续稳态
- **S4 前置必办**：
  1. 服务端 `sysctl -w net.ipv4.tcp_tw_reuse=1` + `net.ipv4.ip_local_port_range=10240 65535`（并持久化到 `/etc/sysctl.conf`；压测端 run.sh 已有 tcp_tw_reuse=1，服务端此前遗漏）
  2. 排查 go-iot ES 客户端短连接高频建断成因，评估连接复用改造

## 6. 分析与结论

1. **1~5 万连接 + 1 万条/s 均可稳态承载**：S1/S2/S3 各 51 分钟 × 3000 万条均零推送失败、零崩溃；RSS 稳态符合模型，停推后正常回落。连接规模每上一个台阶的代价是**内存线性增长 + 对基础设施（端口池/ES 余量）的要求变严**
2. **该速率正好贴着 ES 单节点消费上限（~1 万/s）**：S1/S2 仅偶发秒级 WARN；S3 出现 12.7s 尖峰、余量耗尽——**5 万连接是当前 ES 单节点 + 1 万/s 组合的实际边界**
3. **CPU 与带宽均有大量余量**：go-iot 约 1~1.35 核/4C，网卡 <2%；当前容量限制主要在**内存（每连接 ~66KB）与 ES 写入能力**
4. **10 万目标（场景 S4）前置条件**：按每连接 66KB 外推，10 万连接 go-iot 约 5.7~7G + ES 6G ≈ 12~13G，13G 单机处于临界；上报需降频（≤1000/s）。**必须先完成 S3 暴露的两项整改**（服务端 tcp_tw_reuse/端口范围持久化、ES 客户端连接复用排查），否则连接风暴会复发且更严重

### 6.1 内存模型（连接与推送分开计算）

**公式**：

```
go-iot RSS ≈ 固定开销 0.3G + 每连接内存 × 连接数 + 650B × 未消费消息数
未消费消息数 = max(0, 推送速率 − ES消费速率约1万/s) × 推送时长
```

- **固定开销 ~0.3G**：Go runtime + ES 客户端等基础占用
- **每条消息大小分层账**（以本轮物模型 5 属性报文为例）：

  | 阶段 | 大小 | 说明 |
  |---|---|---|
  | 设备 → go-iot 线上报文 | **75B JSON / ~96B MQTT 报文 / ~151B 含 TCP/IP 及 ACK 均摊** | S1 实测入站带宽 ÷ 条数吻合 |
  | go-iot 落 ES 的文档 | **~133B** | 注入 `createTime`（~27B）与 `deviceId`（4~5 位 ID 时 ~18B）后膨胀 ~1.8 倍，属正常元数据注入 |
  | ES 磁盘占用（压缩后） | **~67B/条** | S2 实测：goiot-properties 索引 4040 万条 / 2.6GB；列存+压缩后反而小于原文 |

- **每条滞留消息 ~650B**：历史高滞留场景反推的经验值。滞留的是原始消息（75B），createTime/deviceId 是写 ES 时才拼接、不占缓冲；650B 与线上体积的差额全部是 Go 堆内开销（MQTT 消息对象包装、队列/bulk 缓冲结构、GC 对齐等），随真实报文大小线性变化
- **验证点**：
  - S1：1 万连接 + 1 万条/s（≤ES 消费能力）→ 未消费=0 → RSS 稳定 0.93~0.97G 零增长 ✅；反推每连接 ~(0.96−0.3)G ÷ 1 万 ≈ **66KB**（含页表/GC 元数据等非线性开销）
  - S2：3 万连接 + 1 万条/s → 模型预估 0.3G + 66KB×3 万 ≈ 2.28G，实测 RSS 全程稳定 2.19~2.34G ✅，跨规模验证通过
  - S3：5 万连接 + 1 万条/s → 模型预估 ≈ 3.6G，实测稳态 3.48~3.85G ✅；尾段短暂 4.60G 为滞留波动，停推即消化

**要点**：连接内存随连接数线性增长；推送内存只与"ES 吃不下的滞留量"成正比——**速率 ≤ ES 消费能力（1 万/s）时推送不占内存，超过后按 速率差×时长 线性上涨**。规划上报速率时以此为红线。

## 6.2 系统稳定性评估（S1~S3 累计约 2.5 小时、9000 万条消息）

| 维度 | 表现 | 评级 |
|---|---|---|
| 进程存活 | **同一进程持续在线**：PID 不变（6513，11:18 启动），贯穿 S1~S3 全程及连接风暴、零重启零崩溃零 panic | ✅ 优 |
| 消息可靠性 | 累计 9000 万条推送**零失败**；ES 延迟但无丢失 | ✅ 优 |
| 连接保持 | 稳态期零掉线（S1 恒 1 万、S2 恒 3 万、S3 恒 49928） | ✅ 优 |
| 内存健康 | RSS 严格符合模型、稳态零增长、停推即回落——三次跨规模验证无泄漏 | ✅ 优 |
| 过载自愈 | 端口耗尽风暴 60s 自愈；ES 延迟尖峰自行回落；Redis 断连风暴自愈 | ✅ 良 |
| ES 写入余量 | S3 已耗尽余量（12.7s 尖峰），5 万连接是当前组合的边界 | ⚠️ 需扩容 |

- **go-iot 本身**：稳定性良好。最有说服力的证据是 S3 连接风暴期间——认证依赖的 ES/Redis 完全不可达的情况下，进程未崩溃、无异常资源增长，端口释放后立即恢复服务
- **基础设施配置是当前短板**（均已知可修）：① 服务端 tcp_tw_reuse/端口范围未调 → 风暴期端口耗尽；② ES 单节点上限 ~1 万/s → 5 万连接已贴死；③ go-iot ES 客户端疑似短连接高频建断，需排查改造
- **待验证项**：全体同时断连会瞬时打爆 Redis（clearClusterId 集中写），虽已自愈，10 万规模下的大面积掉线场景需专项评估

**结论**：连续三轮加压、含一次基础设施故障，go-iot 无一失败，稳定性底子良好；距 10 万目标的差距不在软件稳定性，而在两项配置整改 + ES 容量规划。

## 7. 注意事项 / 踩坑清单

1. **go-iot 启动必须 nohup + stderr 落盘**：`cd /home/goiot && nohup ./go-iot > /dev/null 2> nohup.out &`——进程崩溃堆栈只在 nohup.out，goiot.log 里没有
2. **ulimit**：服务端+压测端都需 `nofile 655350`
3. **大连接量必须多绑定 IP**：`mqtt.binds` 绑定多个源 IP，配合压测端虚拟网卡；虚拟 IP 已持久化到 `/etc/rc.local`
4. **java 参数行必须 `\` 续行**，否则 batchSize/binds 静默失效
5. **达到 maxSendTotal 后进程不会自动退出**：持续打印“已达到最大推送数量”，收尾需手动 kill
6. **"create mqtt client: N ok” 是 job 进度非真实连接数**：绑定失败时 ok 数会虚高，判读以服务端 `ss -tn` 连接数为准
7. **ES 数据膨胀**：压测属性数据按月索引（goiot-properties-*），多轮压测后建议删除；元数据索引（goiot-product/network/device 等）不可删，否则配置丢失
8. **全体同时断连会瞬时打爆 Redis**：强杀压测进程导致设备同时离线，离线清理集中打 Redis 出现大量超时（已自愈）；正常收尾建议 kill 前确认影响或接受自愈
