// Copyright 2021-2024 EMQ Technologies Co., Ltd.
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

//go:build !windows

package zmq

import (
	"context"
	"errors"
	"fmt"

	"github.com/lf-edge/ekuiper/contract/v2/api"
	zmq "github.com/go-zeromq/zmq4"

	"github.com/lf-edge/ekuiper/v2/pkg/infra"
	"github.com/lf-edge/ekuiper/v2/pkg/timex"
)

type zmqSource struct {
	ctx        context.Context
	cancel     context.CancelFunc
	subscriber zmq.Socket
	sc         *c
}

func (s *zmqSource) Provision(ctx api.StreamContext, configs map[string]any) error {
	sc, err := validate(ctx, configs)
	if err != nil {
		return err
	}
	s.sc = sc
	return nil
}

func (s *zmqSource) Connect(ctx api.StreamContext, sch api.StatusChangeHandler) error {
	var err error
	defer func() {
		if err != nil {
			sch(api.ConnectionDisconnected, err.Error())
		} else {
			sch(api.ConnectionConnected, "")
		}
	}()
	ctx2, cancel := context.WithCancel(context.Background())
	s.ctx = ctx2
	s.cancel = cancel
	s.subscriber = zmq.NewSub(ctx2, zmq.WithDialerMaxRetries(-1))
	if s.subscriber == nil {
		cancel()
		return fmt.Errorf("zmq source fails to create socket")
	}
	err = s.subscriber.Dial(s.sc.Server)
	if err != nil {
		cancel()
		return fmt.Errorf("zmq source fails to connect to %s: %v", s.sc.Server, err)
	}
	return nil
}

func (s *zmqSource) Subscribe(ctx api.StreamContext, ingest api.BytesIngest, ingestError api.ErrorIngest) error {
	ctx.GetLogger().Debugf("zmq source subscribe to topic %s", s.sc.Topic)
	if s.sc.Topic != "" {
		err := s.subscriber.SetOption(zmq.OptionSubscribe, s.sc.Topic)
		if err != nil {
			return err
		}
	}
	go infra.SafeRun(func() error {
		for {
			msg, e := s.subscriber.Recv()
			if e != nil {
				if errors.Is(e, context.Canceled) || s.ctx.Err() != nil {
					_ = s.subscriber.Close()
					return nil
				}
				ingestError(ctx, fmt.Errorf("zmq source getting message error: %v", e))
			} else {
				rcvTime := timex.GetNow()
				var m []byte
				for i, f := range msg.Frames {
					if i == 0 && s.sc.Topic != "" {
						continue
					}
					m = append(m, f...)
				}
				meta := make(map[string]any)
				if s.sc.Topic != "" && len(msg.Frames) > 0 {
					meta["topic"] = string(msg.Frames[0])
				}
				ingest(ctx, m, meta, rcvTime)
			}
			select {
			case <-ctx.Done():
				_ = s.subscriber.Close()
				if s.cancel != nil {
					s.cancel()
				}
				return nil
			default:
			}
		}
	})
	return nil
}

func (s *zmqSource) Close(_ api.StreamContext) error {
	return nil
}

func GetSource() api.Source {
	return &zmqSource{}
}

var _ api.BytesSource = &zmqSource{}