<div align="center">
<h1>go-iot 前端工程 (Vue 3)</h1>
</div>

## 介绍

`go-iot` 官方前端管理后台，基于 **Vue 3 + Vite + TypeScript + Element Plus + Pinia + UnoCSS** 构建。
作为 `go-iot` 单体仓库（Monorepo）的一部分，位于主工程的 `frontend/` 目录下。

---

## 核心特性

- **现代技术栈**：Vue 3.4+ / Vite 5 / TypeScript 5 / Pinia / UnoCSS
- **组件库**：基于 Element Plus 定制开发
- **国际化**：内置中英文多语言国际化方案（vue-i18n）
- **Monorepo 一体化**：
  - 本地开发：Vite 自动反向代理后端 `/api`
  - 生产构建：产物直接输出至 `../bin/views`，由 Go 后端直接提供静态托管

---

## 开发与使用

### 1. 安装依赖

确保本地已安装 Node.js (>= 18.0.0) 和 `pnpm` (>= 8.0.0)：

```bash
cd frontend
pnpm install
```

### 2. 本地开发调试

启动 Vite 开发服务器：

```bash
pnpm dev
```
- 访问地址：`http://localhost:3005`
- 接口代理：本地 `/api` 请求将自动反向代理到 Go 后端服务（默认 `http://127.0.0.1:8088/api`）

### 3. 生产打包

执行生产模式构建：

```bash
pnpm build:pro
```
- 构建产物会自动输出至 `../bin/views` 目录（包含 `index.html` 及 `static/` 静态资源），与主工程 `bin/` 目录协同发布。

> **提示**：在主工程根目录下执行 `./build.sh fe` 亦可触发前端构建；执行 `./build.sh all` 可一键编译前端与 Go 后端。

---

## 常用脚本命令

| 命令 | 说明 |
|------|------|
| `pnpm dev` | 启动本地开发服务（mode: base） |
| `pnpm build:pro` | 生产环境打包（输出至 `../bin/views`） |
| `pnpm build:dev` | 开发环境打包测试 |
| `pnpm lint:eslint` | 执行 ESLint 校验与自动修复 |
| `pnpm lint:format` | 执行 Prettier 格式化 |
| `pnpm lint:style` | 执行 Stylelint 样式校验 |
| `pnpm ts:check` | TypeScript 类型检查 |

---

## 默认账号

> 后端初始化账号：`admin` / 默认密码：`123456`

---

## 目录结构

```text
frontend/
├── public/                    # 静态公共资源
├── src/
│   ├── assets/                # 图标、图片等静态资产
│   ├── components/            # 全局通用业务与基础组件
│   ├── hooks/                 # Vue Composition API 组合式函数
│   ├── locales/               # 国际化语言包（zh-CN / en）
│   ├── router/                # 路由配置与动态路由守卫
│   ├── store/                 # Pinia 状态管理
│   ├── views/                 # 业务页面（设备、产品、规则引擎、系统设置等）
│   └── main.ts                # 前端入口文件
├── .env.base                  # 本地基础环境配置
├── .env.pro                   # 生产环境配置（VITE_OUT_DIR=../bin/views）
├── package.json               # 依赖与脚本
└── vite.config.ts             # Vite 构建与代理配置
```

---

## 浏览器支持

本地开发推荐使用现代浏览器（Chrome 80+、Edge、Firefox、Safari 最新版本），不支持 IE。

---

## 许可证

[MIT](./LICENSE)

