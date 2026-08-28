<div align="center">
  <img src="Logo.png" alt="B2C" width="120"/>
  <h2>B2C</h2>
  <h3>An Ultra-Lightweight IoT Edge Streaming Analytics Engine</h3>
</div>

### 1. Introduction
- **B2C** is an enhanced fork of [LF Edge eKuiper](https://github.com/lf-edge/ekuiper), an **IoT edge streaming data analytics engine** that runs on a wide range of resource-constrained hardware.
- Provides a **real-time stream computing framework** similar to [Apache Flink](https://flink.apache.org), enabling data ingestion, transformation, analysis, alerting and forwarding at the edge.
- Quickly build IoT edge analytics applications with **SQL rules** or **Graph rules** (similar to Node-RED).
- Targets IIoT, IoV (Internet of Vehicles), smart energy and other scenarios, significantly reducing response latency, bandwidth and storage costs, and improving system security.

### 2. Key Features
| Feature | Description |
|---------|-------------|
| Ultra-lightweight | Core service package ~4.5MB, first-run memory ~10MB |
| Cross-platform | X86/ARM 32/64, PPC; Linux, OpenWrt, macOS, Docker; industrial PCs, Raspberry Pi, industrial/home gateways, MEC edge clouds |
| Full data analytics | Data extraction/transformation/filtering (ETL), sorting, grouping, aggregation, joins; 60+ built-in functions; 4 types of time windows + count windows |
| Highly extensible | Extend `Source`, `Sink` and `UDF` with **Golang / Python** |
| Easy management | Manage streams/rules/plugins via CLI, REST API, Kubernetes config maps; optional Web management console |
| Ecosystem integration | Seamless integration with [EMQX](https://www.emqx.io/), [Neuron](https://neugates.io/), [NanoMQ](https://nanomq.io/), providing IIoT / IoV end-to-end solutions |

### 3. Quick Start

#### 1. Docker (Recommended)

```bash
docker run -p 9081:9081 -d --name kuiper \
  -e MQTT_SOURCE__DEFAULT__SERVER="tcp://broker.emqx.io:1883" \
  fasteredge/b2c:latest

# Enter the container
docker exec -it kuiper /bin/sh
```

#### 2. Local Build

```bash
# Environment requirement: go.mod / go.work declares toolchain go1.25.13 (with standard-library CVEs fixed)
go mod tidy

# Build server and CLI
go build -o bin/kuiperd ./cmd/kuiperd
go build -o bin/kuiper ./cmd/kuiper

# Start the service (REST port defaults to 9081)
./bin/kuiperd
```

#### 3. 5-Minute Getting Started

```shell
# Create a stream (like creating a DB table): subscribe to device messages
bin/kuiper create stream demo '(temperature float, humidity bigint) WITH (FORMAT="JSON", DATASOURCE="devices/+/messages")'

# Interactive query
bin/kuiper query
kuiper > select * from demo where temperature > 30;

# Publish a message to devices/device_001/messages with any MQTT client
# mqttx pub -h broker.emqx.io -m '{"temperature": 40, "humidity" : 20}' -t devices/device_001/messages

# Matching data is printed in the query window in real time
kuiper > [{"temperature": 40, "humidity" : 20}]
```

### 4. Architecture and Components

```
B2C/
├─ cmd/
│  ├─ kuiperd/     # Server daemon
│  └─ kuiper/      # Command-line tool (CLI)
├─ internal/
│  ├─ topo/        # Rule topology and execution
│  ├─ xsql/        # SQL engine and stream processing
│  ├─ server/      # REST API service
│  └─ processor/   # Stream/rule/plugin management
├─ pkg/            # Common infrastructure (ast, cast, kv, store etc.)
├─ extensions/     # Extensions: source / sink / functions
├─ plugins/        # Pluggable components (portable)
├─ etc/            # Configuration (kuiper.yaml, MQTT, connections etc.)
└─ docs/           # Chinese and English documentation
```

### 5. Configuration

Core configuration lives in `etc/kuiper.yaml`:

| Config key | Default | Description |
|------------|---------|-------------|
| `basic.restPort` | `9081` | REST API service port |
| `basic.consoleLog` | `false` | Console log switch |
| `basic.fileLog` | `true` | File log switch |
| `source` | — | Connection configuration for various data sources (MQTT, EdgeX, HTTP etc.) |
| `sink` | — | Configuration for various sinks (MQTT, file, REST, InfluxDB etc.) |

Common environment variables (Docker):

| Environment variable | Description |
|----------------------|-------------|
| `MQTT_SOURCE__DEFAULT__SERVER` | Default MQTT source server address |
| `KUIPER__BASIC__RESTPORT` | Overrides the REST port |
| `KUIPER__BASIC__CONSOLELOG` | Overrides the console log switch |

### 6. REST API

The service listens on `9081` by default and provides complete stream / rule / plugin management endpoints:

| Path | Method | Description |
|------|--------|-------------|
| `/streams` | GET/POST | List / create streams |
| `/streams/{name}` | GET/DELETE | View / delete the specified stream |
| `/rules` | GET/POST | List / create rules |
| `/rules/{id}` | GET/DELETE | View / delete the specified rule |
| `/rules/{id}/status` | GET | Query rule runtime status |
| `/rules/{id}/start` `/stop` | POST | Start / stop rule |
| `/plugins` | GET/POST | List / install plugins |
| `/functions` | GET | List registered functions |
| `/sinks` / `/sources` | GET | List sink / source plugins |

### 7. Common CLI Commands

```bash
bin/kuiper create stream demo '...'     # Create a stream
bin/kuiper drop stream demo             # Delete a stream
bin/kuiper show streams                 # List all streams
bin/kuiper create rule myRule '...'     # Create a rule
bin/kuiper getstatus rule myRule        # Query rule status
bin/kuiper query                        # Enter interactive query mode
bin/kuiper plugin install ...           # Install a plugin
```

### 8. Security and Reliability (Hardened in This Fork)
- **Dependencies fully upgraded**: `golang.org/x/text`, `x/net`, `x/crypto`, `klauspost/compress`, `go.opentelemetry.io/otel` etc. have all been upgraded to CVE-fixed versions.
- **Toolchain**: `go.work` and `go.mod` pin `toolchain go1.25.13`, fixing known vulnerabilities in the standard library (crypto/tls, net/http, x509 etc.).
- **Core code has 0 vulnerabilities in govulncheck scans** (no cgo dependency).

---

> This project is built on the open-source project **[LF Edge eKuiper](https://github.com/lf-edge/ekuiper)**, with thanks to the upstream community and all contributors.