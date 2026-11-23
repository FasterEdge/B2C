// Copyright 2021-2025 EMQ Technologies Co., Ltd.
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

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/edgexfoundry/go-mod-messaging/v4/messaging"
	"github.com/edgexfoundry/go-mod-messaging/v4/pkg/types"

	"github.com/lf-edge/ekuiper/v2/internal/conf"
)

func subEventsFromMQTT(host string) {
	msgConfig1 := types.MessageBusConfig{
		Broker: types.HostInfo{
			Host:     host,
			Port:     1883,
			Protocol: "tcp",
		},
		Type: messaging.MQTT,
	}

	if msgClient, err := messaging.NewMessageClient(msgConfig1); err != nil {
		conf.Log.Fatal(err)
	} else {
		if ec := msgClient.Connect(); ec != nil {
			conf.Log.Fatal(ec)
		} else {
			// log.Infof("The connection to edgex messagebus is established successfully.")
			messages := make(chan types.MessageEnvelope)
			topics := []types.TopicChannel{{Topic: "result", Messages: messages}}
			err := make(chan error)
			if e := msgClient.Subscribe(topics, err); e != nil {
				conf.Log.Fatal(e)
			} else {
				count := 0
				for {
					select {
					case e1 := <-err:
						conf.Log.Errorf("%s\n", e1)
						return
					case env := <-messages:
						count++
						printPayload(env.Payload)
						if count == 1 {
							return
						}
					}
				}
			}
		}
	}
}

// printPayload 只把消息体作为一段完整 JSON 输出。
// JMeter 的 OS Process Sampler 会合并子进程的 stdout 与 stderr 作为响应体再做
// JSONPath 断言, 所以除了这里, 任何其他输出都会破坏 JSON 的可解析性。
func printPayload(payload any) {
	b, err := json.Marshal(payload)
	if err != nil {
		conf.Log.Errorf("marshal payload error: %v", err)
		return
	}
	fmt.Printf("%s\n", b)
}

func main() {
	// go-mod-messaging 的 MQTT 客户端用标准库 log 打连接日志, 默认走 stderr,
	// 被合并进 JMeter 响应体后会让 JSONPath 断言只看到 java.lang.String。
	log.SetOutput(io.Discard)
	if len(os.Args) == 3 {
		if v := os.Args[1]; v == "mqtt" {
			subEventsFromMQTT(os.Args[2])
		}
		if v := os.Args[1]; v == "redis" {
			panic("edgex v4 does not support redis")
		}
	}
}
