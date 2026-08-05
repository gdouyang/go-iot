# 集群使用说明

本文描述 go-iot **多节点接入集群** 的配置、路由模型、运维注意点。  
实现位置：`pkg/cluster`、`pkg/core/device_gateway.go`、`internal/app` 装配。

---

## 1. 适用场景与能力边界

### 适合

- 多实例水平扩展 **设备接入**（TCP/MQTT 等长连接）
- 管理 API / 规则引擎在 **会话所在节点** 下发功能调用
- 产品、规则等 **每节点都有内存运行态** 的配置同步

### 不提供（当前版本）

- 连接会话热迁移（节点宕机后设备需 **重连** 到存活节点）
- 功能调用的强一致事务 / 跨节点两阶段提交
- eventpush（WebSocket 事件推送）的可靠投递（见 §6）
- 自动服务发现（节点列表仍以配置 `hosts` 为引导）

集群依赖 **共享 Redis**（设备运行态、`ClusterId`、离线命令队列、eventpush Pub/Sub）与 **共享 ES**（元数据）。各节点须连同一套 Redis/ES。

---

## 2. 配置

### 2.1 YAML（`conf/app.yaml`）

字段与 `pkg/option.Cluster` 一致，开关统一为 **`enabled`**：

```yaml
cluster:
  # 是否启用集群（推荐；旧键 enable 仅在未设置 enabled 时兼容）
  enabled: false
  # 本节点唯一名称（写入设备 ClusterId，转发时按此匹配）
  name: node-a
  # 本节点分片序号，须在 [0, 节点数) 内且全集群互不重复
  index: 0
  # 集群内部 HTTP 鉴权 token（Header: x-cluster-request）
  token: "change-me-to-a-long-secret"
  # 本节点对外可被其它节点访问的 API 根地址（无尾斜杠路径）
  url: http://192.168.1.10:8088
  # 集群内所有节点 URL，逗号分隔（可含本机；本机会被过滤）
  hosts: http://192.168.1.10:8088,http://192.168.1.11:8088
```

### 2.2 命令行（与 yaml 可叠加）

| Flag | 含义 |
|------|------|
| `--cluster.enabled` | 是否启用（绑定 `Cluster.Enabled`） |
| `--cluster.name` | 节点名 |
| `--cluster.index` | 分片 index |
| `--cluster.token` | 通讯 token |
| `--cluster.url` | 本机 URL |
| `--cluster.hosts` | 主机列表 |

### 2.3 两节点示例

**节点 A**

```yaml
api-addr: :8088
cluster:
  enabled: true
  name: node-a
  index: 0
  token: "shared-secret"
  url: http://192.168.1.10:8088
  hosts: http://192.168.1.10:8088,http://192.168.1.11:8088
```

**节点 B**

```yaml
api-addr: :8088
cluster:
  enabled: true
  name: node-b
  index: 1
  token: "shared-secret"   # 必须与 A 相同
  url: http://192.168.1.11:8088
  hosts: http://192.168.1.10:8088,http://192.168.1.11:8088
```

### 2.4 配置注意

1. **`name` 全局唯一**，稳定勿改：设备在线时会把 `ClusterId` 写成该 name。  
2. **`index` 互不重复**，且满足 `0 <= index < 节点数`。节点数 = 配置中 peer 数 + 1（本机）。  
3. **`url` 必须是其它节点能访问到的地址**（勿写 `127.0.0.1` 除非真在同机多进程且端口不同）。  
4. **`token` 全员一致**；内部接口靠该 Header 鉴权，勿用弱口令。  
5. 改 `hosts` / 扩缩容后 **分片会变**（`crc32(deviceId) % Size`），出站 client 负责节点可能迁移，需重启或重新 Connect。  
6. 单机：`enabled: false`（或省略集群段），不要依赖 `ClusterId` 路由。

---

## 3. 架构与三类路由

```text
                    ┌─────────────────────────────────────┐
  设备长连接         │  节点本地 Session（内存）              │
  上线               │  Redis: device.clusterId = 本机 name  │
                    └─────────────────────────────────────┘

  ┌─ ① 设备亲和（有会话）────────────────────────────────┐
  │ 功能调用 / 断连 / 连接信息 / 部分设备查询               │
  │ → 读 ClusterId → 本机执行 或 HTTP 点对点到该节点        │
  └──────────────────────────────────────────────────────┘

  ┌─ ② 分片（无会话 / 出站 client）──────────────────────┐
  │ Connect、启动恢复 netclient                            │
  │ → crc32(deviceId) % Size == index 的节点负责           │
  │ → 只投递该节点，不再 Broadcast 碰运气                   │
  └──────────────────────────────────────────────────────┘

  ┌─ ③ 全员复制（配置 / 规则运行态）──────────────────────┐
  │ 产品发布/停用、规则启停、部分设备激活类 API              │
  │ → BroadcastInvoke：每个存活节点本地都执行一遍           │
  │ → product_controller 等仍写 Broadcast 是正确语义      │
  └──────────────────────────────────────────────────────┘
```

### 3.1 功能调用（推荐路径）

业务应走统一入口，避免手写 `if cluster`：

```text
core.GetDeviceGateway().Invoke(ctx, msg, InvokeOptions{...})
// 或规则侧：core.DoCmdInvokeCluster(msg)  （内部已转 Gateway）
```

| 条件 | 行为 |
|------|------|
| 未开集群 / ClusterId 空 / 等于本机 | 本机 `DoCmdInvoke` |
| ClusterId 为其它节点 | `POST {peer}/api/cluster/cmd-invoke`（带 token） |
| `OfflineCache` 且已离线 | 本机写 **共享 Redis** 离线队列（不必 RPC） |
| `ForceLocal` | 强制本机（集群内部入口使用，防路由环） |
| 对端失败 | **返回错误**，不再回落本机乱执行 |

内部协议：

- 路径：`POST /api/cluster/cmd-invoke`
- Header：`x-cluster-request: <token>`
- Body：`FuncInvoke` JSON  
- 对端 `ForceLocal` 执行

### 3.2 管理 HTTP 按设备转发

设备会话相关 API 使用：

```text
cluster.ProxyOrLocal(deviceId, request)
```

已覆盖例如：设备详情实时状态、连接信息、Disconnect。  
转发为 **克隆原请求** 到对端同路径（仍带用户会话场景下的原 Header + 集群 token）。

### 3.3 出站 Connect（分片单点）

```text
cluster.ResolveShardOwner(deviceId)
  → 本机：connectClientDevice
  → 远端：SingleInvoke(peer.Name)  // 单节点，非广播
```

对端 `Name`/`Index` 依赖 keepalive 回填；未完成探测时可能报 *shard owner not found / has no name yet*，等数秒或检查网络与 token。

### 3.4 广播（产品 / 规则）

`product_controller`、`rule_controller` 等中的：

```go
if ctl.IsNotClusterRequest() {
    // 本机持久化 / 改状态
    cluster.BroadcastInvoke(ctl.Request)
}
```

**应保留。** 语义是「用户入口触发 → 每个节点执行同一操作」，与「设备连在哪台」无关。  
`IsNotClusterRequest()` 防止集群转发后再广播造成环与重复。

---

## 4. 节点 Registry 与内部 API

### 4.1 成员关系

- 启动时用 `hosts` 建立 peer URL 列表。  
- 每 5s 向对端 `POST /api/cluster/keepalive`，请求体为本节点信息。  
- 响应体带对端 `name`/`index`/`url`，写入 **Registry**（`Alive`、`LastSeen`）。  
- `SingleInvoke` / 功能调用 RPC / 分片解析均查 Registry。

### 4.2 运维接口

| 方法 | 路径 | 鉴权 | 说明 |
|------|------|------|------|
| GET | `/api/cluster/health` | 集群 token **或** 登录用户 | 健康汇总（green/yellow/red），类似 ES `_cluster/health` 的简化版 |
| GET | `/api/cluster/nodes` | 集群 token **或** 登录用户 | 查看本机+对端列表 |
| POST | `/api/cluster/keepalive` | 集群 token | 节点互探；响应为本节点快照 |
| POST | `/api/cluster/cmd-invoke` | 集群 token | 内部功能调用 |

示例：

```bash
# 集群健康
curl -H "x-cluster-request: shared-secret" http://192.168.1.10:8088/api/cluster/health

# 查看节点
curl -H "x-cluster-request: shared-secret" http://192.168.1.10:8088/api/cluster/nodes
```

#### `GET /api/cluster/health` 字段与状态

| status | 含义 |
|--------|------|
| **green** | 未开集群（单机），或已开集群且配置对端全部存活且已有 name |
| **yellow** | 已开集群但：无 peer 配置 / 部分 peer 不可达 / 部分 peer 尚无 name |
| **red** | 本机 `name` 为空，或配置了 peer 但全部不可达 |

主要字段：`enabled`、`local`、`numberOfNodes`、`numberOfNodesAlive`、`numberOfPeers`、`numberOfPeersAlive`、`unnamedPeers`、`message`、`nodes`。

说明：

- 状态是 **本节点 Registry 视角**，不是全局投票共识；各节点各自探测，短时间可能不一致。  
- **不是** ES 的分片/副本健康，只表示接入集群成员与可达性。  
- 可用作监控探活参考；K8s readiness 是否绑定需自行权衡（yellow 时业务往往仍可用）。

---

## 5. 设备 `ClusterId` 生命周期

| 事件 | 行为 |
|------|------|
| 设备上线（online 事件） | Redis/内存 `clusterId` = 本机 `name` |
| 设备离线（offline 事件） | **仅当当前归属仍是本机** 时清空 `clusterId`（避免 A 的延迟 offline 清掉已在 B 重连的归属） |
| 功能调用 | 有 `clusterId` 则点对点；空则本机处理 / 离线队列 |

注意：节点宕机时会话丢失，设备应重连；重连前 `clusterId` 可能短暂指向死节点，调用会失败直至重连刷新或离线清空逻辑生效。

---

## 6. eventpush（WebSocket 事件推送）说明

管理端设备事件推送 **不走** Registry 点对点，而使用 **Redis Pub/Sub**（频道 `go:cluster:eventpush`，包 `pkg/api/eventpush`）：

- 本机有 WS 订阅且产生事件时，可 Pub 到集群；其它节点 Sub 后推给本地订阅者。  
- 允许丢消息；与功能调用的「必须到会话节点」不是同一模型。  
- 集群关闭时不订阅读频道。  
- **升级注意**：旧频道 `go:cluster:realtime` 已废弃，滚动升级期间新旧节点互不收 eventpush 集群消息，建议尽快全量升级。

运维上：保证各节点 Redis 相同即可；不要用 `/cluster/nodes` 的 Alive 状态推断 eventpush 是否可达。

---

## 7. 启动与依赖顺序

`internal/app` 中相关顺序：

1. `cluster.Config`（Registry + keepalive 协程）  
2. `es` / `redis` / store 注册  
3. `RegNodeInvoker`（HTTP 功能调用）  
4. 模型与 boot 恢复（含按 **Shard** 恢复 netclient）  
5. 内置 MQTT、HTTP API（含集群内部路由）

因此：**先保证 API 端口与 token 互通**，再依赖跨节点功能调用；刚启动几秒内 Name 未回填时分片转发可能短暂失败。

---

## 8. 注意事项清单（排障）

1. **跨节点 405 / Method Not Allowed**  
   - 检查 `x-cluster-request` 是否等于各节点 `token`；`enabled` 是否为 true。

2. **`cluster node not found` / `shard owner ... no name`**  
   - keepalive 未成功（URL 不可达、token 错、防火墙）。  
   - 查 `GET /api/cluster/nodes` 是否已有对端 `name` 且 `alive: true`。

3. **功能调用总失败或总在本机**  
   - 设备是否在线、`clusterId` 是否为本机/对端 name。  
   - 是否误用旧逻辑：应用 `DeviceGateway`，不要自行 Redis Pub 命令。

4. **Connect 连到错误节点 / 重复连接**  
   - 检查各节点 `index` 是否唯一且覆盖 `0..N-1`。  
   - 扩容后 Size 变化，重新评估分片。

5. **产品/规则改了只在一台生效**  
   - 用户请求是否走了带 `IsNotClusterRequest` 的入口；  
   - Broadcast 是否因对端不 Alive 被跳过（看 nodes 与日志）。

6. **配置键写错**  
   - yaml / flag 统一使用 **`enabled`**（`cluster.enabled`）。旧键 `enable` 仅在未设置 `enabled` 时兼容；建议对照启动日志 `cluster enabled: name=...`。

7. **安全**  
   - 集群 token 等效于节点间 root；勿暴露公网无 TLS 的 API；生产建议专网 + 强 token。  
   - `/api/cluster/*` 除 `nodes` 外均勿对公网开放。

8. **时间与超时**  
   - 管理侧设备命令默认约 13s；内部 invoke 默认约 10s；可按链路调整。

---

## 9. 与「单机」行为对照

| 项目 | 单机 | 集群 |
|------|------|------|
| 功能调用 | 本机 Session | Gateway 按 ClusterId RPC |
| 设备断连 API | 本机会话 | Proxy 到会话节点 |
| Connect | 本机 | 分片负责节点 |
| 产品发布 | 本机 | 本机 + Broadcast |
| eventpush | 本机 eventbus | + Redis 扇出（`go:cluster:eventpush`） |
| ClusterId | 可写本机名 | 路由关键字段 |

---

## 10. 代码索引（二次开发）

| 能力 | 包/符号 |
|------|---------|
| 节点表 | `cluster.GetRegistry` / `ListNodes` / `FindNode` |
| 设备 HTTP 代理 | `cluster.ProxyOrLocal` / `ResolveDeviceOwner` |
| 分片 | `cluster.Shard` / `ResolveShardOwner` / `OwnerIndex` |
| 广播 | `cluster.BroadcastInvoke` |
| 功能调用 | `core.GetDeviceGateway().Invoke` / `core.NodeInvoker` |
| 装配 | `internal/app.New` → `cluster.NewHTTPNodeInvoker` |
| 离线队列 | `core.OfflineCommandQueue`（Redis 实现，共享） |

**产品/规则请继续使用 Broadcast；设备会话与命令请使用 Gateway / Proxy / 分片 API，不要混用语义。**

---

## 11. 版本说明

本文对应集群重构后的行为（DeviceGateway HTTP 命令、Registry、分片单点 Connect、离线清理 ClusterId）。  
若你本地仍看到 Redis 频道 `go:cluster:cmdinvoke` 用于**功能调用**，说明运行的是旧二进制，请重新构建部署。
