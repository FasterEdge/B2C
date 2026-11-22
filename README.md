<div align="center">
  <img src="Logo.png" alt="B2C" width="120"/>
  <h2>B2C</h2>
  <h3>eKuiper 同步分支 + MQTT v5 请求-响应网关定制</h3>
</div>

**语言 / Language：** [简体中文](./README.md) | [English](./README_en.md) | [中文导览](./README-CN.md)

> 本仓库曾长期保留一份未改动的上游日语文档 `README-JP.md`。该文件把本项目描述为
> "LF Edge eKuiper" 且从未更新 fork 事实，继续留存会造成误导，故已删除。
> 日文用户请改读上游仓库 <https://github.com/lf-edge/ekuiper> 的日文文档。

### 一、这是什么（先说清楚）

- **B2C 是 [LF Edge eKuiper](https://github.com/lf-edge/ekuiper) 的同步副本（fork）**，不是重新实现的引擎。
  分界点提交为 `fce9a2e1`（上游 `feat: support array_avg (#4108)`，2026-08-18）。
- 从分界点至今共 **55 个自有提交**，代码净增 **632 行**（其中 Go 代码净增 1277 行），相对上游约 12.6 万行代码（`git ls-files | xargs wc -l` 统计为 125418 行），净增占比约 **0.50%**。
  因此**本仓库 99% 以上的能力来自上游 eKuiper**，请勿将其理解为"B2C 自研引擎"。
- 本分支唯一的自研功能是 **MQTT v5 请求-响应网关**（见第六节），其余自有提交主要是
  安全修复、构建/CI 适配与文档修正。
- 自有 CI 见 `.github/workflows/check.yml`，仅执行 `go build ./cmd/...`（上游 eKuiper workflow
  为同步保留内容，多数作业带仓库守卫，在本 fork 不运行）。该文件头部注释亦明确记载
  "本仓为 lf-edge/ekuiper 同步副本"。
- **模块路径仍为上游的 `github.com/lf-edge/ekuiper/v2`**（`go.mod:424` 声明 `go 1.27.4`）。
  这是 fork 的常态：改模块路径需要同步改上千处 import，收益低而风险高，故未做。

#### 代码量与提交构成（可自行复核）

```bash
# 自有提交数、改动规模、仓库总行数（相对上游分界点）
git rev-list --count fce9a2e1..HEAD        # => 55
git diff --shortstat fce9a2e1..HEAD        # => 140 files changed, 3947 insertions(+), 3315 deletions(-)
git ls-files | wc -l                       # => 1859（这是文件数，不是行数）
git ls-files | xargs wc -l                 # 末行 total 125418（行数口径）

# 仅看 Go 代码的净增
git diff --numstat fce9a2e1..HEAD | awk '$3 ~ /\.go$/ {a+=$1; d+=$2} END {print "net", a-d}'   # => net 1277
```

> 说明：`git ls-files | wc -l` 数的是被跟踪文件数（1859 个），**不是代码行数**；
> 行数口径是 `git ls-files | xargs wc -l` 的 total（125418 行）。
> "净增 632 行" = 3947 insertions − 3315 deletions，取自 `git diff --shortstat`；
> 占比 632 / 125418 ≈ 0.50%。

### 二、能力归属：上游 vs 本分支

为避免误读，下表逐条区分。**凡未标注 [本分支新增] 的条目，均为上游 eKuiper 的能力**。

#### [上游能力] 数据接入与流转

- 规则拓扑与执行（`internal/topo`）、SQL 引擎（`internal/xsql`）。
- 内置源：MQTT、EdgeX、HTTP pull/push、File、Simulator、Video 等；
  内置目标：MQTT、File、REST、InfluxDB v1/v2、Kafka、Redis、TDengine 等。
- 安装即用的编译型源/目标（`extensions/sources`、`extensions/sinks`）：
  `kafka`、`sql`、`video`、`zmq` 源；`image`、`influx`、`influx2`、`kafka`、`sql`、`tdengine3`、`zmq` 目标。
- 可扩展 Source / Sink / SQL 函数，支持 **Golang 与 Python** 两种插件机制。

#### [上游能力] 计算与查询

- 基于 SQL 的流式计算：过滤、投影、分组聚合、窗口、多流连接（JOIN）、`GROUP BY` / `HAVING`。
- 窗口类型由 `pkg/ast/statement.go:180-188` 定义，共 **6 种**：
  `TUMBLING` / `HOPPING` / `SLIDING` / `SESSION` / `COUNT` / `STATE`。
  （常见"4 类时间窗口 + 计数窗口"的说法是上游文档的旧表述，未计入后来加入的 `STATE_WINDOW`。）
- 内置 SQL 函数：`internal/binder/function/funcs_*.go` 中注册的函数名去重后共 **192 个**
  （覆盖数学、字符串、聚合、数组、对象、类型转换、日期时间、窗口函数等）。
  可用下面的命令复核，它统计的是注册表中的字面量而非文档口径：

  ```bash
  grep -rhoE 'builtins\["[a-zA-Z_0-9]+"\]' internal/binder/function/funcs_*.go \
    | sed 's/builtins\["//;s/"\]//' | sort -u | wc -l   # => 192
  ```

  > **统计口径（重要）**：该通配符会把同目录的 `funcs_*_test.go` **一并计入**——
  > 15 个测试文件里有 91 处 `builtins["..."]` 引用，覆盖 68 个名字，但这些名字全都是
  > 非测试文件中已注册名字的子集，所以去重后不改变结果。
  > 若严格只统计非测试文件（`ls internal/binder/function/funcs_*.go | grep -v '_test\.go$'`），
  > 去重后同样是 **192 个**，与上面口径一致；早期文档所写的 191 是漏数。

#### [上游能力] 管理面

- CLI（`bin/kuiper`）、REST API（默认端口 `9081`）、Kubernetes config map 管理。
- REST 路由表在 `internal/server/rest.go` 中注册：`grep -c HandleFunc internal/server/rest.go` 得 **62 个**
  `HandleFunc` 路由，另有 4 个 pprof 路由用 `r.Handle(` 注册（`/debug/pprof/goroutine`、`heap`、
  `threadcreate`、`block`），合计 **66 条**路由注册语句。覆盖流、表、规则、连接、插件、
  配置导入导出、规则 trace、指标 dump 等，规模远超早期文档所列的十几条。

  ```bash
  grep -c HandleFunc internal/server/rest.go                    # => 62
  grep -cE 'r\.Handle(Func)?\(' internal/server/rest.go          # => 66
  ```
- 可视化 Web 管理控制台（eKuiper Manager）为**独立项目**，不在本仓库内。

#### [本分支新增] MQTT v5 请求-响应网关

本分支对 REST sink 与 MQTT v5 客户端做了定制，把"收到请求 → 转发 HTTP → 把响应送回请求方"
这一闭环补齐。详见第六节。

#### [本分支新增] 安全修复（针对上游既有缺陷）

以下均为**修复上游缺陷**，不是新增攻击面：

| 方向 | 位置 | 说明 |
|------|------|------|
| InfluxDB v2 line protocol 注入 | `extensions/impl/influx2/influx2.go` | 未转义 tag/field/string 值 |
| SQL 注入 | `extensions/impl/sql/sqldatabase/sqlgen/` | `quoteValue` 未转义单引号 |
| 远程 DoS（整数越界 panic） | `extensions/impl/onnx/onnx.go` | Float16 分支 `make([]byte,0,4)` 长度 0 |
| 远程 DoS（OOM） | `extensions/functions/image/`、`extensions/impl/image/image.go` | 图像解码/缩放无尺寸上限，加 `maxImageDim=8192` 与 `DecodeConfig` 预检查 |
| SQL 注入 | `extensions/impl/tdengine3/tdengine3.go` | 字符串值未转义 + 动态字段名未校验 |
| ONNX/TFLite 类型转换 | `extensions/impl/onnx/onnx.go`、`extensions/impl/tflite/tfLite.go` | Int 家族误传全参数数组致模型必然失败 |
| 原生插件符号表污染 | `internal/plugin/native/manager.go` | `storeSymbols` 部分写入后失败注册会永久污染符号表 |
| 启动期空指针 | `internal/plugin/native/` | `findAll` 未传 registry，启动加载插件时解引用 nil manager |
| MQTT 空订阅 | `internal/io/mqtt/v5client/client.go` | 全部主题已订阅时仍发空 SUBSCRIBE，broker 依协议断开连接 |
| 检查点状态快照 | `internal/topo/operator/` | 先快照算子状态再写 checkpoint store |

#### [本分支新增] 构建与 CI 适配

- **去掉 cgo 依赖**：用纯 Go 的 `go-zeromq/zmq4` 替换 `pebbe/zmq4`，消除交叉编译时对 libzmq 的依赖
  （提交 `a2ad93dd`）。
- 依赖版本整体升级以消除已知 CVE；Go toolchain 随上游演进（当前 `go 1.27.4`）。
- 新增 `.github/workflows/check.yml`（只跑 `go build ./cmd/...`）、`.github/dependabot.yml`、`SECURITY.md`。
- 补齐 `.dockerignore`，把 `.git`、`.github`、文档排除出 Docker 构建上下文。
- Dockerfile 允许 Go toolchain 切换；`deploy/chart/ekuiper/` 下的 chart 模板与 values 已按
  fork 的镜像名/构建路径调整（提交 `bb252019`）。

  > **关于 chart 版本号的说明（不是 bug）**：`deploy/chart/ekuiper/Chart.yaml:17,21` 的
  > `version: 1.4.0` / `appVersion: 2.3.0-beta.5` 是**上游 eKuiper 的 Helm chart 版本号**，
  > 与本仓库的 git tag（当前最新 `v1.0.20261123`）是两套独立的版本体系，并无对应关系。
  > 该 chart 属上游遗留资产，本 fork **不发布该 chart**（无对应 workflow，见第二节 CI 说明）；
  > 本仓只借用了其中的模板与 values 做构建路径对齐。读者不应把两者数字对不上视为版本错配。

### 三、关于"超轻量"的诚实说明

上游文档长期宣称"核心服务安装包约 4.5MB、首次运行内存约 10MB"。
**本分支实测无法复现该数字**，因此 README 不再沿用：

| 构建 | 命令 | 实测大小 |
|------|------|----------|
| `kuiperd` windows/amd64 | `CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o b2cd cmd/kuiperd/main.go` | **41.02 MB** |
| `kuiperd` linux/amd64 | 同上，加 `GOOS=linux GOARCH=amd64` | **40.32 MB** |
| `kuiperd` linux/arm64 | 同上，加 `GOOS=linux GOARCH=arm64` | **37.38 MB** |

- 差异主要来自上游多年累积的内置插件与依赖（InfluxDB/Kafka/Redis/TDengine 客户端、
  ONNX/TFLite 运行时绑定、脚本引擎等），并非本分支引入。
- 若要复现"约 4.5MB"，需要用 `make pkg_prebuild` 之类的最小 tag 组合裁剪插件，本仓库未提供该配置。
- 内存占用与具体规则负载强相关，**本分支未做任何性能评测**，不给任何内存数字。

### 四、快速开始

#### 1. Docker 方式

> 注意：`fasteredge/b2c:latest` 镜像**本仓库未发布**（无对应 workflow 发布到该仓库名）。
> 请使用上游镜像 `lfedge/ekuiper:latest`，或自行构建。

```bash
docker run -p 9081:9081 -d --name kuiper \
  -e MQTT_SOURCE__DEFAULT__SERVER="tcp://broker.emqx.io:1883" \
  -e KUIPER__BASIC__RESTIP=0.0.0.0 \
  lfedge/ekuiper:latest

docker exec -it kuiper /bin/sh
```

#### 2. 本地构建

```bash
# 环境要求：go.mod / go.work 声明 go 1.27.4（go.work:1）
go build -o bin/kuiperd ./cmd/kuiperd
go build -o bin/kuiper ./cmd/kuiper

# 启动服务（REST 端口默认 9081）
./bin/kuiperd
```

> `go build ./cmd/...` 是本仓库唯一的构建验收命令（见 `.github/workflows/check.yml`）。
> `go build ./...` 不适用：`extensions/*` 是 Go plugin（`main` 包无 `main` 函数）。

#### 3. 5 分钟上手

```shell
bin/kuiper create stream demo '(temperature float, humidity bigint) WITH (FORMAT="JSON", DATASOURCE="devices/+/messages")'

bin/kuiper query
kuiper > select * from demo where temperature > 30;

# 用任意 MQTT 客户端向 devices/device_001/messages 发布消息
# mqttx pub -h broker.emqx.io -m '{"temperature": 40, "humidity" : 20}' -t devices/device_001/messages
```

> 注意：上述示例依赖公网 `broker.emqx.io`。若网络不可达，请改为本地 broker 地址。

### 五、架构与组件

```
B2C/
├─ cmd/
│  ├─ kuiperd/     # 服务端守护进程
│  └─ kuiper/      # 命令行工具（CLI）
├─ internal/
│  ├─ topo/        # 规则拓扑与执行
│  ├─ xsql/        # SQL 引擎与流式处理
│  ├─ server/      # REST API 服务（路由表在 internal/server/rest.go）
│  └─ processor/   # 流/规则/插件管理
├─ pkg/            # 通用基础设施（ast、cast、kv、store 等）
├─ extensions/     # 内置扩展：source / sink / functions
├─ plugins/        # 用户自装插件目录（默认 git 忽略内容，见下）
├─ etc/            # 配置（kuiper.yaml、MQTT、连接等）
├─ sdk/            # Golang / Python 插件 SDK 与示例
└─ docs/           # 中英文文档
```

**关于空的 `plugins/` 目录**：`plugins/sources`、`plugins/sinks`、`plugins/functions` 在仓库中是空目录，
这是设计如此——`.gitignore:17-19` 忽略了它们的内容，它们是**运行时插件安装目标目录**，
不是示例代码库。可运行的插件示例在：

- `sdk/go/example/`（Golang 插件示例，如 `mirror`、`random`）
- `sdk/python/example/`（Python 插件示例）
- `plugins/portable/readme.md`（portable 插件的目录结构与加载方式说明）

`etc/sources/randomGo.json`、`etc/sinks/fileGo.json`、`etc/functions/echoGo.json` 是本分支
（提交 `7e966c57`）补入的**编译型插件声明示例**，可直接参考其字段格式。

### 六、自研功能详解：MQTT v5 请求-响应网关

这是本分支唯一的功能性新增，用于把上游缺失的"请求-响应"闭环补齐。

**解决的问题**：设备通过 MQTT v5 的 `ResponseTopic` / `CorrelationData` 属性发起请求，
边缘侧需要把请求转成 HTTP 调用业务系统，再把 HTTP 响应按原路送回请求方。
上游 eKuiper 的 REST sink 只能单向把消息发往 HTTP，无法把响应送回。

**实现位置**：

| 文件 | 作用 |
|------|------|
| `internal/io/mqtt/v5client/client.go:279` 起 | `ParseMsg` 从 MQTT v5 报文中提取 `ResponseTopic`、`CorrelationData`、`ContentType`，写入元数据供 SQL `meta()` 使用 |
| `internal/io/mqtt/v5client/client.go:186` 起 | `Publish` 把 `correlationData` / `responseTopic` 属性映射回 MQTT v5 标准发布属性，其余作为 user property |
| `internal/io/http/client.go:112` 起 | 新增 `Response` / `ResponseConf` / `ResponseMqttConf` 配置结构 |
| `internal/io/http/rest_sink.go:48` 起 | `responseMqttPublisher`：按消息解析回传主题，把 HTTP 响应体发布回 MQTT |
| `internal/io/http/rest_sink.go:477` 起 | `buildEnvelope`：可选地把 `status` / `headers` 包进回传体 |

**配置形态**（完整文档见 `docs/zh_CN/guide/sinks/builtin/rest.md` 与英文对应章节）：

```json
{
  "id": "ruleReqResp",
  "sql": "SELECT command, meta('responseTopic') AS responseTopic, meta('correlationData') AS correlationData FROM reqStream",
  "actions": [{
    "rest": {
      "url": "http://your-backend/api",
      "method": "post",
      "metaHeaders": { "X-Correlation-Id": "correlationData" },
      "response": {
        "type": "mqtt",
        "mqtt": {
          "server": "tcp://127.0.0.1:1883",
          "protocolVersion": "5",
          "topic": "{{.responseTopic}}",
          "correlationData": "{{.correlationData}}",
          "forwardStatus": true
        }
      }
    }
  }]
}
```

**边界与未做项（务必如实了解）**：

- 仅支持 MQTT 一种回传通道（`type` 目前只接受 `mqtt`）。
- `forwardStatus` / `forwardHeaders` / `forwardErrors` / `forwardEmpty` 默认均为 `false`，
  保持上游原始的"回传裸响应体"行为，不破坏既有用户。
- **未做任何性能评测**：没有吞吐、延迟、资源占用的基准数据，也没有与其它方案
  （如 HTTP 长轮询、WebSocket）的对比。论文如需性能指标，需另行设计实验。
- 未支持 QoS 1/2 的发布确认重试语义，也未做回传消息的持久化。

### 七、配置说明

核心配置位于 `etc/kuiper.yaml`：

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| `basic.restPort` | `9081` | REST API 服务端口 |
| `basic.restIp` | `127.0.0.1` | REST 绑定地址 |
| `basic.ip` / `basic.port` | `127.0.0.1` / `20498` | CLI 服务绑定地址与端口 |
| `basic.consoleLog` | `false` | 控制台日志开关 |
| `basic.fileLog` | `true` | 文件日志开关 |
| `basic.authentication` | `false` | REST API 鉴权开关 |
| `basic.allowExternalFileAccess` | `false` | 是否允许访问上传目录之外的文件 |
| `source.httpServerIp` | `127.0.0.1` | HTTP 数据接入服务绑定地址 |
| `source` | — | 各类数据源连接配置 |
| `sink` | — | 各类目标连接配置 |

**关于默认绑定到 `127.0.0.1`**：这一安全加固**来自上游提交 `60e16db7`**
（`refactor(conf): harden default security (#4089)`，2026-07-29），早于本分支分界点
`fce9a2e1`（2026-08-18），因此属于上游能力而非本分支成果。它在 `RELEASE_NOTES.md`
中被描述为行为变更，本 README 沿用同一事实描述。

常用环境变量（Docker）：`MQTT_SOURCE__DEFAULT__SERVER`、`KUIPER__BASIC__RESTPORT`、
`KUIPER__BASIC__RESTIP`、`KUIPER__BASIC__ALLOWEXTERNALFILEACCESS`。

### 八、REST API

服务默认监听 `9081`，完整路由表见 `internal/server/rest.go:190-290`，常用端点：

| 路径 | 方法 | 说明 |
|------|------|------|
| `/streams` | GET/POST | 查看 / 创建流 |
| `/streams/{name}` | GET/DELETE/PUT | 查看 / 删除 / 更新指定流 |
| `/tables` | GET/POST | 查看 / 创建表 |
| `/rules` | GET/POST | 查看 / 创建规则 |
| `/rules/{name}/status` | GET | 查询规则运行状态 |
| `/rules/{name}/start` `/stop` | POST | 启动 / 停止规则 |
| `/rules/validate` | POST | 校验规则定义 |
| `/connections` | GET/POST | 查看 / 创建连接 |
| `/plugins` | GET/POST | 查看 / 安装插件 |
| `/functions` | GET | 查看已注册函数 |
| `/sinks` / `/sources` | GET | 查看目标 / 源插件 |
| `/data/export` `/data/import` | GET/POST | 配置导出 / 导入 |

### 九、CLI 常用命令

```bash
bin/kuiper create stream demo '...'     # 创建流
bin/kuiper drop stream demo             # 删除流
bin/kuiper show streams                 # 列出所有流
bin/kuiper create rule myRule '...'     # 创建规则
bin/kuiper getstatus rule myRule        # 查询规则状态
bin/kuiper query                        # 进入交互式查询
bin/kuiper plugin install ...           # 安装插件
```

### 十、安全与可靠性

- **依赖升级**：`golang.org/x/text`、`x/net`、`x/crypto`、`klauspost/compress`、
  `go.opentelemetry.io/otel` 等升级至修复 CVE 的版本（提交 `d6603d7c` 及后续）。
- **工具链**：`go.mod:424` 与 `go.work:1` 声明 `go 1.27.4`。
- **govulncheck**：对 `./cmd/...` 与 `./internal/io/...` 扫描结果为
  `No vulnerabilities found.`（复现命令见下）。
- **默认绑定 localhost**、**默认禁止外部文件访问**：来自上游提交 `60e16db7`，非本分支成果。
- **生产部署注意**：REST API 默认无鉴权（`authentication: false`）。一旦把 `restIp`
  改为 `0.0.0.0` 暴露到网络，必须同时启用鉴权与 TLS。

```bash
# 依赖漏洞扫描（本分支实际执行过的命令）
govulncheck ./cmd/... ./internal/io/...
```

---

> 本项目基于开源项目 **[LF Edge eKuiper](https://github.com/lf-edge/ekuiper)** 构建。
> 上游采用 Apache License 2.0，本分支保留其许可与版权声明。
> 本分支代码规模占比很小，请以上游仓库的文档与社区为主要参考来源。
