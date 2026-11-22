# Release Notes

## 说明：本仓库的性质

B2C 是 [LF Edge eKuiper](https://github.com/lf-edge/ekuiper) 的同步副本（fork）。
分界点为提交 `fce9a2e1`（上游 `feat: support array_avg (#4108)`，2026-08-18），
分界点之后共 54 个自有提交、净增 344 行。

下列条目按来源分为两部分：**来自上游（随同步带入）** 与 **本分支自行提交**。
请勿把上游变更记为本分支成果。

## 来自上游：默认安全加固（commit `60e16db7`）

以下行为变更**由上游提交 `60e16db7`（`refactor(conf): harden default security (#4089)`，
2026-07-29）引入**，其时间早于本分支分界点 `fce9a2e1`（2026-08-18），因此属于上游能力，
经同步带入本仓库，并非本分支的改动。

- REST API 默认仅绑定 localhost（`etc/kuiper.yaml: basic.restIp`）
- CLI 服务默认仅绑定 localhost（`etc/kuiper.yaml: basic.ip`）
- HTTP 数据接入服务默认仅绑定 localhost（`etc/kuiper.yaml: source.httpServerIp`）
- 外部文件访问默认关闭（`etc/kuiper.yaml: basic.allowExternalFileAccess: false`）
- 生产环境若暴露 REST API，应启用鉴权（`basic.authentication` 目前仍默认 `false`）

> **行为变更：** 默认情况下，REST 管理 API、CLI 服务与 HTTP 数据接入服务仅监听 `127.0.0.1`。
> 远程设备通过 HTTP 上报数据、以及远程 CLI/REST 客户端，在运维方显式设置绑定地址
> （例如 `0.0.0.0`）并在适当时启用鉴权之前，将无法连接。

## 本分支自行提交的变更

### 新增功能：MQTT v5 请求-响应网关

- 提交：`b7f527b1`（网关闭环）、`ffb1c99a`（常规场景适配 + 修复默认 protocolVersion 被空串覆盖）、
  `918a7dd6`（补齐 forwardStatus/forwardHeaders/forwardErrors/forwardEmpty 与 PATCH/HEAD）、
  `ae96c4d7`（`query` / `fileName` / `metaHeaders` 三项转发能力）。
- 能力：REST sink 把 HTTP 响应发布回 MQTT 主题，支持从 MQTT v5 `ResponseTopic` /
  `CorrelationData` 动态解析回传目标；也可用固定主题做单向协议桥接。
- 文档：`docs/zh_CN/guide/sinks/builtin/rest.md`、`docs/en_US/guide/sinks/builtin/rest.md`。
- **未做性能评测**：无吞吐、延迟、资源占用基准数据。

### 安全修复（针对上游既有缺陷）

- `4cf65502` InfluxDB v2 line protocol 注入（未转义 tag/field/string 值）
- `0f7b4566` SQL 注入（sqlgen `quoteValue` 未转义单引号）
- `a7cae6ce` ONNX Float16 分支长度 0 导致越界 panic（远程 DoS）
- `3504ed3c` / `552ae35c` 图像解码与缩放缺少尺寸上限导致 OOM（远程 DoS）
- `18af61f8` TDengine3 sink 字符串未转义 + 动态字段名校验缺失（SQL 注入）
- `e2a58993` / `0a569cf0` ONNX / TFLite Int 家族类型转换误传全参数数组
- `4efedf52` 原生插件 `storeSymbols` 部分写入导致符号表永久污染
- `9856f9cf` 原生插件全局 manager 在初始化成功前被发布
- `af1d8c45` 启动期插件加载解引用 nil manager
- `49f75591` MQTT v5 全部主题已订阅时仍发空 SUBSCRIBE，broker 依协议断开连接
- `0188eeb0` 先快照算子状态再写入 checkpoint store

### 构建与工程化

- `a2ad93dd` 用纯 Go 的 `go-zeromq/zmq4` 替换 cgo 依赖 `pebbe/zmq4`，消除交叉编译对 libzmq 的依赖
- `d6603d7c` 依赖整体升级消除 CVE；工具链随上游演进（当前 `go 1.27.4`）
- `b6b0613d` 新增自有 CI `check.yml`（仅 `go build ./cmd/...`）
- `613580e9` 新增 Dependabot 配置
- `b16c69ca` 补 `.dockerignore`，收敛 Docker 构建上下文
- `341db019` Dockerfile 允许 Go toolchain 切换
- `bb252019` helm chart 与 fork 构建路径对齐
- `feb93f09` / `01f8321b` / `4259a58b` 为上游 workflow 补仓库守卫，避免 fork 误触发
- `2b3e511a` 新增 `SECURITY.md`

### 文档

- `75fb80e6` / `1acc2a34` 重写中英文 README
- `a7ca5b28` 修复 14 处失效链接
- `3a3551af` 修复 README-JP 失效链接
- `91f12e1e` / `58f2897f` 移除上游 CONTRIBUTING，改指上游正式 URL
- 当前版本：README 系列改为诚实叙事，逐条区分 [上游能力] 与 [本分支新增]，
  删除无法复现的「核心包约 4.5MB」（实测 stripped 二进制约 37–41MB）
