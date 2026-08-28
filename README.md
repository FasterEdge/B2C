<div align="center">
  <img src="Logo.png" alt="B2C" width="120"/>
  <h2>B2C</h2>
  <h3>超轻量物联网边缘流式分析引擎</h3>
</div>

### 一、项目简介
- **B2C** 是 [LF Edge eKuiper](https://github.com/lf-edge/ekuiper) 的增强分支，一款可运行在各类资源受限硬件上的**物联网边缘流式数据分析引擎**。
- 提供类似 [Apache Flink](https://flink.apache.org) 的**实时流式计算框架**，可在边缘端完成数据接入、转换、分析、告警与转发。
- 通过 **SQL 规则** 或 **Graph 规则**（类似 Node-RED）快速创建物联网边缘分析应用。
- 面向 IIoT、车联网（IoV）、智慧能源等场景，显著降低响应延迟、节省带宽与存储成本、提升系统安全性。

### 二、主要特性
| 特性 | 说明 |
|------|------|
| 超轻量 | 核心服务安装包约 4.5MB，首次运行内存约 10MB |
| 跨平台 | X86/ARM 32/64、PPC；Linux、OpenWrt、macOS、Docker；工控机、树莓派、工业/家庭网关、MEC 边缘云 |
| 完整数据分析 | 数据抽取/转换/过滤（ETL）、排序、分组、聚合、连接；60+ 内置函数；4 类时间窗口 + 计数窗口 |
| 高可扩展 | 支持用 **Golang / Python** 扩展 `Source`（数据源）、`Sink`（目标）、`UDF`（自定义函数） |
| 便捷管理 | CLI、REST API、Kubernetes config map 管理流/规则/插件；可配 Web 管理控制台 |
| 生态集成 | 与 [EMQX](https://www.emqx.io/)、[Neuron](https://neugates.io/)、[NanoMQ](https://nanomq.io/) 无缝集成，提供 IIoT / IoV 端到端方案 |

### 三、快速开始

#### 1. Docker 方式（推荐）

```bash
docker run -p 9081:9081 -d --name kuiper \
  -e MQTT_SOURCE__DEFAULT__SERVER="tcp://broker.emqx.io:1883" \
  fasteredge/b2c:latest

# 进入容器
docker exec -it kuiper /bin/sh
```

#### 2. 本地构建

```bash
# 环境要求：go.mod / go.work 声明 toolchain go1.25.13（已修复标准库 CVE）
go mod tidy

# 构建服务端与 CLI
go build -o bin/kuiperd ./cmd/kuiperd
go build -o bin/kuiper ./cmd/kuiper

# 启动服务（REST 端口默认 9081）
./bin/kuiperd
```

#### 3. 5 分钟上手

```shell
# 创建流（类似数据库建表）：订阅设备消息
bin/kuiper create stream demo '(temperature float, humidity bigint) WITH (FORMAT="JSON", DATASOURCE="devices/+/messages")'

# 交互式查询
bin/kuiper query
kuiper > select * from demo where temperature > 30;

# 用任意 MQTT 客户端向 devices/device_001/messages 发布消息
# mqttx pub -h broker.emqx.io -m '{"temperature": 40, "humidity" : 20}' -t devices/device_001/messages

# 符合条件的数据会实时打印在 query 窗口
kuiper > [{"temperature": 40, "humidity" : 20}]
```

### 四、架构与组件

```
B2C/
├─ cmd/
│  ├─ kuiperd/     # 服务端守护进程
│  └─ kuiper/      # 命令行工具（CLI）
├─ internal/
│  ├─ topo/        # 规则拓扑与执行
│  ├─ xsql/        # SQL 引擎与流式处理
│  ├─ server/      # REST API 服务
│  └─ processor/   # 流/规则/插件管理
├─ pkg/            # 通用基础设施（ast、cast、kv、store 等）
├─ extensions/     # 扩展：source / sink / 函数
├─ plugins/        # 可插拔组件（portable）
├─ etc/            # 配置（kuiper.yaml、MQTT、连接等）
└─ docs/           # 中英文文档
```

### 五、配置说明

核心配置位于 `etc/kuiper.yaml`：

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| `basic.restPort` | `9081` | REST API 服务端口 |
| `basic.consoleLog` | `false` | 控制台日志开关 |
| `basic.fileLog` | `true` | 文件日志开关 |
| `source` | — | 各类数据源（MQTT、EdgeX、HTTP 等）连接配置 |
| `sink` | — | 各类目标（MQTT、文件、REST、InfluxDB 等）配置 |

常用环境变量（Docker）：

| 环境变量 | 说明 |
|----------|------|
| `MQTT_SOURCE__DEFAULT__SERVER` | 默认 MQTT 源服务器地址 |
| `KUIPER__BASIC__RESTPORT` | 覆盖 REST 端口 |
| `KUIPER__BASIC__CONSOLELOG` | 覆盖控制台日志开关 |

### 六、REST API

服务默认监听 `9081` 端口，提供完整的流 / 规则 / 插件管理接口：

| 路径 | 方法 | 说明 |
|------|------|------|
| `/streams` | GET/POST | 查看 / 创建流 |
| `/streams/{name}` | GET/DELETE | 查看 / 删除指定流 |
| `/rules` | GET/POST | 查看 / 创建规则 |
| `/rules/{id}` | GET/DELETE | 查看 / 删除指定规则 |
| `/rules/{id}/status` | GET | 查询规则运行状态 |
| `/rules/{id}/start` `/stop` | POST | 启动 / 停止规则 |
| `/plugins` | GET/POST | 查看 / 安装插件 |
| `/functions` | GET | 查看已注册函数 |
| `/sinks` / `/sources` | GET | 查看目标 / 源插件 |

### 七、CLI 常用命令

```bash
bin/kuiper create stream demo '...'     # 创建流
bin/kuiper drop stream demo             # 删除流
bin/kuiper show streams                 # 列出所有流
bin/kuiper create rule myRule '...'     # 创建规则
bin/kuiper getstatus rule myRule        # 查询规则状态
bin/kuiper query                        # 进入交互式查询
bin/kuiper plugin install ...           # 安装插件
```

### 八、安全与可靠性（本分支已加固）
- **依赖全面升级**：`golang.org/x/text`、`x/net`、`x/crypto`、`klauspost/compress`、`go.opentelemetry.io/otel` 等均已升级至修复 CVE 的版本。
- **工具链**：`go.work` 与 `go.mod` 固定 `toolchain go1.25.13`，修复标准库（crypto/tls、net/http、x509 等）已知漏洞。
- **核心代码 govulncheck 扫描 0 漏洞**（不依赖 cgo）。

---

> 本项目基于开源项目 **[LF Edge eKuiper](https://github.com/lf-edge/ekuiper)** 构建，感谢上游社区与所有贡献者的工作。