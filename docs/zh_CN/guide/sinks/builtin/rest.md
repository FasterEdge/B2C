# REST动作

该动作用于将输出消息发布到 RESTful API 中。

| 属性名称          | 是否可选 | 说明                                                                                                                                                                                                                                       |
|---------------|------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| method        | 是    | RESTful API 的 HTTP 方法。 这是一个不区分大小写的字符串，其值范围为"get"，"post"，"put"，"patch"，"delete" 和 "head"。 默认值为 "get"，支持动态获取。                                                                                                                              |
| url           | 否    | RESTful API 终端地址，例如 `https://www.example.com/api/dummy`，支持动态获取。                                                                                                                                                                          |
| bodyType      | 是    | 消息体的类型。 当前，支持以下类型："none", "json", "text", "html", "xml", "javascript", "form", "binary" 和 "formdata"。 对于 "get" 和 "head"，不需要正文，因此默认值为 "none"。 对于其他 http 方法，默认值为 "json"。对于 "html"，"xml" 和 "javascript"，必须仔细设置 dataTemplate 以确保格式正确。支持动态获取。 |
| formdata      | true | 设置 bodyType 为 formdata 时可用。本属性用于设置表单数据的键值对。编码后的内容（字节形式）将作为文件附件传输，而每个键值对代表multipart表单的一个部分                                                                                                                                                |
| fileFieldName | true | 设置 bodyType 为 formdata 时可用，定义表单提交中文件部分的参数名称                                                                                                                                                                                              |

| timeout | 是 | HTTP 请求超时的时间（毫秒），默认为5000毫秒 |
| headers | 是 | 要为 HTTP 请求设置的其它 HTTP 头。支持动态获取。 |
| debugResp | 是 | 控制是否将响应信息打印到控制台中。 如果将其设置为 `true`，则打印响应。 如果设置为`false`
，则跳过打印日志。默认值为 `false`。 |
| response | 是 | 响应回传配置，用于实现"请求-响应"网关闭环：HTTP 请求完成后，将响应体发布到指定的 MQTT 主题。配置项包括 `type`（目前仅支持 `mqtt`）与 `mqtt`（MQTT 回传配置）。详情请见[响应回传（请求-响应网关）](#响应回传请求-响应网关)。 |
| certificationPath | 是 | 证书路径。可以为绝对路径，也可以为相对路径。如果指定的是相对路径，那么父目录为执行 `kuiperd`
命令的路径。比如，如果你在 `/var/kuiper` 中运行 `bin/kuiperd` ，那么父目录为 `/var/kuiper`; 如果运行从 `/var/kuiper/bin`
中运行`./kuiperd`，那么父目录为 `/var/kuiper/bin`。 |
| privateKeyPath | 是 | 私钥路径。可以为绝对路径，也可以为相对路径，相对路径的用法与 `certificationPath` 类似。 |
| rootCaPath | 是 | 根证书路径，用以验证服务器证书。可以为绝对路径，也可以为相对路径，相对路径的用法与 `certificationPath`
类似。 |
| insecureSkipVerify | 是 | 控制是否跳过证书认证。如果被设置为 `true`，那么跳过证书认证；否则进行证书验证。缺省为 `true`。 |
| oAuth | 是 | 定义类 OAuth 的认证流程。其他的认证方式如 apikey 可以直接在 headers
设置密钥，不需要使用这个配置。详情请见[OAuth 配置](../../sources/builtin/http_pull.md#OAuth)。 |

其他通用的 sink 属性也支持，请参阅[公共属性](../overview.md#公共属性)。

::: v-pre
REST 服务通常需要特定的数据格式。 这可以由公共目标属性 `dataTemplate` 强制使用。 请参考[数据模板](../data_template.md)。 以下是用于连接到 Edgex Foundry core 命令的示例配置。dataTemplate`{{.key}}` 表示将打印出键值，即 result [key]。 因此，这里的模板是在结果中仅选择字段 `key`，并将字段名称更改为 `newKey`。`sendSingle` 是另一个常见属性。 设置为 true 表示如果结果是数组，则每个元素将单独发送。
:::

```json
{
  "rest": {
    "url": "http://127.0.0.1:59882/api/v1/device/cc622d99-f835-4e94-b5cb-b1eff8699dc4/command/51fce08a-ae19-4bce-b431-b9f363bba705",
    "method": "post",
    "dataTemplate": "\"newKey\":\"{{.key}}\"",
    "sendSingle": true
  }
}
```

使用 OAuth 风格鉴权的示例:

OAuth Header 模板仅支持简单占位符 <code v-pre>{{.access_token}}</code>、<code v-pre>{{.refresh_token}}</code>、<code v-pre>{{.token_type}}</code>、<code v-pre>{{.id_token}}</code> 和 <code v-pre>{{.expires_in}}</code>，字段名必须与 token 接口返回的 JSON 字段完全一致。其他占位符，例如 <code v-pre>{{.message}}</code>、<code v-pre>{{.scope}}</code> 和 <code v-pre>{{.custom_token}}</code>，只会根据规则输出求值，不能读取 token 响应字段。因此，token 接口必须返回 Header 配置所使用的受支持字段名。单个 Header 不能同时包含 OAuth 占位符和规则输出模板，但不同 Header 可以分别使用 OAuth 和规则输出模板。

```json
{
  "id": "ruleFollowBack",
  "sql": "SELECT follower FROM followStream",
  "actions": [
    {
      "rest": {
        "url": "https://com.awebsite/follows",
        "method": "POST",
        "sendSingle": true,
        "bodyType": "json",
        "dataTemplate": "{\"data\":{\"relationships\":{\"follower\":{\"data\":{\"type\":\"users\",\"id\":\"1398589\"}},\"followed\":{\"data\":{\"type\":\"users\",\"id\":\"{{.follower}}\"}}},\"type\":\"follows\"}}",
        "headers": {
          "Content-Type": "application/vnd.api+json",
          "Authorization": "Bearer {{.access_token}}"
        },
        "oAuth": {
          "access": {
            "url": "https://com.awebsite/oauth/token",
            "body": "{\"grant_type\": \"password\",\"username\": \"user@gmail.com\",\"password\": \"mypass\"}",
            "expire": "3600"
          }
        }
      }
    }
  ]
}
```

Visualization mode
以可视化图形交互创建 rules 的 SQL 和 Actions

Text mode
以json格式创建 rules 的 SQL 和 Actions

创建写 taosdb rest示例：

```json
{
  "id": "rest1",
  "sql": "SELECT tele[0].Tag00001 AS temperature, tele[0].Tag00002 AS humidity FROM neuron",
  "actions": [
    {
      "rest": {
        "bodyType": "text",
        "dataTemplate": "insert into mqtt.kuiper values (now, {{.temperature}}, {{.humidity}})",
        "debugResp": true,
        "headers": {
          "Authorization": "Basic cm9vdDp0YW9zZGF0YQ=="
        },
        "method": "POST",
        "sendSingle": true,
        "url": "http://xxx.xxx.xxx.xxx:6041/rest/sql"
      }
    }
  ]
}
```

## 设置动态输出参数

很多情况下，我们需要根据结果数据，决定写入的目的地址和参数。在 REST sink 里，`method`，`url`，`bodyType` 和 `headers` 支持动态参数。动态参数可通过数据模板语法配置。接下来，让我们使用动态参数改写上例。假设我们收到了数据中包含了 http 方法和 url 后缀等元数据。我们可以通过改写 SQL 语句，在输出结果中得到这两个值。规则输出的单条数据类似：

```json
{
  "method": "post",
  "url": "http://xxx.xxx.xxx.xxx:6041/rest/sql",
  "temperature": 20,
  "humidity": 80
}
```

在规则 action 中，可以通过数据模板语法取得结果数据作为属性变量。如下例子中，`method` 和 `url` 为动态变量。

```json
{
  "id": "rest2",
  "sql": "SELECT tele[0]->Tag00001 AS temperature, tele[0]->Tag00002 AS humidity, method, concat(\"http://xxx.xxx.xxx.xxx:6041/rest/sql\", urlPostfix) as url FROM neuron",
  "actions": [
    {
      "rest": {
        "bodyType": "text",
        "dataTemplate": "insert into mqtt.kuiper values (now, {{.temperature}}, {{.humidity}})",
        "debugResp": true,
        "headers": {
          "Authorization": "Basic cm9vdDp0YW9zZGF0YQ=="
        },
        "method": "{{.method}}",
        "sendSingle": true,
        "url": "{{.url}}"
      }
    }
  ]
}
```

## 文件上传

如需将数据通过文件形式上传至HTTP服务器，可采用`bodyType=formdata`配置方式。

**核心特性**：

- Content type 采用 "multipart/form-data"
- 规则生成的二进制数据将作为表单文件内容上传
- 可通过`formData`配置其他表单属性

**最佳实践**：

- 对于高频数据源，应配置批量或窗口聚合功能，避免频繁上传小文件

**配置示例**：

```json
{
  "id": "restUpload",
  "sql": "SELECT value1, value2 FROM neuron",
  "actions": [
    {
      "rest": {
        "url": "http://yoururlhere.com",
        "method": "post",
        "fileFieldName": "file1",
        "formData": {
          "key1": "value1",
          "key2": "value2"
        },
        "batchSize": 10,
        "format": "delimited",
        "sendSingle": true
      }
    }
  ]
}
```

本示例中，每10条记录将生成一个 CSV 文件上传。

## 响应回传（请求-响应网关）

`response` 配置项使得 REST sink 可以充当"请求-响应网关"的响应通道：当 HTTP 请求成功完成后，REST sink 会把 HTTP 响应体发布到配置的 MQTT 主题，从而把响应送回给请求方。该能力与 MQTT v5 的请求-响应机制（`ResponseTopic` / `CorrelationData` 属性）配合，可以搭建完整的请求转发与转换闭环。

**配置结构**：

| 属性名称                | 是否可选 | 说明                                                                                                  |
|---------------------|------|-----------------------------------------------------------------------------------------------------|
| response.type       | 否    | 回传类型，目前仅支持 `mqtt`。                                                                              |
| response.mqtt       | 否    | MQTT 回传配置，具体属性如下。                                                                                  |
| response.mqtt.server       | 否    | MQTT broker 地址，例如 `tcp://127.0.0.1:1883`。                                                             |
| response.mqtt.protocolVersion | 是    | MQTT 协议版本：`3.1`、`3.1.1`、`4` 或 `5`。默认值为 `3.1.1`。                                                   |
| response.mqtt.clientId      | 是    | MQTT 客户端 ID，默认自动生成。                                                                                  |
| response.mqtt.username      | 是    | MQTT 用户名。                                                                                         |
| response.mqtt.password      | 是    | MQTT 密码。                                                                                           |
| response.mqtt.topic         | 否    | 回传主题，支持动态参数，例如 `{{.responseTopic}}`，可从规则输出中解析请求携带的响应主题。                                            |
| response.mqtt.correlationData | 是    | 回传关联数据，支持动态参数，例如 `{{.correlationData}}`。当协议版本为 `5` 时，会写入 MQTT v5 标准的 `CorrelationData` 属性。             |
| response.mqtt.qos           | 是    | MQTT QoS 级别，默认值为 `0`。                                                                               |
| response.mqtt.retained      | 是    | 是否保留消息，默认值为 `false`。                                                                              |

**工作原理**：

1. 请求方（例如 MQTT v5 客户端）发布请求消息，消息的 `ResponseTopic` 属性声明了期望接收响应的主题。
2. MQTT source 将 `ResponseTopic`、`CorrelationData` 等属性提取为元数据，规则通过 `meta()` 函数（例如 `meta(responseTopic)`）将其转为输出字段。
3. 规则转换后，REST sink 以 HTTP 方式转发请求到目标服务。
4. HTTP 请求成功后，REST sink 根据 `response` 配置，把 HTTP 响应体发布到 `response.mqtt.topic` 解析出的主题，并将关联数据一并回传。
5. 请求方在其 `ResponseTopic` 上收到 HTTP 响应。

**配置示例**：

以下示例中，MQTT v5 客户端向 `device/commands` 主题发送请求（携带 `ResponseTopic` 与 `CorrelationData` 属性），规则将其转换为 HTTP POST 请求转发到目标 API，并把 HTTP 响应通过 MQTT 发布回请求方的 `ResponseTopic`。

```json
{
  "id": "mqttHttpGateway",
  "sql": "SELECT command, id, meta(responseTopic) AS responseTopic, meta(correlationData) AS correlationData FROM device/commands",
  "actions": [
    {
      "rest": {
        "url": "http://target-service/api/execute",
        "method": "post",
        "bodyType": "json",
        "sendSingle": true,
        "response": {
          "type": "mqtt",
          "mqtt": {
            "server": "tcp://127.0.0.1:1883",
            "protocolVersion": "5",
            "topic": "{{.responseTopic}}",
            "correlationData": "{{.correlationData}}",
            "qos": 1
          }
        }
      }
    }
  ]
}
```

**注意事项**：

- `response.mqtt.topic` 与 `response.mqtt.correlationData` 支持动态参数。要使用它们，需在规则的 SQL 中通过 `meta()` 函数（如 `meta(responseTopic)`)把 MQTT v5 请求属性暴露为输出字段，再以 `{{.字段名}}` 形式引用。
- 仅有 HTTP 响应体非空时才会回传；HTTP 请求失败或返回非 2xx 状态码时不会回传。
- 若请求方不携带 `ResponseTopic`，且 `topic` 未配置静态主题，回传会因主题为空而失败并记录错误日志。
- 回传的 MQTT 连接按规则独立创建，规则停止时自动释放。
