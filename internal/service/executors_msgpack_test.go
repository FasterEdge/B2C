// Copyright 2023 EMQ Technologies Co., Ltd.
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

//go:build msgpack

package service

import (
	"io"
	"net"
	"reflect"
	"sync"
	"testing"

	"github.com/msgpack-rpc/msgpack-rpc-go/rpc"

	"github.com/stretchr/testify/require"

	"github.com/lf-edge/ekuiper/v2/internal/pkg/def"
	"github.com/lf-edge/ekuiper/v2/internal/testx"
	"github.com/lf-edge/ekuiper/v2/internal/topo/topotest"
)

// fullReadConn 把底层连接的单次 Read/Write 补成"读满/写满"语义。
// 用例用作 mock 服务端的 msgpack-rpc-go 与 msgpack-go 都停在 2013 年，
// 内部用裸 reader.Read(data) 填充定长缓冲区：TCP 一旦把一条消息切成两段，
// 第二次 Read 只拿到部分字节，缓冲区尾部就留着零值（例如 "world" 只到 "wo" ，尾部留零字节）；
// 消息头被切碎时更会让函数名字段错位，Resolver 查不到名字返回零值，
// 于是 rpc 库在 f.Type() 上 panic。这里用 io.ReadFull/完整写把它挡住。
type fullReadConn struct {
	net.Conn
	wmu sync.Mutex
}

func (c *fullReadConn) Read(p []byte) (int, error) {
	return io.ReadFull(c.Conn, p)
}

func (c *fullReadConn) Write(p []byte) (int, error) {
	c.wmu.Lock()
	defer c.wmu.Unlock()
	return c.Conn.Write(p) // net.Conn.Write 在整个缓冲写完前不会返回短写
}

type fullReadListener struct {
	net.Listener
}

func (l fullReadListener) Accept() (net.Conn, error) {
	c, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	return &fullReadConn{Conn: c}, nil
}

func TestMsgpackService(t *testing.T) {
	if testx.Race {
		t.Skip("skip msgpack service test in race mode")
	}
	// mock server
	res := Resolver{"SayHello": reflect.ValueOf(SayHello), "object_detection": reflect.ValueOf(object_detection), "get_feature": reflect.ValueOf(get_feature), "getStatus": reflect.ValueOf(getStatus)}
	serv := rpc.NewServer(res, true, nil)
	l, err := net.Listen("tcp", ":50000")
	require.NoError(t, err)
	serv.Listen(fullReadListener{Listener: l})
	go serv.Run()
	// Comment out because the bug in the msgpack rpc
	// defer serv.Stop()

	// Reset
	streamList := []string{"helloStr", "commands", "fakeBin"}
	topotest.HandleStream(false, streamList, t)
	// Data setup
	tests := []topotest.RuleTest{
		{
			Name: `TestRestRule1`,
			Sql:  `SELECT helloFromMsgpack(name) as wc FROM helloStr`,
			R: [][]map[string]interface{}{
				{{
					"wc": map[string]interface{}{
						"message": "world",
					},
				}},
				{{
					"wc": map[string]interface{}{
						"message": "golang",
					},
				}},
				{{
					"wc": map[string]interface{}{
						"message": "peacock",
					},
				}},
			},
			M: map[string]interface{}{
				"op_2_project_0_exceptions_total":   int64(0),
				"op_2_project_0_process_latency_us": int64(0),
				"op_2_project_0_records_in_total":   int64(3),
				"op_2_project_0_records_out_total":  int64(3),

				"sink_memory_0_0_exceptions_total":  int64(0),
				"sink_memory_0_0_records_in_total":  int64(3),
				"sink_memory_0_0_records_out_total": int64(3),
			},
		}, {
			Name: `TestRestRule2`,
			Sql:  `SELECT objectDetectFromMsgpack(*)->result FROM commands`,
			R: [][]map[string]interface{}{
				{{
					"kuiper_field_0": "get success",
				}},
				{{
					"kuiper_field_0": "detect success",
				}},
				{{
					"kuiper_field_0": "delete success",
				}},
			},
			M: map[string]interface{}{
				"op_2_project_0_exceptions_total":   int64(0),
				"op_2_project_0_process_latency_us": int64(0),
				"op_2_project_0_records_in_total":   int64(3),
				"op_2_project_0_records_out_total":  int64(3),

				"sink_memory_0_0_exceptions_total":  int64(0),
				"sink_memory_0_0_records_in_total":  int64(3),
				"sink_memory_0_0_records_out_total": int64(3),
			},
		}, {
			Name: `TestRestRule3`,
			Sql:  `SELECT getFeatureFromMsgpack(self)->feature[0]->box->h FROM fakeBin`,
			R: [][]map[string]interface{}{
				{{
					"kuiper_field_0": int64(106), // Convert by the testing tool
				}},
				{{
					"kuiper_field_0": int64(107),
				}},
				{{
					"kuiper_field_0": int64(108),
				}},
			},
			M: map[string]interface{}{
				"op_2_project_0_exceptions_total":   int64(0),
				"op_2_project_0_process_latency_us": int64(0),
				"op_2_project_0_records_in_total":   int64(3),
				"op_2_project_0_records_out_total":  int64(3),

				"sink_memory_0_0_exceptions_total":  int64(0),
				"sink_memory_0_0_records_in_total":  int64(3),
				"sink_memory_0_0_records_out_total": int64(3),
			},
			//}, {
			//	Name: `TestRestRule4`,
			//	Sql:  `SELECT getStatusFromMsgpack(), command FROM commands`,
			//	R: [][]map[string]interface{}{
			//		{{
			//			"getStatusFromRest": true,
			//			"command": "get",
			//		}},
			//		{{
			//			"getStatusFromRest": true,
			//			"command": "detect",
			//		}},
			//		{{
			//			"getStatusFromRest": true,
			//			"command": "delete",
			//		}},
			//	},
			//	M: map[string]interface{}{
			//		"op_2_project_0_exceptions_total":   int64(0),
			//		"op_2_project_0_process_latency_us": int64(0),
			//		"op_2_project_0_records_in_total":   int64(3),
			//		"op_2_project_0_records_out_total":  int64(3),
			//
			//		"sink_memory_0_0_exceptions_total":  int64(0),
			//		"sink_memory_0_0_records_in_total":  int64(3),
			//		"sink_memory_0_0_records_out_total": int64(3),
			//	},
		},
	}
	topotest.HandleStream(true, streamList, t)
	topotest.DoRuleTest(t, tests, &def.RuleOption{
		BufferLength: 100,
		SendError:    true,
	}, 0)
}
