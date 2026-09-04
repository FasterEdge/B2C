package native

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lf-edge/ekuiper/v2/internal/plugin"
)

func TestManager_Register_Security_PathTraversal(t *testing.T) {
	s := httptest.NewServer(
		http.FileServer(http.Dir("../testzips")),
	)
	defer s.Close()
	endpoint := s.URL

	tests := []struct {
		name    string
		pName   string
		u       string
		wantErr bool
	}{
		{
			name:    "path traversal attempt",
			pName:   "../../evil",
			u:       endpoint + "/sources/random2.zip",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &plugin.IOPlugin{
				Name: tt.pName,
				File: tt.u,
			}
			err := manager.Register(plugin.SOURCE, p)
			if tt.wantErr {
				if assert.Error(t, err) {
					// Check for the specific validation error
					assert.Contains(t, err.Error(), "path escapes from parent")
				}
			} else {
				assert.NoError(t, err)
				// Cleanup
				manager.Delete(plugin.SOURCE, tt.pName, false)
			}
		})
	}
}

// TestManager_Register_FailedSymbolsNotLeaked 覆盖 storeSymbols 部分写入缺陷:
// 失败注册 (符号冲突) 必须零残留 — 冲突符号既不能出现在 ListSymbols,
// 也不能覆盖已有符号的映射。
func TestManager_Register_FailedSymbolsNotLeaked(t *testing.T) {
	s := httptest.NewServer(
		http.FileServer(http.Dir("../testzips")),
	)
	defer s.Close()
	endpoint := s.URL

	// 1. 注册 echo2 插件 (符号 [echo2, echo3]) 成功
	p := &plugin.FuncPlugin{
		IOPlugin: plugin.IOPlugin{
			Name: "echo2",
			File: endpoint + "/functions/echo2.zip",
		},
		Functions: []string{"echo2", "echo3"},
	}
	assert.NoError(t, manager.Register(plugin.FUNCTION, p))
	t.Cleanup(func() { manager.Delete(plugin.FUNCTION, "echo2", false) })

	// 2. 尝试注册 misc (符号 [misc, echo3]) — echo3 冲突, 必须失败
	bad := &plugin.FuncPlugin{
		IOPlugin: plugin.IOPlugin{
			Name: "misc",
			File: endpoint + "/functions/echo2.zip",
		},
		Functions: []string{"misc", "echo3"},
	}
	err := manager.Register(plugin.FUNCTION, bad)
	if assert.Error(t, err, "misc should fail with echo3 conflict") {
		assert.Contains(t, err.Error(), "echo3 already exists")
	}

	// 3. 失败注册不得残留脏符号
	//    - misc 必须完全不存在 (失败注册零残留)
	//    - echo3 必须仍属于 echo2 (未被失败注册覆盖)
	if _, ok := manager.GetPluginBySymbol(plugin.FUNCTION, "misc"); ok {
		t.Fatalf("failed register leaked symbol misc")
	}
	owner, ok := manager.GetPluginBySymbol(plugin.FUNCTION, "echo3")
	if !ok {
		t.Fatalf("echo3 symbol missing after failed register")
	}
	if owner != "echo2" {
		t.Fatalf("echo3 owner = %q, want echo2", owner)
	}
}
