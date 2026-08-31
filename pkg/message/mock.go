// Copyright 2024 EMQ Technologies Co., Ltd.
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

package message

import (
	"github.com/lf-edge/ekuiper/contract/v2/api"

	"github.com/lf-edge/ekuiper/v2/pkg/ast"
)

type MockPartialConverter struct {
	i int
}

func (m *MockPartialConverter) Encode(_ api.StreamContext, _ any) ([]byte, error) {
	panic("unexpected Encode call in MockPartialConverter")
}

func (m *MockPartialConverter) Decode(_ api.StreamContext, _ []byte) (any, error) {
	panic("unexpected Decode call in MockPartialConverter")
}

func (m *MockPartialConverter) DecodeField(ctx api.StreamContext, b []byte, f string) (any, error) {
	r := m.i % 2
	m.i++
	return r, nil
}

type MockMerger struct {
	count  int
	frames []map[string]any
}

func (m *MockMerger) Split(ctx api.StreamContext, b []byte) [][]byte {
	return [][]byte{b}
}

func (m *MockMerger) Merging(ctx api.StreamContext, b []byte) error {
	if m.frames == nil {
		m.frames = make([]map[string]any, 2)
	}
	m.frames[m.count%2] = map[string]any{"data": b}
	m.count++
	return nil
}

func (m *MockMerger) Trigger(ctx api.StreamContext) ([]any, bool) {
	result := make([]any, len(m.frames))
	for i, frame := range m.frames {
		result[i] = frame
	}
	return result, true
}

func (m *MockMerger) ResetSchema(schema map[string]*ast.JsonStreamField) {}
