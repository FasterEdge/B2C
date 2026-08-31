// Copyright 2024-2025 EMQ Technologies Co., Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	stdhttp "net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/lf-edge/ekuiper/contract/v2/api"
	"github.com/pingcap/failpoint"

	"github.com/lf-edge/ekuiper/v2/internal/io/mqtt"
	"github.com/lf-edge/ekuiper/v2/internal/pkg/httpx"
	"github.com/lf-edge/ekuiper/v2/pkg/connection"
	"github.com/lf-edge/ekuiper/v2/pkg/errorx"
)

type RestSink struct {
	*ClientConf
	headerTemplates    map[string]restHeaderTemplate
	hasHeaderTemplates bool
	hasDynamicHeaders  bool
	// responseMqtt is the optional MQTT publisher used to relay the HTTP
	// response back to the caller (request-reply gateway pattern).
	responseMqtt *responseMqttPublisher
}

// responseMqttPublisher publishes the HTTP response body to an MQTT topic.
type responseMqttPublisher struct {
	cw  *connection.ConnWrapper
	cli *mqtt.Connection
	// topicTemplate is the configured topic, supporting dynamic props like
	// {{.responseTopic}}. It is resolved per message against the rule output.
	topicTemplate  string
	corrTemplate   string
	qos            byte
	retained       bool
	forwardStatus  bool
	forwardHeaders bool
	forwardErrors  bool
	forwardEmpty   bool
}

var (
	_ api.BytesCollector = &RestSink{}
)

type restHeaderTemplate struct {
	value        string
	plannerValue string
	kind         restHeaderKind
}

type restHeaderKind uint8

const (
	restHeaderStatic restHeaderKind = iota
	restHeaderOAuth
	restHeaderSQL
)

var (
	oauthTemplatePattern       = regexp.MustCompile(`{{\s*\.(access_token|refresh_token|token_type|id_token|expires_in)\s*}}`)
	oauthFieldReferencePattern = regexp.MustCompile(`{{[^{}]*\b(access_token|refresh_token|token_type|id_token|expires_in)\b[^{}]*}}`)
)

const oauthTemplateMarkerPrefix = "__ekuiper_oauth_"

var bodyTypeFormat = map[string]string{
	"json": "json",
	"form": "urlencoded",
}

func (r *RestSink) Provision(ctx api.StreamContext, configs map[string]any) error {
	r.ClientConf = &ClientConf{}
	err := r.InitConf(ctx, "", configs)
	if err != nil {
		return err
	}
	if r.ClientConf.config.Format == "" {
		r.ClientConf.config.Format = "json"
	}
	if rf, ok := bodyTypeFormat[r.ClientConf.config.BodyType]; ok && r.ClientConf.config.Format != rf {
		return fmt.Errorf("format must be %s if bodyType is %s", rf, r.ClientConf.config.BodyType)
	}
	r.headerTemplates = make(map[string]restHeaderTemplate, len(r.config.Headers))
	if r.accessConf != nil {
		r.tokenHeaderTemplates = make(map[string]string, len(r.config.Headers))
		r.requiredTokenFields = make(map[string]struct{})
	}
	for k, v := range r.config.Headers {
		if r.accessConf != nil {
			h, fields, err := newOAuthHeaderTemplate(k, v)
			if err != nil {
				return err
			}
			r.headerTemplates[k] = h
			if h.kind != restHeaderSQL {
				r.tokenHeaderTemplates[k] = v
			}
			for _, field := range fields {
				r.requiredTokenFields[field] = struct{}{}
			}
		} else {
			kind := restHeaderStatic
			if strings.Contains(v, "{{") {
				kind = restHeaderSQL
			}
			h := restHeaderTemplate{value: v, plannerValue: v, kind: kind}
			r.headerTemplates[k] = h
		}
		if h := r.headerTemplates[k]; h.kind != restHeaderStatic {
			r.hasHeaderTemplates = true
			if h.kind == restHeaderSQL {
				r.hasDynamicHeaders = true
			}
		}
	}
	return nil
}

// Consume separates OAuth response fields from templates that are evaluated
// against rule output by the common sink transform operator.
func (r *RestSink) Consume(props map[string]any) {
	deletePropFold(props, "oauth")
	if r.accessConf != nil {
		deletePropFold(props, "body")
		for key, value := range props {
			if strings.EqualFold(key, "headers") {
				switch headers := value.(type) {
				case map[string]any:
					maskedHeaders := make(map[string]any, len(headers))
					for k := range headers {
						maskedHeaders[k] = r.headerTemplates[k].plannerValue
					}
					props[key] = maskedHeaders
				case map[string]string:
					maskedHeaders := make(map[string]string, len(headers))
					for k := range headers {
						maskedHeaders[k] = r.headerTemplates[k].plannerValue
					}
					props[key] = maskedHeaders
				}
			}
		}
	}
}

func newOAuthHeaderTemplate(name, value string) (restHeaderTemplate, []string, error) {
	matches := oauthTemplatePattern.FindAllStringSubmatch(value, -1)
	withoutOAuth := oauthTemplatePattern.ReplaceAllString(value, "")
	if oauthFieldReferencePattern.MatchString(withoutOAuth) {
		return restHeaderTemplate{}, nil, fmt.Errorf("header %q uses an unsupported OAuth template; only simple placeholders such as {{.access_token}} are supported", name)
	}
	hasOAuth := len(matches) > 0
	hasSQL := strings.Contains(withoutOAuth, "{{")
	if hasOAuth && hasSQL {
		return restHeaderTemplate{}, nil, fmt.Errorf("header %q cannot mix OAuth and SQL templates", name)
	}
	h := restHeaderTemplate{value: value, plannerValue: value, kind: restHeaderStatic}
	fields := make([]string, 0, len(matches))
	if hasOAuth {
		h.kind = restHeaderOAuth
		h.plannerValue = oauthTemplateMarkerPrefix + name
		for _, match := range matches {
			fields = append(fields, match[1])
		}
	} else if hasSQL {
		h.kind = restHeaderSQL
	}
	return h, fields, nil
}

func deletePropFold(props map[string]any, name string) {
	for key := range props {
		if strings.EqualFold(key, name) {
			delete(props, key)
		}
	}
}

func (r *RestSink) Close(ctx api.StreamContext) error {
	if r.responseMqtt != nil && r.responseMqtt.cw != nil {
		_ = connection.DetachConnection(ctx, r.responseMqtt.cw.ID)
	}
	return nil
}

func (r *RestSink) Connect(ctx api.StreamContext, sch api.StatusChangeHandler) error {
	err := r.Conn(ctx)
	if err != nil {
		return err
	}
	// If response relay is configured, establish the reply channel.
	if r.config.Response != nil && r.config.Response.Mqtt != nil {
		rc := r.config.Response.Mqtt
		mqttProps := map[string]any{
			"server": rc.Server,
		}
		// Only set non-empty fields so that the MQTT connection keeps its
		// defaults (e.g. protocolVersion defaults to 3.1.1).
		if rc.ProtocolVersion != "" {
			mqttProps["protocolVersion"] = rc.ProtocolVersion
		}
		if rc.ClientId != "" {
			mqttProps["clientid"] = rc.ClientId
		}
		if rc.Username != "" {
			mqttProps["username"] = rc.Username
		}
		if rc.Password != "" {
			mqttProps["password"] = rc.Password
		}
		refId := fmt.Sprintf("%s-%s-rest-response", ctx.GetRuleId(), ctx.GetOpId())
		cw, err := connection.FetchConnection(ctx, refId, "mqtt", mqttProps, sch)
		if err != nil {
			return fmt.Errorf("rest sink response mqtt: %v", err)
		}
		conn, e := cw.Wait(ctx)
		if conn == nil {
			return fmt.Errorf("rest sink response mqtt not ready: %v", e)
		}
		cli, ok := conn.(*mqtt.Connection)
		if !ok {
			return fmt.Errorf("rest sink response: connection is not mqtt")
		}
		r.responseMqtt = &responseMqttPublisher{
			cw:             cw,
			cli:            cli,
			topicTemplate:  rc.Topic,
			corrTemplate:   rc.CorrelationData,
			qos:            rc.Qos,
			retained:       rc.Retained,
			forwardStatus:  rc.ForwardStatus,
			forwardHeaders: rc.ForwardHeaders,
			forwardErrors:  rc.ForwardErrors,
			forwardEmpty:   rc.ForwardEmpty,
		}
		ctx.GetLogger().Debugf("rest sink response mqtt connected to %s", rc.Server)
	}
	sch(api.ConnectionConnected, "")
	return nil
}

func (r *RestSink) Collect(ctx api.StreamContext, item api.RawTuple) error {
	logger := ctx.GetLogger()
	bodyType := r.config.BodyType
	method := r.config.Method
	u := r.config.Url
	headers := r.prepareHeaders(item)
	formData := r.config.FormData
	query := r.config.Query
	fileName := r.config.FileName

	dp, hasDynamicProps := item.(api.HasDynamicProps)
	dynamicURL := false
	dynamicBodyType := false
	if hasDynamicProps {
		nb, ok := dp.DynamicProps(bodyType)
		if ok {
			bodyType = nb
			dynamicBodyType = true
		}
		nm, ok := dp.DynamicProps(method)
		if ok {
			method = nm
		}
		nu, ok := dp.DynamicProps(u)
		if ok {
			u = nu
			dynamicURL = true
		}
		if bodyType == "formdata" {
			// Resolve form fields per tuple. Keeping this local avoids a data race
			// and prevents one tuple from changing another tuple's templates.
			formData = make(map[string]string, len(r.config.FormData))
			for k, v := range r.config.FormData {
				if nv, ok := dp.DynamicProps(v); ok {
					formData[k] = nv
				} else {
					formData[k] = v
				}
			}
			if fileName != "" {
				if nf, ok := dp.DynamicProps(fileName); ok {
					fileName = nf
				}
			}
		}
		if len(query) > 0 {
			resolved := make(map[string]string, len(query))
			for k, v := range query {
				if nv, ok := dp.DynamicProps(v); ok {
					resolved[k] = nv
				} else {
					resolved[k] = v
				}
			}
			query = resolved
		}
	}

	normalizedMethod, validationErr := normalizeHTTPMethod(method)
	if validationErr != nil {
		return fmt.Errorf("invalid dynamic method: %w", validationErr)
	}
	method = normalizedMethod
	bodyType = strings.ToLower(strings.TrimSpace(bodyType))
	if _, ok := bodyTypeMap[bodyType]; !ok {
		return fmt.Errorf("invalid dynamic body type %s", bodyType)
	}
	if dynamicBodyType {
		if required, ok := bodyTypeFormat[bodyType]; ok && strings.ToLower(r.config.Format) != required {
			return fmt.Errorf("format must be %s if bodyType is %s", required, bodyType)
		}
	}
	if dynamicURL {
		if err := httpx.IsHttpUrl(u); err != nil {
			return fmt.Errorf("invalid dynamic url %s: %w", u, err)
		}
	}
	if len(query) > 0 {
		merged, err := mergeQueryParams(u, query)
		if err != nil {
			return fmt.Errorf("invalid query params: %w", err)
		}
		u = merged
	}

	// Map source metadata (e.g. MQTT v5 responseTopic / correlationData,
	// exposed through meta() in SQL) onto outgoing HTTP headers. The headers
	// map is cloned on the first hit so the configured shared map is never
	// mutated.
	headers = r.applyMetaHeaders(item, headers)

	switch r.config.Compression {
	case "zstd":
		if headers == nil {
			headers = make(map[string]string)
		}
		headers["Content-Encoding"] = "zstd"
	case "gzip":
		if headers == nil {
			headers = make(map[string]string)
		}
		headers["Content-Encoding"] = "gzip"
	}

	resp, err := r.Send(ctx, bodyType, method, u, headers, formData, r.config.FileFieldName, fileName, item.Raw())
	failpoint.Inject("recoverAbleErr", func() {
		err = errors.New("connection reset by peer")
	})
	defer func() {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
	}()
	if err != nil {
		originErr := err
		recoverAble := errorx.IsRecoverAbleError(originErr)
		if recoverAble {
			logger.Errorf("rest sink meet error:%v, recoverAble:%v, ruleID:%v", originErr.Error(), recoverAble, ctx.GetRuleId())
			return errorx.NewIOErr(fmt.Sprintf(`rest sink fails to send out the data:err=%s recoverAble=%v method=%s path="%s"`,
				originErr.Error(),
				recoverAble,
				method,
				u))
		}
		return fmt.Errorf(`rest sink fails to send out the data:err=%s recoverAble=%v method=%s path="%s"`,
			originErr.Error(),
			recoverAble,
			method, u)
	} else {
		logger.Debugf("rest sink got response %v", resp)
		// When a response relay is configured, capture the raw response body
		// first so it can be forwarded back to the caller, then restore the
		// body for the normal parsing flow.
		var relayBody []byte
		if r.responseMqtt != nil && resp != nil && resp.Body != nil {
			raw, readErr := io.ReadAll(resp.Body)
			if readErr != nil {
				logger.Warnf("rest sink failed to read response body for relay: %v", readErr)
			} else {
				relayBody = raw
				resp.Body = io.NopCloser(bytes.NewReader(raw))
			}
		}
		_, b, err := r.parseResponse(ctx, resp, "", r.config.DebugResp || r.responseMqtt != nil, true)
		// A response relay is opt-in for errors. Preserve legacy sink behavior
		// unless forwardErrors is explicitly enabled.
		if err != nil && resp.StatusCode >= 300 && r.responseMqtt != nil && r.responseMqtt.forwardErrors {
			if len(relayBody) > 0 || r.responseMqtt.forwardEmpty {
				if publishErr := r.responseMqtt.Publish(ctx, item, relayBody, resp); publishErr != nil {
					return errorx.NewIOErr(fmt.Sprintf("rest sink response relay failed: %v", publishErr))
				}
				return nil
			}
			// Do not silently swallow an error when there is no body to relay.
			return fmt.Errorf(`rest sink response error: status=%d. | method=%s path="%s"`, resp.StatusCode, method, u)
		}
		// do not record response body error as it is not an error in the sink action.
		if err != nil {
			if strings.HasPrefix(err.Error(), BODY_ERR) {
				logger.Warnf("rest sink response body error: %v", err)
			} else {
				return fmt.Errorf(`parse response error: %s. | method=%s path="%s" status=%d response_body="%s"`,
					err,
					method,
					u,
					resp.StatusCode,
					b,
				)
			}
		}
		if r.config.DebugResp {
			logger.Infof("Response raw content: %s\n", b)
		}
		// Relay the HTTP response back to the caller through the configured
		// reply channel (request-reply gateway pattern).
		if r.responseMqtt != nil && (len(relayBody) > 0 || r.responseMqtt.forwardEmpty) {
			if err := r.responseMqtt.Publish(ctx, item, relayBody, resp); err != nil {
				logger.Errorf("rest sink response relay failed: %v", err)
				return errorx.NewIOErr(fmt.Sprintf("rest sink response relay failed: %v", err))
			}
		}
	}
	return nil
}

// Publish relays the HTTP response body back to the caller via MQTT. The topic
// and correlation data are resolved from the rule output per message, so a
// request's MQTT v5 response topic / correlation data can be echoed back.
func (p *responseMqttPublisher) Publish(ctx api.StreamContext, item api.RawTuple, body []byte, resp *stdhttp.Response) error {
	topic := p.topicTemplate
	props := make(map[string]string, 2)
	if dp, ok := item.(api.HasDynamicProps); ok {
		if nt, transformed := dp.DynamicProps(p.topicTemplate); transformed {
			topic = nt
		}
		if p.corrTemplate != "" {
			if nc, transformed := dp.DynamicProps(p.corrTemplate); transformed {
				props["correlationData"] = nc
			}
		}
	}
	if topic == "" {
		return fmt.Errorf("response topic is empty")
	}
	if p.forwardStatus || p.forwardHeaders {
		var err error
		body, err = p.buildEnvelope(body, resp)
		if err != nil {
			return fmt.Errorf("encode response relay: %w", err)
		}
	}
	ctx.GetLogger().Debugf("relaying rest response to mqtt topic %s, body %s", topic, string(body))
	return p.cli.Publish(ctx, topic, p.qos, p.retained, body, props)
}

func (p *responseMqttPublisher) buildEnvelope(body []byte, resp *stdhttp.Response) ([]byte, error) {
	var envelopeBody any = string(body)
	if len(body) > 0 && json.Valid(body) {
		envelopeBody = json.RawMessage(body)
	}
	envelope := map[string]any{"body": envelopeBody}
	if p.forwardStatus && resp != nil {
		envelope["status"] = resp.StatusCode
	}
	if p.forwardHeaders && resp != nil {
		headers := make(map[string][]string, len(resp.Header))
		for key, values := range resp.Header {
			headers[key] = append([]string(nil), values...)
		}
		envelope["headers"] = headers
	}
	return json.Marshal(envelope)
}

func (r *RestSink) prepareHeaders(item api.RawTuple) map[string]string {
	if !r.hasHeaderTemplates {
		if r.config.Compression != "" {
			return cloneHeaders(r.config.Headers)
		}
		return r.config.Headers
	}
	oauthState := r.oauthRuntimeState()
	if r.accessConf != nil && !r.hasDynamicHeaders && oauthState != nil && r.config.Compression == "" {
		return oauthState.headers
	}
	headers := make(map[string]string, len(r.headerTemplates))
	dp, hasDynamicProps := item.(api.HasDynamicProps)
	for k, headerTemplate := range r.headerTemplates {
		if headerTemplate.kind != restHeaderSQL && oauthState != nil {
			if resolved, ok := oauthState.headers[k]; ok {
				headers[k] = resolved
				continue
			}
		}
		value := headerTemplate.value
		if headerTemplate.kind == restHeaderSQL && hasDynamicProps {
			if dynamicValue, ok := dp.DynamicProps(headerTemplate.value); ok {
				value = dynamicValue
			}
		}
		headers[k] = value
	}
	return headers
}

func cloneHeaders(headers map[string]string) map[string]string {
	result := make(map[string]string, len(headers))
	for k, v := range headers {
		result[k] = v
	}
	return result
}

// mergeQueryParams appends key/value pairs to the query string of an
// existing URL, preserving params already present in the URL.
func mergeQueryParams(rawURL string, params map[string]string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	q := parsed.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	parsed.RawQuery = q.Encode()
	return parsed.String(), nil
}

// metaProvider is the minimal duck-typed interface for reading source
// metadata. api.MetaInfo is not used directly because some tuples (e.g.
// xsql.RawTuple) expose Meta without the full MetaInfo contract.
type metaProvider interface {
	Meta(key, table string) (any, bool)
}

// applyMetaHeaders maps source metadata keys (e.g. MQTT v5
// responseTopic/correlationData, exposed via meta() in SQL) onto outgoing
// HTTP headers. It returns a clone on the first hit so the configured shared
// headers map is never mutated, preserving the static-headers fast path.
func (r *RestSink) applyMetaHeaders(item api.RawTuple, headers map[string]string) map[string]string {
	if len(r.config.MetaHeaders) == 0 {
		return headers
	}
	meta, ok := item.(metaProvider)
	if !ok {
		return headers
	}
	var merged map[string]string
	cloned := false
	for headerName, metaKey := range r.config.MetaHeaders {
		v, ok := meta.Meta(metaKey, "")
		if !ok {
			continue
		}
		if !cloned {
			merged = cloneHeaders(headers)
			cloned = true
		}
		merged[headerName] = fmt.Sprintf("%v", v)
	}
	if !cloned {
		return headers
	}
	return merged
}

func GetSink() api.Sink {
	return &RestSink{}
}

var _ api.BytesCollector = &RestSink{}
