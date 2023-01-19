# go-iot

#### 介绍
使用go实现的iot接入系统，以`物模型`为主体用来对接不同厂商的设备来实现统一接入的目的

> 项目参考了https://github.com/jetlinks/jetlinks-community，https://github.com/megaease/easegress

#### 功能目录
- 产品管理
- 设备管理
- 规则引擎
- 通知管理
- 设备告警
- 角色管理
- 用户管理
- 系统设置

#### 网络协议
- tcp server
- tcp client
- mqtt broker
- mqtt client
- http server
- websocket server
- modbus tcp

#### 使用说明

1. ide使用vs code
2. go版本1.19.1
3. go mod tidy

```
docker run --name mysql -p 3306:3306 -e MYSQL_ROOT_PASSWORD=root -d mysql:8

docker run -d --name elasticsearchv7 -p 9200:9200 -p 9300:9300 -e "discovery.type=single-node" -e "ES_JAVA_OPTS=-Xms1024m -Xmx1024m" elasticsearch:7.17.7

docker run --name redis6 -d -it -p 6379:6379 redis:6
```

#### 压力测试
主机配置：
| 标题 | 配置 |
| --- | --- |
| CPU | Intel(R) Xeon(R) CPU E5-2689 0 @ 2.60GHz   2.60 GHz |
| RAM | 32 GB（三星 DDR3 1600MHz）|
| 硬盘 | 三星SSD 970 EVO 500GB |
| 操作系统 | Windows 10 专业版 |

使用两个Linux虚拟机，一个server，一个client，虚拟机配置：
| 标题 | 配置 |
| --- | --- |
| CPU | 4C |
| RAM | 12G, 8G |
| 硬盘 | 50G |
| 操作系统 | CentOS Linux release 7.8.2003 (Core) |

ES内存设置为6G

修改最大文件数在`/etc/security/limits.conf`中追加以下配置
```
* soft nofile 65535
* hard nofile 65535
```
docker使用默认配置

- MQTT Broker测试10000设备连接，每隔1秒上报5个属性

| id | 名称 | 类型 |
| --- | --- | --- |
| temp | 温度 | float |
| light | 亮度 | int |
| test1 | long测试 | long |
| current | 电流 | int |
| fre | 功率 | double |

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
```
CPU、内存使用情况
```shell
top - 15:29:57 up 10:34,  2 users,  load average: 4.69, 1.75, 0.83
Tasks: 140 total,   1 running, 139 sleeping,   0 stopped,   0 zombie
%Cpu(s): 67.4 us,  3.1 sy,  0.0 ni, 19.5 id,  0.4 wa,  0.0 hi,  9.7 si,  0.0 st
KiB Mem : 83.2/6831120  [|||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||         ]
KiB Swap:  0.6/3145724 [|                                                            ]

  PID USER      PR  NI    VIRT    RES    SHR S  %CPU %MEM     TIME+ COMMAND                                                                      
13100 root      20   0 1023088 299000  12652 S 245.8  4.4   3:38.39 go-iot                                                                       
 3297 1000      20   0 7474404   3.8g  38236 S  41.2 58.3  14:30.41 java
```
go-iot运行状态
```
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

ES写入速度 5000/s，压力主要在ES中，持续写入`2540000`条数据无丢失，但有延迟，当中断请求结束后2分钟go协程恢复正常