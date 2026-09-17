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

package tdengine3

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	"github.com/lf-edge/ekuiper/contract/v2/api"
	_ "github.com/taosdata/driver-go/v3/taosWS"

	"github.com/lf-edge/ekuiper/v2/pkg/cast"
)

type TaosConfig struct {
	ProvideTs    bool     `json:"provideTs"`
	Port         int      `json:"port"`
	Host         string   `json:"host"`
	User         string   `json:"user"`
	Password     string   `json:"password"`
	Database     string   `json:"database"`
	Table        string   `json:"table"`
	TsFieldName  string   `json:"tsFieldName"`
	Fields       []string `json:"fields"`
	STable       string   `json:"sTable"`
	TagFields    []string `json:"tagFields"`
	DataTemplate string   `json:"dataTemplate"`
	DataField    string   `json:"dataField"`
}

type tdengineSink3 struct {
	cfg *TaosConfig
	cli *sql.DB
}

func (t *tdengineSink3) Ping(ctx api.StreamContext, props map[string]any) error {
	url := fmt.Sprintf(`%s:%s@tcp(%s)/%s`, t.cfg.User, t.cfg.Password, cast.JoinHostPortInt(t.cfg.Host, t.cfg.Port), t.cfg.Database)
	taos, err := sql.Open("taosSql", url)
	if err != nil {
		return fmt.Errorf("Failed to connect to tdengine3: %s", err)
	}

	defer taos.Close()
	return nil
}

func (t *tdengineSink3) Provision(ctx api.StreamContext, props map[string]any) error {
	t.cfg = &TaosConfig{
		Host:     "localhost",
		Port:     6041,
		User:     "root",
		Password: "taosdata",
	}
	err := cast.MapToStruct(props, t.cfg)
	if err != nil {
		return err
	}
	if t.cfg.Database == "" {
		return fmt.Errorf("property database is required")
	}
	if t.cfg.Table == "" {
		return fmt.Errorf("property table is required")
	}
	if t.cfg.TsFieldName == "" {
		return fmt.Errorf("property TsFieldName is required")
	}
	if t.cfg.STable != "" && len(t.cfg.TagFields) == 0 {
		return fmt.Errorf("property tagFields is required when sTable is set")
	}
	return nil
}

func (t *tdengineSink3) Connect(ctx api.StreamContext, sch api.StatusChangeHandler) error {
	ctx.GetLogger().Infof("tdengine3 sink connection")
	url := fmt.Sprintf(`%s:%s@ws(%s)/%s`, t.cfg.User, t.cfg.Password, cast.JoinHostPortInt(t.cfg.Host, t.cfg.Port), t.cfg.Database)
	taosCli, err := sql.Open("taosWS", url)
	t.cli = taosCli
	return err
}

func (t *tdengineSink3) Close(ctx api.StreamContext) error {
	ctx.GetLogger().Infof("tdengine3 sink close")
	t.cli.Close()
	return nil
}

func (t *tdengineSink3) Collect(ctx api.StreamContext, item api.MessageTuple) error {
	sqlStr, sqlE := t.cfg.buildSql(item)
	if sqlE != nil {
		return fmt.Errorf("failed to build sql to tdengine3: %s", sqlE)
	}
	ctx.GetLogger().Debugf("tdengine3 sink collect sql: %s", sqlStr)
	_, e := t.cli.Exec(sqlStr)
	if e != nil {
		return fmt.Errorf("failed to exec sql to tdengine3: %s", e)
	}
	return nil
}

func (t *tdengineSink3) CollectList(ctx api.StreamContext, items api.MessageTupleList) error {
	items.RangeOfTuples(func(_ int, tuple api.MessageTuple) bool {
		err := t.Collect(ctx, tuple)
		if err != nil {
			ctx.GetLogger().Error(err)
		}
		return true
	})
	return nil
}

func (cfg *TaosConfig) buildSql(item api.MessageTuple) (string, error) {
	mapData := item.ToMap()
	var keys, vals, tags []string
	if len(mapData) == 0 {
		return "", fmt.Errorf("data is empty")
	}
	table := cfg.Table
	if dp, ok := item.(api.HasDynamicProps); ok {
		temp, transformed := dp.DynamicProps(table)
		if transformed {
			table = temp
		}
	}

	sTable := cfg.STable
	if dp, ok := item.(api.HasDynamicProps); ok {
		temp, transformed := dp.DynamicProps(sTable)
		if transformed {
			sTable = temp
		}
	}

	if cfg.ProvideTs {
		if v, ok := mapData[cfg.TsFieldName]; !ok {
			return "", fmt.Errorf("timestamp field not found : %s", cfg.TsFieldName)
		} else {
			keys = append(keys, cfg.TsFieldName)
			vals = append(vals, fmt.Sprintf(`%v`, v))
		}
	} else {
		vals = append(vals, "now")
		keys = append(keys, cfg.TsFieldName)
	}

	if len(cfg.TagFields) > 0 {
		for _, v := range cfg.TagFields {
			switch mapData[v].(type) {
			case string:
				tags = append(tags, quoteTdengineString(fmt.Sprint(mapData[v])))
			default:
				tags = append(tags, fmt.Sprintf(`%v`, mapData[v]))
			}
		}
	}

	if len(cfg.Fields) != 0 {
		for _, k := range cfg.Fields {
			if k == cfg.TsFieldName {
				continue
			}
			if contains(cfg.TagFields, k) {
				continue
			}
			if v, ok := mapData[k]; ok {
				keys = append(keys, k)
				if reflect.String == reflect.TypeOf(v).Kind() {
					vals = append(vals, quoteTdengineString(fmt.Sprint(v)))
				} else {
					vals = append(vals, fmt.Sprintf(`%v`, v))
				}
			} else {
				return "", fmt.Errorf("field not found : %s", k)
			}
		}
	} else {
		for k, v := range mapData {
			if k == cfg.TsFieldName {
				continue
			}
			if contains(cfg.TagFields, k) {
				continue
			}
			if !isSafeDynamicFieldName(k) {
				return "", fmt.Errorf("invalid dynamic field name %q: expected [A-Za-z_][A-Za-z0-9_]*", k)
			}
			keys = append(keys, k)
			if reflect.String == reflect.TypeOf(v).Kind() {
				vals = append(vals, quoteTdengineString(fmt.Sprint(v)))
			} else {
				vals = append(vals, fmt.Sprintf(`%v`, v))
			}
		}
	}

	sqlStr := fmt.Sprintf("INSERT INTO %s (%s)", table, strings.Join(keys, ","))
	if sTable != "" {
		sqlStr += " USING " + sTable
	}
	if len(tags) != 0 {
		sqlStr += " TAGS(" + strings.Join(tags, ",") + ")"
	}
	sqlStr += " values (" + strings.Join(vals, ",") + ")"
	return sqlStr, nil
}

// quoteTdengineString 转义 TDengine 双引号字符串字面量: 反斜杠与双引号需转义,
// 否则业务数据值(可来自不可信的上游消息)含 `"` / `\` 会破坏字面量边界并注入 SQL。
func quoteTdengineString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return `"` + s + `"`
}

// isSafeDynamicFieldName 校验动态消息键(未在 cfg.Fields 显式配置时取自 mapData 键):
// 只允许 [A-Za-z_][A-Za-z0-9_]*, 防止键名注入 SQL 语法(与 extensions/impl/sql 对齐)。
func isSafeDynamicFieldName(identifier string) bool {
	if len(identifier) == 0 {
		return false
	}
	for i := 0; i < len(identifier); i++ {
		c := identifier[i]
		if c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' {
			continue
		}
		if i > 0 && c >= '0' && c <= '9' {
			continue
		}
		return false
	}
	return true
}

func contains(slice []string, target string) bool {
	for _, element := range slice {
		if element == target {
			return true
		}
	}
	return false
}

func GetSink() api.Sink {
	return &tdengineSink3{}
}
