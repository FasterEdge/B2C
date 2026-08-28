// Copyright 2026 FasterEdge
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
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lf-edge/ekuiper/contract/v2/api"
	"github.com/lf-edge/ekuiper/v2/internal/conf"
	"github.com/lf-edge/ekuiper/v2/internal/io/mqtt"
	"github.com/lf-edge/ekuiper/v2/internal/pkg/store"
	"github.com/lf-edge/ekuiper/v2/internal/xsql"
	"github.com/lf-edge/ekuiper/v2/pkg/connection"
	"github.com/lf-edge/ekuiper/v2/pkg/modules"
	mockContext "github.com/lf-edge/ekuiper/v2/pkg/mock/context"
	mqttserver "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/hooks/auth"
	"github.com/mochi-mqtt/server/v2/listeners"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	modules.RegisterConnection("mqtt", mqtt.CreateConnection)
}

// TestRestSinkResponseRelay verifies the MQTT -> HTTP -> MQTT request-reply
// gateway loop: a rule output carries an MQTT v5 response topic and correlation
// data (resolved as dynamic props), the REST sink forwards the request as HTTP,
// and the HTTP response is published back to the MQTT response topic with the
// correlation data echoed.
func TestRestSinkResponseRelay(t *testing.T) {
	// ---- 1. Start an in-process MQTT v5 broker ----
	server := mqttserver.New(nil)
	_ = server.AddHook(new(auth.AllowHook), nil)
	tcp := listeners.NewTCP(listeners.Config{ID: "relay", Address: ":13883"})
	require.NoError(t, server.AddListener(tcp))
	go func() {
		_ = server.Serve()
	}()
	defer func() {
		_ = server.Close()
		tcp.Close(nil)
	}()
	brokerURL := "mqtt://127.0.0.1:13883"

	// ---- 2. Start a stub HTTP target that echoes a JSON body ----
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","echoed":true}`))
	}))
	defer httpServer.Close()

	dataDir, err := conf.GetDataLoc()
	require.NoError(t, err)
	require.NoError(t, store.SetupDefault(dataDir))
	require.NoError(t, connection.InitConnectionManager4Test())

	ctx := mockContext.NewMockContext("relayRule", "op1")
	s := &RestSink{}

	// Subscribe to the reply topic first so that the relayed (QoS 0) message
	// is not missed.
	replyCh := make(chan []byte, 1)
	corrCh := make(chan string, 1)
	sub := NewTestSubscriber(t, brokerURL, "reply/dev/1", replyCh, corrCh)
	defer sub.Close()
	time.Sleep(500 * time.Millisecond)

	require.NoError(t, s.Provision(ctx, map[string]any{
		"url":      httpServer.URL,
		"method":   "post",
		"bodyType": "json",
		"response": map[string]any{
			"type": "mqtt",
			"mqtt": map[string]any{
				"server":          brokerURL,
				"protocolVersion": "5",
				"topic":           "{{.responseTopic}}",
				"correlationData": "{{.correlationData}}",
				"qos":             0,
			},
		},
	}))
	require.NoError(t, s.Connect(ctx, func(string, string) {}))

	// ---- 3. Publish a request with MQTT v5 response topic / correlation data
	// as dynamic props (as a rule output would after SELECT meta(...)).
	raw := &xsql.RawTuple{
		Rawdata: []byte(`{"command":"get","id":1}`),
		Props: map[string]string{
			"{{.responseTopic}}":   "reply/dev/1",
			"{{.correlationData}}": "corr-abc-123",
		},
	}
	require.NoError(t, s.Collect(ctx, raw))

	// ---- 4. Expect the HTTP response body back on MQTT with the correlation
	// data echoed as a v5 property.
	select {
	case payload := <-replyCh:
		assert.JSONEq(t, `{"status":"ok","echoed":true}`, string(payload))
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for the relayed HTTP response on MQTT")
	}
	select {
	case corr := <-corrCh:
		assert.Equal(t, "corr-abc-123", corr)
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for the correlation data property")
	}
}

// NewTestSubscriber is a minimal MQTT v5 subscriber used by the relay test to
// capture the response payload and the correlation data property.
type TestSubscriber struct {
	cli api.BytesSource
	ctx api.StreamContext
}

func NewTestSubscriber(t *testing.T, brokerURL, topic string, payloadCh chan []byte, corrCh chan string) *TestSubscriber {
	ctx := mockContext.NewMockContext("sub", "op1")
	sub, ok := mqtt.GetSource().(api.BytesSource)
	require.True(t, ok, "mqtt source should be a bytes source")
	require.NoError(t, sub.Provision(ctx, map[string]any{
		"server":          brokerURL,
		"protocolVersion": "5",
		"datasource":      topic,
		"qos":             0,
	}))
	require.NoError(t, sub.Connect(ctx, func(string, string) {}))
	require.NoError(t, sub.Subscribe(ctx, func(_ api.StreamContext, data []byte, meta map[string]any, _ time.Time) {
		payloadCh <- data
		if cd, ok := meta["correlationData"]; ok {
			if s, ok := cd.(string); ok {
				corrCh <- s
			}
		}
	}, func(_ api.StreamContext, err error) {
		t.Logf("subscriber ingest error: %v", err)
	}))
	return &TestSubscriber{cli: sub, ctx: ctx}
}

func (ts *TestSubscriber) Close() {
	if ts.cli != nil {
		_ = ts.cli.Close(ts.ctx)
	}
}
