<div align="center">
  <img src="Logo.png" alt="B2C" width="120"/>
  <h2>B2C</h2>
  <h3>An eKuiper Sync Fork with an MQTT v5 Request-Response Gateway</h3>
</div>

**Language：** [简体中文](./README.md) | [English](./README_en.md) | [中文导览](./README-CN.md)

> This repository used to keep an unmodified upstream Japanese doc, `README-JP.md`. That file
> describes the project as "LF Edge eKuiper" and never recorded the fork's actual nature, so
> keeping it was misleading and it has been deleted. Japanese readers should use the upstream
> repository <https://github.com/lf-edge/ekuiper>.

### 1. What This Is (Read This First)

- **B2C is a sync fork of [LF Edge eKuiper](https://github.com/lf-edge/ekuiper)**, not a reimplementation.
  The divergence point is commit `fce9a2e1` (upstream `feat: support array_avg (#4108)`, 2026-08-18).
- From the divergence point to HEAD there are **55 own commits**, adding **632 lines net** (1277 net lines in Go code).
  Against upstream's ~126,000 tracked lines (`git ls-files | xargs wc -l` reports 125418 lines) that is about **0.50%**.
  Therefore **more than 99% of this repository's capability comes from upstream eKuiper** — do not read B2C as an in-house engine.
- The only self-developed feature is the **MQTT v5 request-response gateway** (section 6).
  The remaining own commits are security fixes, build/CI adaptations and documentation corrections.
- The fork's own CI is `.github/workflows/check.yml`, which runs only `go build ./cmd/...`.
  The upstream eKuiper workflows are kept as synced content; most jobs carry repository guards
  and do not run in this fork. That file's header comment also states explicitly
  "this repository is a sync copy of lf-edge/ekuiper".
- **The module path is still upstream's `github.com/lf-edge/ekuiper/v2`** (`go.mod:424` declares
  `go 1.27.4`). Renaming it would touch thousands of imports for little gain and high risk,
  so it was deliberately left alone.

#### Scale and Commit Composition (verify it yourself)

```bash
# Own commit count, change size, and total tracked lines relative to the upstream divergence point
git rev-list --count fce9a2e1..HEAD        # => 55
git diff --shortstat fce9a2e1..HEAD        # => 140 files changed, 3947 insertions(+), 3315 deletions(-)
git ls-files | wc -l                       # => 1859 (file count, NOT line count)
git ls-files | xargs wc -l                 # final "total" line: 125418 (line count)

# Net line change in Go code only
git diff --numstat fce9a2e1..HEAD | awk '$3 ~ /\.go$/ {a+=$1; d+=$2} END {print "net", a-d}'   # => net 1277
```

> Note: `git ls-files | wc -l` counts tracked **files** (1859), not lines.
> The line-count figure comes from `git ls-files | xargs wc -l` (total 125418).
> "632 lines net" = 3947 insertions − 3315 deletions from `git diff --shortstat`;
> the share is 632 / 125418 ≈ 0.50%.

### 2. Capability Attribution: Upstream vs. This Fork

Every entry below is tagged to avoid misreading. **Anything not tagged [FORK ADDITION] is an
upstream eKuiper capability.**

#### [UPSTREAM] Ingestion and Routing

- Rule topology and execution (`internal/topo`), SQL engine (`internal/xsql`).
- Built-in sources: MQTT, EdgeX, HTTP pull/push, File, Simulator, Video, etc.
- Built-in sinks: MQTT, File, REST, InfluxDB v1/v2, Kafka, Redis, TDengine, etc.
- Compiled sources/sinks shipped in-tree (`extensions/sources`, `extensions/sinks`):
  sources `kafka`, `sql`, `video`, `zmq`; sinks `image`, `influx`, `influx2`, `kafka`, `sql`,
  `tdengine3`, `zmq`.
- Pluggable Source / Sink / SQL function, with both **Golang and Python** plugin mechanisms.

#### [UPSTREAM] Computation and Query

- SQL-based stream processing: filter, project, group/aggregate, windows, multi-stream JOIN,
  `GROUP BY` / `HAVING`.
- Window types are defined at `pkg/ast/statement.go:180-188` — **6 kinds**:
  `TUMBLING` / `HOPPING` / `SLIDING` / `SESSION` / `COUNT` / `STATE`.
  (The familiar "4 time windows plus a count window" phrasing is the older upstream wording;
  it predates `STATE_WINDOW`.)
- Built-in SQL functions: **192** distinct names are registered across
  `internal/binder/function/funcs_*.go` (math, string, aggregate, array, object, type conversion,
  datetime, window functions). Reproduce with:

  ```bash
  grep -rhoE 'builtins\["[a-zA-Z_0-9]+"\]' internal/binder/function/funcs_*.go \
    | sed 's/builtins\["//;s/"\]//' | sort -u | wc -l   # => 192
  ```

  > **Counting rule (important):** this glob also pulls in the sibling `funcs_*_test.go`
  > files — the 15 test files contain 91 `builtins["..."]` references covering 68 names,
  > every one of which is a subset of names already registered in non-test files, so the
  > deduplicated count is unchanged. Restricting strictly to non-test files
  > (`ls internal/binder/function/funcs_*.go | grep -v '_test\.go$'`) also yields
  > **192**, matching the figure above; the older docs' 191 was an undercount.

#### [UPSTREAM] Management

- CLI (`bin/kuiper`), REST API (default port `9081`), Kubernetes config maps.
- The REST route table is registered in `internal/server/rest.go`: `grep -c HandleFunc internal/server/rest.go`
  gives **62** `HandleFunc` routes, plus 4 pprof routes registered via `r.Handle(` (`/debug/pprof/goroutine`,
  `heap`, `threadcreate`, `block`), for **66** route registrations in total. They cover streams, tables,
  rules, connections, plugins, config import/export, rule tracing and metric dumps — far more than
  the dozen listed by older docs.

  ```bash
  grep -c HandleFunc internal/server/rest.go                    # => 62
  grep -cE 'r\.Handle(Func)?\(' internal/server/rest.go          # => 66
  ```
- The web management console (eKuiper Manager) is a **separate project** and is not in this repo.

#### [FORK ADDITION] MQTT v5 Request-Response Gateway

This fork customizes the REST sink and the MQTT v5 client to close the loop
"receive request → forward as HTTP → send the response back to the requester". See section 6.

#### [FORK ADDITION] Security Fixes (for pre-existing upstream defects)

All of the following **fix upstream defects**; none add attack surface.

| Area | Location | Issue |
|------|----------|-------|
| InfluxDB v2 line protocol injection | `extensions/impl/influx2/influx2.go` | Unescaped tag/field/string values |
| SQL injection | `extensions/impl/sql/sqldatabase/sqlgen/` | `quoteValue` did not escape single quotes |
| Remote DoS (out-of-range panic) | `extensions/impl/onnx/onnx.go` | Float16 branch used `make([]byte,0,4)`, length 0 |
| Remote DoS (OOM) | `extensions/functions/image/`, `extensions/impl/image/image.go` | No size cap on decode/resize; added `maxImageDim=8192` and `DecodeConfig` pre-checks |
| SQL injection | `extensions/impl/tdengine3/tdengine3.go` | Unescaped strings + unvalidated dynamic field names |
| ONNX/TFLite type conversion | `extensions/impl/onnx/onnx.go`, `extensions/impl/tflite/tfLite.go` | Int family passed the whole args array instead of the current tensor |
| Native plugin symbol table pollution | `internal/plugin/native/manager.go` | Partial `storeSymbols` write permanently polluted the symbol table on failure |
| Startup nil dereference | `internal/plugin/native/` | `findAll` was not given the registry, so startup plugin loading dereferenced a nil manager |
| MQTT empty subscribe | `internal/io/mqtt/v5client/client.go` | An empty SUBSCRIBE packet was still sent when all topics were subscribed, which brokers drop per spec |
| Checkpoint state snapshot | `internal/topo/operator/` | Snapshot operator state before writing the checkpoint store |

#### [FORK ADDITION] Build and CI Adaptation

- **cgo dependency removed**: replaced `pebbe/zmq4` with the pure-Go `go-zeromq/zmq4`,
  removing the libzmq requirement for cross-compilation (commit `a2ad93dd`).
- Dependencies broadly upgraded to eliminate known CVEs; the Go toolchain tracks upstream
  (currently `go 1.27.4`).
- Added `.github/workflows/check.yml` (only `go build ./cmd/...`), `.github/dependabot.yml`
  and `SECURITY.md`.
- Added `.dockerignore` to keep `.git`, `.github` and docs out of the Docker build context.
- Dockerfiles allow a Go toolchain switch; the chart templates and values under
  `deploy/chart/ekuiper/` were adjusted for the fork's image name and build paths (commit `bb252019`).

  > **About the chart version numbers (not a bug):** `deploy/chart/ekuiper/Chart.yaml:17,21` carries
  > `version: 1.4.0` / `appVersion: 2.3.0-beta.5`, which are **upstream eKuiper's Helm chart version**
  > numbers. They belong to a separate versioning scheme from this repository's git tags
  > (currently `v1.0.20261116`) and have no relation to them. The chart is a leftover upstream asset;
  > this fork **does not publish it** (no workflow does so — see the CI note in section 2). Only the
  > templates and values are reused here for build-path alignment. Readers should not read the
  > mismatched numbers as a version inconsistency.

### 3. An Honest Note on "Ultra-Lightweight"

Upstream docs long claimed "core server package ~4.5MB, first-run memory ~10MB".
**This fork could not reproduce that figure**, so the README no longer repeats it:

| Build | Command | Measured size |
|-------|---------|---------------|
| `kuiperd` windows/amd64 | `CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o b2cd cmd/kuiperd/main.go` | **41.02 MB** |
| `kuiperd` linux/amd64 | same, plus `GOOS=linux GOARCH=amd64` | **40.32 MB** |
| `kuiperd` linux/arm64 | same, plus `GOOS=linux GOARCH=arm64` | **37.38 MB** |

- The gap comes mostly from years of accumulated built-in plugins and dependencies
  (InfluxDB/Kafka/Redis/TDengine clients, ONNX/TFLite runtime bindings, the script engine, etc.),
  not from anything this fork introduced.
- Getting back to "~4.5MB" would require a minimal build-tag combination (e.g. the upstream
  `make pkg_prebuild` style); this repository provides no such configuration.
- Memory usage depends strongly on rule workload. **This fork ran no performance benchmarks at
  all**, so no memory figures are claimed here.

### 4. Quick Start

#### 1. Docker

> Note: the `fasteredge/b2c:latest` image **is not published by this repository**
> (no workflow pushes to that name). Use the upstream image `lfedge/ekuiper:latest`,
> or build your own.

```bash
docker run -p 9081:9081 -d --name kuiper \
  -e MQTT_SOURCE__DEFAULT__SERVER="tcp://broker.emqx.io:1883" \
  -e KUIPER__BASIC__RESTIP=0.0.0.0 \
  lfedge/ekuiper:latest

docker exec -it kuiper /bin/sh
```

#### 2. Local Build

```bash
# Environment requirement: go.mod / go.work declares go 1.27.4 (go.work:1)
go build -o bin/kuiperd ./cmd/kuiperd
go build -o bin/kuiper ./cmd/kuiper

# Start the service (REST port defaults to 9081)
./bin/kuiperd
```

> `go build ./cmd/...` is this repository's only build acceptance command
> (see `.github/workflows/check.yml`). `go build ./...` does not apply here:
> `extensions/*` are Go plugins (`main` packages without a `main` function).

#### 3. Five-Minute Getting Started

```shell
bin/kuiper create stream demo '(temperature float, humidity bigint) WITH (FORMAT="JSON", DATASOURCE="devices/+/messages")'

bin/kuiper query
kuiper > select * from demo where temperature > 30;

# Publish a message to devices/device_001/messages with any MQTT client
# mqttx pub -h broker.emqx.io -m '{"temperature": 40, "humidity" : 20}' -t devices/device_001/messages
```

> Note: the example above depends on the public broker `broker.emqx.io`. Point it at a local
> broker if that host is unreachable.

### 5. Architecture and Components

```
B2C/
├─ cmd/
│  ├─ kuiperd/     # Server daemon
│  └─ kuiper/      # Command-line tool (CLI)
├─ internal/
│  ├─ topo/        # Rule topology and execution
│  ├─ xsql/        # SQL engine and stream processing
│  ├─ server/      # REST API service (route table in internal/server/rest.go)
│  └─ processor/   # Stream/rule/plugin management
├─ pkg/            # Common infrastructure (ast, cast, kv, store etc.)
├─ extensions/     # Built-in extensions: source / sink / functions
├─ plugins/        # Runtime plugin install targets (contents gitignored, see below)
├─ etc/            # Configuration (kuiper.yaml, MQTT, connections etc.)
├─ sdk/            # Golang / Python plugin SDK and examples
└─ docs/           # Chinese and English documentation
```

**About the empty `plugins/` directories**: `plugins/sources`, `plugins/sinks` and
`plugins/functions` are empty on purpose — `.gitignore:17-19` ignores their contents, because
they are **runtime install targets**, not an example code base. Runnable plugin examples live in:

- `sdk/go/example/` (Golang plugin examples such as `mirror`, `random`)
- `sdk/python/example/` (Python plugin examples)
- `plugins/portable/readme.md` (portable plugin directory layout and loading rules)

`etc/sources/randomGo.json`, `etc/sinks/fileGo.json` and `etc/functions/echoGo.json` are
compiled-plugin declaration examples added by this fork (commit `7e966c57`); use them as a
field-format reference.

### 6. The Self-Developed Feature: MQTT v5 Request-Response Gateway

This is the only functional addition in this fork. It closes a loop upstream does not provide.

**Problem**: a device issues a request carrying MQTT v5 `ResponseTopic` / `CorrelationData`
properties; the edge must turn that into an HTTP call to a business system and route the HTTP
response back to the requester. Upstream eKuiper's REST sink only sends messages to HTTP
one-way and cannot return a response.

**Where the code lives**:

| File | Role |
|------|------|
| `internal/io/mqtt/v5client/client.go:279` onwards | `ParseMsg` extracts `ResponseTopic`, `CorrelationData` and `ContentType` from MQTT v5 packets into metadata, exposed to SQL `meta()` |
| `internal/io/mqtt/v5client/client.go:186` onwards | `Publish` maps `correlationData` / `responseTopic` back onto standard MQTT v5 publish properties; anything else becomes a user property |
| `internal/io/http/client.go:112` onwards | New `Response` / `ResponseConf` / `ResponseMqttConf` configuration structs |
| `internal/io/http/rest_sink.go:48` onwards | `responseMqttPublisher`: resolves the reply topic per message and publishes the HTTP response back to MQTT |
| `internal/io/http/rest_sink.go:477` onwards | `buildEnvelope`: optionally wraps `status` / `headers` into the relayed body |

**Configuration shape** (full docs: `docs/en_US/guide/sinks/builtin/rest.md`):

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

**Boundaries and what was NOT done (state these honestly)**:

- MQTT is the only relay channel supported (`type` currently accepts only `mqtt`).
- `forwardStatus` / `forwardHeaders` / `forwardErrors` / `forwardEmpty` all default to `false`,
  preserving upstream's original raw-body relay behavior so existing users are not broken.
- **No performance evaluation was done**: there are no throughput, latency or resource-usage
  numbers, and no comparison against alternatives (HTTP long polling, WebSocket, etc.).
  A thesis that needs performance metrics must design that experiment separately.
- QoS 1/2 publish-acknowledgement retry semantics and persistence of relayed messages are
  not implemented.

### 7. Configuration

Core configuration lives in `etc/kuiper.yaml`:

| Config key | Default | Description |
|------------|---------|-------------|
| `basic.restPort` | `9081` | REST API service port |
| `basic.restIp` | `127.0.0.1` | REST bind address |
| `basic.ip` / `basic.port` | `127.0.0.1` / `20498` | CLI service bind address and port |
| `basic.consoleLog` | `false` | Console log switch |
| `basic.fileLog` | `true` | File log switch |
| `basic.authentication` | `false` | REST API authentication switch |
| `basic.allowExternalFileAccess` | `false` | Whether files outside the upload directory may be read |
| `source.httpServerIp` | `127.0.0.1` | HTTP data ingestion bind address |
| `source` | — | Connection configuration for various sources |
| `sink` | — | Connection configuration for various sinks |

**On binding to `127.0.0.1` by default**: that hardening comes from upstream commit
`60e16db7` (`refactor(conf): harden default security (#4089)`, 2026-07-29), which predates the
divergence point `fce9a2e1` (2026-08-18). It is therefore an upstream capability, not a
fork achievement. `RELEASE_NOTES.md` describes it as a behavior change, and this README keeps
that same factual framing.

Common environment variables (Docker): `MQTT_SOURCE__DEFAULT__SERVER`,
`KUIPER__BASIC__RESTPORT`, `KUIPER__BASIC__RESTIP`, `KUIPER__BASIC__ALLOWEXTERNALFILEACCESS`.

### 8. REST API

The service listens on `9081` by default. The full route table is at
`internal/server/rest.go:190-290`; commonly used endpoints:

| Path | Method | Description |
|------|--------|-------------|
| `/streams` | GET/POST | List / create streams |
| `/streams/{name}` | GET/DELETE/PUT | View / delete / update a stream |
| `/tables` | GET/POST | List / create tables |
| `/rules` | GET/POST | List / create rules |
| `/rules/{name}/status` | GET | Query rule runtime status |
| `/rules/{name}/start` `/stop` | POST | Start / stop a rule |
| `/rules/validate` | POST | Validate a rule definition |
| `/connections` | GET/POST | List / create connections |
| `/plugins` | GET/POST | List / install plugins |
| `/functions` | GET | List registered functions |
| `/sinks` / `/sources` | GET | List sink / source plugins |
| `/data/export` `/data/import` | GET/POST | Configuration export / import |

### 9. Common CLI Commands

```bash
bin/kuiper create stream demo '...'     # Create a stream
bin/kuiper drop stream demo             # Delete a stream
bin/kuiper show streams                 # List all streams
bin/kuiper create rule myRule '...'     # Create a rule
bin/kuiper getstatus rule myRule        # Query rule status
bin/kuiper query                        # Enter interactive query mode
bin/kuiper plugin install ...           # Install a plugin
```

### 10. Security and Reliability

- **Dependency upgrades**: `golang.org/x/text`, `x/net`, `x/crypto`, `klauspost/compress`,
  `go.opentelemetry.io/otel` and others moved to CVE-fixed versions (commit `d6603d7c` and later).
- **Toolchain**: `go.mod:424` and `go.work:1` declare `go 1.27.4`.
- **govulncheck**: scanning `./cmd/...` and `./internal/io/...` reported
  `No vulnerabilities found.` (command below).
- **Default localhost binding** and **default external-file-access denial**: both come from
  upstream commit `60e16db7`, not from this fork.
- **Production note**: the REST API has no authentication by default
  (`authentication: false`). Once `restIp` is changed to `0.0.0.0` and exposed to a network,
  authentication and TLS must be enabled.

```bash
# Dependency vulnerability scan (the exact command this fork ran)
govulncheck ./cmd/... ./internal/io/...
```

---

> This project is built on the open-source project **[LF Edge eKuiper](https://github.com/lf-edge/ekuiper)**.
> Upstream is Apache License 2.0 and this fork retains its license and copyright notices.
> The fork's own code is a small fraction of the whole — treat the upstream repository's docs
> and community as the primary reference.
