# go-iot

#### 介绍
使用 Go 实现的 IoT 接入系统，以`物模型`为主体用来对接不同厂商的设备来实现统一接入的目的。本仓库采用 **Monorepo** 架构，整合了 Go 后端核心与 Vue3 前端工程。

#### 架构与目录

```text
go-iot/
├── conf/                      # 系统与业务配置文件
├── internal/ / pkg/           # Go 后端核心代码
├── frontend/                  # Vue3 + Vite + TypeScript + Element Plus 前端源码
├── bin/                       # 构建输出目录
│   ├── go-iot (或 .exe)       # 后端可执行文件
│   └── views/                 # 前端编译静态产物（index.html、static/ 等）
├── build.sh                   # 一体化构建脚本
├── main.go                    # 后端入口
└── go.mod / go.sum
```

#### 架构图
![IOT架构](./doc/img/IOT架构.png "IOT架构")

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
- TCP Server / TCP Client
- MQTT Broker / MQTT Client
- HTTP Server
- WebSocket Server
- Modbus TCP

---

#### 运行与构建说明

##### 1. 运行依赖
- **Go**：Go 1.24+
- **Node.js**：Node.js >= 18.0.0，包管理器推荐 `pnpm`
- **Redis**：设备运行态、HTTP Session、集群 eventpush 等
- **Elasticsearch**：业务元数据与时序数据（支持 7.x 及 8.x 版本，7.x 最低版本为 7.17.7）
- **集群（可选）**：[集群配置、路由模型与注意事项](./doc/cluster.md)

```bash
# Elasticsearch 7.x / 8.x（单节点快速启动示例）
docker run -d --name elasticsearch -p 9200:9200 -p 9300:9300 -e "discovery.type=single-node" -e "xpack.security.enabled=false" -e "ES_JAVA_OPTS=-Xms1024m -Xmx1024m" elasticsearch:8.13.0

docker run --name redis6 -d -it -p 6379:6379 redis:6
```

##### 2. 本地开发模式（前后端独立调试）
- **后端服务**：
  ```bash
  go run main.go
  # 默认监听 8088 端口
  ```
- **前端开发**：
  ```bash
  cd frontend
  pnpm install
  pnpm dev
  # 启动后访问 http://localhost:3005 （自动代理请求至本地后端 /api）
  ```

##### 3. 一体化生产构建
仓库根目录提供了统一构建脚本 `build.sh`：

```bash
# 1. 仅构建前端（产物自动输出到 bin/views/）
./build.sh fe

# 2. 一体化构建（自动构建前端 + 编译本机 Go 二进制）
./build.sh all

# 3. 仅构建指定平台的 Go 二进制
./build.sh linux           # -> bin/go-iot-linux-amd64
./build.sh mac             # -> bin/go-iot-darwin-amd64
./build.sh macm            # -> bin/go-iot-darwin-arm64
./build.sh                 # 本机默认（Windows -> bin/go-iot.exe，Linux/Mac -> bin/go-iot）
```

##### 4. 生产部署与分发结构
构建完成后，直接将 `bin/` 目录打包分发至服务器即可，标准交付目录结构如下：

```text
bin/ (部署根目录)
├── go-iot (或 Windows 下 go-iot.exe)
├── conf/                          # 配置文件目录
└── views/                         # 前端静态产物目录
    ├── index.html                 # 前端入口页面
    ├── favicon.ico
    ├── logo.png
    └── static/                    # JS / CSS / 静态资源
        ├── index-xxx.js
        ├── index-xxx.css
        └── ...
```

在服务器进入该目录直接启动程序：
```bash
cd bin
./go-iot          # Windows 下为 .\go-iot.exe
```
打开浏览器访问 `http://localhost:8088`，后端服务会自动挂载并托管运行目录下的 `views/index.html` 与 `views/static/`。

##### 5. 单元测试
```bash
go test ./pkg/... -count=1 -short
```

---

#### 默认账号
> 首次启动若库中无用户：`admin` / 配置项 `admin.password`（默认 `123456`）  
> **生产务必**修改 `admin.password` 或环境变量 `GOIOT_ADMIN_PASSWORD`。  
> 密码使用 **bcrypt** 存储；历史 MD5 账号在登录成功后会自动升级。

#### 安全相关配置（P0）

| 配置 | 说明 |
|------|------|
| `admin.password` | 仅首次创建 admin 时生效 |
| `script.http-enabled` | 脚本 `HttpRequest` 总开关（默认 true） |
| `script.http-block-private` | 禁止脚本访问私网/本机（默认 true） |
| `script.vm-pool-size` | 每个产品编解码 JS 引擎池大小（默认 20，最大 500） |

管理 API 需登录；业务资源（设备/产品等）读写均校验角色权限。

#### 压力测试
- [压力测试](./doc/benchmark.md)
