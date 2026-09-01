# AGENTS.md

给在本仓库改代码的 agent / 开发者用。先读本文件，再改代码。项目文档以中文为主，代码标识符保持英文。

## 这是什么

`go-iot` 是 **物模型驱动** 的 IoT 接入平台（Go 后端 + Vue3 管理端，Monorepo）。

产品（Product）定义物模型、编解码、网络类型；设备（Device）是产品实例。设备报文经 Codec 解成物模型属性/事件/功能，再进 EventBus、规则引擎、时序存储。

**领域主链路**

```text
设备 ⇄ Network(server/client) ⇄ Session ⇄ Codec(goja JS)
                                      ↓
                                   TSL 物模型
                                      ↓
                    EventBus → 规则引擎 / 通知 / 时序 / 管理端推送
```

**南向 vs 北向**

| 方向 | 网络类型 | 含义 |
|------|----------|------|
| 北向（设备连上来） | `MQTT_BROKER` `TCP_SERVER` `HTTP_SERVER` `WEBSOCKET_SERVER` `COAP_SERVER` | 产品独占端口；`GOIOT_MQTT_BROKER` 是平台共用 Broker（默认 1883） |
| 南向（平台连出去） | `MQTT_CLIENT` `TCP_CLIENT` `MODBUS` | 平台作客户端/主站 |

`MODBUS` 采集走 **点表 + 采集组**（`CollectorConfig`），不要再把轮询写进物模型 `functions[].expands.interval`。`OnInvoke` 仍用于写线圈/设定点。

## 常用命令

```bash
# 后端
go run main.go                          # 默认 :8088，读 conf/app.yaml
go test ./pkg/... -count=1 -short       # 默认单测（不依赖本机 ES/Redis 的用例）
go test ./pkg/network/clients/modbus -count=1
go build -o bin/go-iot.exe main.go      # Windows；其它平台用 build.sh

# 前端（Node >= 18，pnpm）
cd frontend
pnpm install
pnpm dev                                # http://localhost:3005，/api 代理到 :8088
pnpm build:pro                          # 产物 -> ../bin/views/
pnpm lint:eslint && pnpm ts:check

# 一体化
./build.sh fe | all | linux | mac | macm
```

依赖：Go 1.24+、Redis、Elasticsearch 7.17.7+ / 8.x。可选 TDengine（`tdengine.enabled`）。默认账号 `admin` / `admin.password`（默认 `123456`）。配置文件 `conf/app.yaml`，环境变量前缀 `GOIOT_`。

Windows 上 `go test -race` 需要 gcc；MSYS2 可能还要 `-ldflags=-extldflags=-Wl,--disable-dynamicbase`。无 gcc 时不要开 `-race`。

## 目录

```text
main.go                 入口：option → logger → app.New/Run
internal/app/           组装根：依赖、Start/Stop 顺序
pkg/registry/           blank import：API / codec / 协议 / 通知 的 init 注册
pkg/core/               领域核心：Product/Device/Codec/Session/Invoke/Collector
pkg/tsl/                物模型 schema
pkg/codec/              goja 脚本编解码 + VM 池
pkg/network/            协议：servers/*、clients/*、platform/mqtt5
pkg/api/                HTTP 管理 API（chi）；web/ 是 mux/session
pkg/models/             ES ORM 实体；models/device|base|network|rule|notify
pkg/service/product/    产品发布/脚本/点表等业务（可测，ES 经 hook 注入）
pkg/store/              DeviceStore（Redis / mock）
pkg/eventbus/           属性/事件/上下线/告警
pkg/ruleengine/         规则
pkg/timeseries/         ES / TDengine / mock / noop
pkg/cluster/            多节点分片与 HTTP RPC 转发
pkg/agent/              管理端 AI 助手（工具 + codec skill）
pkg/option/             flag / yaml / 环境变量
conf/app.yaml           运行配置
frontend/               Vue3 + Vite + Element Plus + Pinia + UnoCSS
frontend/src/views/iot/ 业务页面（产品/设备/规则/告警/agent）
doc/                    已入库文档（cluster、benchmark）
docs/                   本地笔记/设计稿（.gitignore，不要当交付文档）
```

新代码优先挂在 `internal/app.App` 字段上；现有路径仍大量用 `core.Reg*` 包级全局，兼容期不要无故拆掉。

## 后端约定

### 启动顺序（不要打乱）

`internal/app.Start`：注册 ES 模型 → 默认 admin/网络 → 恢复菜单/网络/规则/通知/客户端 → eventpush → 内置 MQTT5 → HTTP API。协议工厂必须先被 `pkg/registry` blank import 注册。`main.go` 只负责 parse/logger/app，不要把业务 `init` 塞回 `main`。

### 加管理 API

1. 在对应 `pkg/api/*_controller.go` 的 `init()` 里 `web.RegisterAPI(path, method, handler)`。path **不要**带 `/api`（`web.APIPrefix` 会加）。
2. 用 `NewAuthController` + `ctl.isForbidden(resource, QueryAction|CreateAction|SaveAction|DeleteAction)`。漏鉴权等于漏洞。
3. `RegResource` 注册菜单资源；前端按钮用 `v-hasPermi="'product-mgr:save'"` 这种 `资源:动作`。
4. 返回走 `RespController`：`RespOk` / `RespError` / `Resp(data)`，保持 `{ success, result, message }` 形态。
5. 产品/设备写操作通常还要校验 `CreateId`（创建人）。

`CreateAction` 的权限 Id 是 `"add"`（不是 `"create"`），已进角色权限数据，不要改 Id。ES 用户名配置键是 `es.username` / `--es.username` / `GOIOT_ES_USERNAME`。

### 加协议 / Codec

- 实现 `core.Codec`（`OnConnect` / `OnMessage` / `OnInvoke` / `OnClose`）和需要的 `Session`。
- `init()` 里 `core.RegCodecCreator` 或 `network.RegNetworkMetaConfigCreator`。
- 在 `pkg/registry/registry.go` blank import 新包，否则生产进程加载不到。
- 南向采集实现 `core.CollectorRuntime`（含 `Normalize` 点表解析/校验），`core.RegCollectorRuntime(networkType, runtime)`。产品/设备服务只按网络类型调该接口，禁止 import 具体协议包做 Parse/Validate。
- 编解码文档：`pkg/agent/codecdoc/*.md` + `pkg/agent/skills/catalog.go`。改脚本 API 必须同步这两处，否则管理端 Agent 会教错。

脚本宿主是 **goja**。`HttpRequest` 受 `script.http-enabled` / `script.http-block-private` 约束。不要在脚本全局创建对象；模板见 `frontend/src/views/iot/product/detail/codec/`。

### 物模型 vs 点表

- **TSL**（`pkg/tsl`）：协议无关的 properties / events / functions。改 schema 要能 round-trip JSON，并跑 `pkg/tsl` 测试。
- **点表**（目前仅 MODBUS）：`pkg/network/clients/modbus.CollectorConfig`，产品级 JSON，存在 `models.ProductCollector`，**不进 TSL**。校验用 `Validate(tsl)`。设备可用 `metaconfig` 覆盖间隔。
- 产品发布走 `pkg/service/product`：编脚本、装运行时、Reload 采集器。HTTP 与 Agent 共用这层，不要在 controller 里复制发布逻辑。

### 数据落在哪

| 数据 | 存储 |
|------|------|
| 用户/角色/产品/设备/规则/通知/点表等元数据 | Elasticsearch（自研 `pkg/es/orm`） |
| 设备运行态、Session、离线命令、eventpush | Redis |
| 属性/事件/日志时序 | ES 或 TDengine（产品 `storePolicy`） |

实体加字段：改 `pkg/models/entitys.go`（或子包 struct tag），并在 `models.RegisterModels()` 注册。ORM tag 形态：`orm:"pk;column(id_);..."`。

### 并发与生命周期

长连接路径（session、broker、codec VM、eventpush）默认多 goroutine。共享 map/struct 字段要有锁或 atomic；`Disconnect`/`Stop` 必须幂等，不要 `close` 已关闭 channel。锁序一旦在某协议里定了（例如 mqtt5 的 `infoMu` 不与 `broker.Lock` 嵌套），后续改动遵守原锁序。

功能调用：`core.DoCmdInvoke` / `DoCmdInvokeOffline`。集群下会话在哪台节点，调用就转发到哪台（`pkg/cluster` HTTP RPC）。不要假设本进程一定持有该设备 session。

日志用 `go-iot/pkg/logger`（可 `logs` 别名）。测试里 `logger.InitNop()`，不要打真实文件。

## 前端约定

- 栈：Vue 3.4 + Vite + Element Plus + Pinia + vue-i18n + UnoCSS。IoT 页面多在 `frontend/src/views/iot/<域>/`，接口集中在同目录 `api.js`，经 `@/axios` 调后端（开发代理 `/api` → `:8088`）。
- 改页面时 **跟随该文件现有风格**（不少 IoT 页是 Options API / `lang="jsx"`，不要无故改成 `<script setup>`）。
- 新文案同时改 `frontend/src/locales/zh-CN.ts` 与 `en.ts`。权限指令 `v-hasPermi` 必须与后端 Resource/Action id 一致。
- 产品详情 tab：基本信息 / 物模型 / 编解码；`networkType === 'MODBUS'` 才显示「点表/采集」。
- 生产构建输出到 `bin/views/`，由 Go 托管。不要把 `frontend/node_modules`、`bin/`、`logs/` 提交进 git。

改了 UI / 路由 / 接口绑定后，要用浏览器走通：登录 → 打开改过的列表/详情 → 提交/保存 → 再进相关页确认状态一致。只截一张静态图不算验证。

## 测试

- 框架：`testing` + `github.com/stretchr/testify/require`。
- 网络协议：`pkg/network/testhelper`（`InitCore`、`FreeTCPPort`、`SetupProductDevice`、mock store）。
- 产品业务：`pkg/service/product` 用函数 hook 避开真 ES。
- 需要 ES/Redis 的集成测试用 `testing.Short()` 跳过，保证 `go test ./pkg/... -short` 可在无中间件环境绿。
- 新协议/采集/会话：补成功路径 + 断开/重入幂等；能测的并发用 `-race`。
- 不要为了让测试过而吞 panic、睡固定很久、或依赖本机 502/1883 上已有服务。端口用 `FreeTCPPort`。

## 改动范围（必须遵守）

- 只改任务需要的文件。不顺手格式化无关文件，不重写前端脚手架，不升级依赖除非任务要求。
- 对外 HTTP 路径、JSON 字段、物模型结构、网络类型枚举保持兼容。要破坏兼容先说明。
- 不要把业务元数据「顺便」迁到 SQL（那是独立大项，见改造路线图）。
- 不要提交密钥、真实 `conf/app.yaml` 生产密码、`bin/` 二进制、日志。
- 集群能力边界见 `doc/cluster.md`：无会话热迁移、无跨节点强一致事务。不要实现半吊子「强一致」。
- 脚本可访问的 HTTP 默认禁止私网；不要为了方便把 `script.http-block-private` 改成默认 false。

## 做一类任务时先看这些文件

| 任务 | 先读 |
|------|------|
| 启动/依赖/关闭 | `internal/app/app.go`、`pkg/registry/registry.go` |
| 物模型 | `pkg/tsl/`、`frontend/src/views/iot/product/detail/tsl/` |
| 编解码脚本 | `pkg/codec/`、`pkg/agent/codecdoc/`、`frontend/.../detail/codec/` |
| Modbus 采集 | `pkg/core/collector.go`、`pkg/network/clients/modbus/`、`pkg/models/device/collector.go` |
| 产品发布 | `pkg/service/product/product.go`、`pkg/api/product_controller.go` |
| 设备调用/离线 | `pkg/core/invoke.go`、`pkg/core/device_gateway.go` |
| 管理 API / 鉴权 | `pkg/api/auth.go`、`pkg/api/web/` |
| 集群 | `doc/cluster.md`、`pkg/cluster/` |
| 时序 | `pkg/timeseries/`、`pkg/core/timeseries.go` |
| 前端产品/设备 | `frontend/src/views/iot/product/`、`.../device/` |
| 管理端 Agent | `pkg/agent/`、`frontend/src/views/iot/agent/` |

更长的设计稿（本地）：`docs/plans/`。已入库说明：`README.md`、`doc/cluster.md`、`doc/benchmark.md`。
