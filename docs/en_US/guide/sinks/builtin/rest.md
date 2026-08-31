# REST action

The action is used for publish output message into a RESTful API.

| Property name        | Optional | Description                                                                                                                                                                                                                                                                                                                                                                                       |
|----------------------|----------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| method               | true     | The HTTP method for the RESTful API. It is a case insensitive string whose value is among "get", "post", "put", "patch", "delete" and "head". The default value is "get".                                                                                                                                                                                                                         |
| url                  | false    | The RESTful API endpoint, such as `https://www.example.com/api/dummy`                                                                                                                                                                                                                                                                                                                             |
| bodyType             | true     | The type of the body. Currently, these types are supported: "none", "json", "text", "html", "xml", "javascript", "form", "binary" and "formdata". For "get" and "head", no body is required so the default value is "none". For other http methods, the default value is "json" For "html", "xml" and "javascript", the dataTemplate must be carefully set up to make sure the format is correct. |
| timeout              | true     | The timeout (milliseconds) for a HTTP request, defaults to 5000 ms                                                                                                                                                                                                                                                                                                                                |
| headers              | true     | The additional headers to be set for the HTTP request.                                                                                                                                                                                                                                                                                                                                            |
| formdata             | true     | If bodyType is formdata, this property specifies key-value pairs for form data. The encoded body (in bytes) will be transmitted as a file. Each key-value pair represents one part of the multipart form.                                                                                                                                                                                         |
| fileFieldName        | true     | Specifies the form field name when uploading files via multipart/form-data                                                                                                                                                                                                                                                                                                                        |
| fileName             | true     | When bodyType is formdata, the file name of the multipart file attachment. Supports dynamic props (e.g. `{{.filename}}`). Defaults to a millisecond timestamp.                                                                                                                                                                                                                                     |
| query                | true     | Structured URL query parameters as key-value pairs, appended (not overwriting) to the URL query string. Supports dynamic props. Compared to embedding params in `url`, this property handles URL encoding automatically.                                                                                                                                                                            |
| metaHeaders          | true     | Maps source metadata onto outgoing HTTP headers: the key is the HTTP header name and the value is the metadata key (e.g. MQTT v5 `correlationData` / `responseTopic`, accessible via SQL `meta()`). The header is written only when the metadata entry exists, and the static headers configuration is never mutated.                                                                               |
| debugResp            | true     | Control if print the response information into the console. If set it to `true`, then print response; If set to `false`, then skip print log. The default is `false`.                                                                                                                                                                                                                             |
| certificationPath    | true     | The certification path. It can be an absolute path, or a relative path. If it is an relative path, then the base path is where you excuting the `kuiperd` command. For example, if you run `bin/kuiperd` from `/var/kuiper`, then the base path is `/var/kuiper`; If you run `./kuiperd` from `/var/kuiper/bin`, then the base path is `/var/kuiper/bin`.                                         |
| privateKeyPath       | true     | The private key path. It can be either absolute path, or relative path, which is similar to use of certificationPath.                                                                                                                                                                                                                                                                             |
| rootCaPath           | true     | The location of root ca path. It can be an absolute path, or a relative path, which is similar to use of certificationPath.                                                                                                                                                                                                                                                                       |
| tlsMinVersion        | true     | Specifies the minimum version of the TLS protocol that will be negotiated with the client. Accept values are `tls1.0`, `tls1.1`, `tls1.2` and `tls1.3`. Default: `tls1.2`.                                                                                                                                                                                                                        |
| renegotiationSupport | true     | Determines how and when the client handles server-initiated renegotiation requests. Support `never`, `once` or `freely` options. Default: `never`.                                                                                                                                                                                                                                                |
| insecureSkipVerify   | true     | Control if to skip the certification verification. If it is set to `true`, then skip certification verification; Otherwise, verify the certification. The default value is `true`.                                                                                                                                                                                                                |
| oAuth                | true     | Define the authentication flow to follow the OAuth style. Other authentication method like apikey can directly set the key to header only, not need to set this configuration. Refer to [OAuth configuration](../../sources/builtin/http_pull.md#OAuth) in httppull source for more information.                                                                                                  |
| response.mqtt.forwardStatus | true | When `true`, wrap the relay payload as `{status, body}` to preserve the HTTP status code. Defaults to `false` for legacy raw-body behavior. |
| response.mqtt.forwardHeaders | true | When `true`, add HTTP `headers` to the relay envelope. Defaults to `false`. |
| response.mqtt.forwardErrors | true | When `true`, relay non-2xx HTTP responses too. Defaults to `false`, preserving legacy sink error behavior. |
| response.mqtt.forwardEmpty | true | When `true`, relay an empty HTTP body as well. Defaults to `false`. |

Other common sink properties are supported. Please refer to the [sink common properties](../overview.md#common-properties) for more information.

::: v-pre
REST service usually requires a specific data format. That can be imposed by the common sink property `dataTemplate`.
Please check the [data template](../data_template.md). Below is a sample configuration for connecting to Edgex Foundry
core command. The dataTemplate <code v-pre>{{.key}}</code> means it will print out the value of key, that is
result[key]. So the template here is to select only field `key` in the result and change the field name
to `newKey`. `sendSingle` is another common property. Set to true means that if the result is an array, each element
will be sent individually.
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

Example to use oAuth style authentication:

OAuth header templates support only the simple placeholders <code v-pre>{{.access_token}}</code>, <code v-pre>{{.refresh_token}}</code>, <code v-pre>{{.token_type}}</code>, <code v-pre>{{.id_token}}</code>, and <code v-pre>{{.expires_in}}</code>. The names must exactly match the JSON fields returned by the token endpoint. Other placeholders, such as <code v-pre>{{.message}}</code>, <code v-pre>{{.scope}}</code>, and <code v-pre>{{.custom_token}}</code>, are evaluated only against the rule output and cannot read fields from the token response. Token endpoints must therefore return the supported field names used by the configured headers. A single header cannot mix an OAuth placeholder with a rule-output template, but different headers may use OAuth and rule-output templates separately.

```json
{
  "id": "ruleFollowBack",
  "sql": "SELECT follower FROM followStream",
  "actions": [{
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
  }]
}
```

## Visualization mode

Use visualization create rules SQL and Actions

## Text mode

Use text json create rules SQL and Actions

Example for taosdb rest：

```json
{"id": "rest1",
  "sql": "SELECT tele[0].Tag00001 AS temperature, tele[0].Tag00002 AS humidity FROM neuron",
  "actions": [
    {
      "rest": {
        "bodyType": "text",
        "dataTemplate": "insert into mqtt.kuiper values (now, {{.temperature}}, {{.humidity}})",
        "debugResp": true,
        "headers": {"Authorization": "Basic cm9vdDp0YW9zZGF0YQ=="},
        "method": "POST",
        "sendSingle": true,
        "url": "http://xxx.xxx.xxx.xxx:6041/rest/sql"
      }
    }
  ]
}
```

## Configure dynamic properties

There are many scenarios that we need to sink to dynamic url and configurations through REST sink. The properties `method`, `url`,`bodyType` and `headers` support dynamic property through jsonpath syntax. Let's look at an example to modify the previous sample to a dynamic version. Assume we receive data which have metadata like http method and url postfix. We can modify the SQL to fetch these metadata in the result. The rule result will be like:

```json
{
  "method":"post",
  "url":"http://xxx.xxx.xxx.xxx:6041/rest/sql",
  "temperature": 20,
  "humidity": 80
}
```

Then in the action, we set the `method` and `url` to be the value of the result by using data template syntax as below:

```json
{"id": "rest2",
  "sql": "SELECT tele[0]->Tag00001 AS temperature, tele[0]->Tag00002 AS humidity, method, concat(\"http://xxx.xxx.xxx.xxx:6041/rest/sql\", urlPostfix) as url FROM neuron",
  "actions": [
    {
      "rest": {
        "bodyType": "text",
        "dataTemplate": "insert into mqtt.kuiper values (now, {{.temperature}}, {{.humidity}})",
        "debugResp": true,
        "headers": {"Authorization": "Basic cm9vdDp0YW9zZGF0YQ=="},
        "method": "{{.method}}",
        "sendSingle": true,
        "url": "{{.url}}"
      }
    }
  ]
}
```

## File Upload

To upload data as files to an HTTP server, use `bodyType=formdata` configuration.

**Key Characteristics**:

- Uses "multipart/form-data" content type
- Binary results will be uploaded as form file content
- Additional form attributes can be configured via `formData`

**Best Practices**:

- For high-frequency data sources, configure batching or window aggregation to avoid frequent small file uploads.

**Example Configuration**:

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

In this example, the format `delimited` will encode the content into csv which containing 10 records each and upload.

**Custom file name**: the default file name is a millisecond timestamp; use `fileName` to customize it, with dynamic props support:

```json
{
  "rest": {
    "url": "http://yoururlhere.com/upload",
    "method": "post",
    "bodyType": "formdata",
    "fileFieldName": "file1",
    "fileName": "{{.deviceId}}-{{.seq}}.csv",
    "formData": {
      "deviceId": "{{.deviceId}}"
    }
  }
}
```

## Structured query parameters and metadata headers

**query**: appends URL query parameters as key-value pairs (without overwriting the query string already present in `url`, and handles URL encoding automatically). Supports dynamic props:

```json
{
  "rest": {
    "url": "http://yoururlhere.com/api",
    "method": "get",
    "query": {
      "device": "{{.deviceId}}",
      "timestamp": "{{.ts}}"
    }
  }
}
```

**metaHeaders**: maps source metadata (e.g. MQTT v5 `correlationData` / `responseTopic`, exposed through SQL `meta()`) onto outgoing HTTP headers. Headers are only written when the metadata entry exists, and the static `headers` configuration is never mutated:

```json
{
  "rest": {
    "url": "http://yoururlhere.com/api",
    "method": "post",
    "headers": {
      "X-API-Key": "secret"
    },
    "metaHeaders": {
      "X-Correlation-Id": "correlationData",
      "X-Response-Topic": "responseTopic"
    }
  }
}
```
