package v1

// 词典导入进度端点的单元测试：使用接口替身验证 checking / idle 快照的
// 响应契约，状态机与初始状态由 service/v1 的测试覆盖。

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dongwlin/lexi-loop/apps/server/internal/handler/httpresp"
	"github.com/dongwlin/lexi-loop/apps/server/internal/service"
)

type dictImportData struct {
	State          string  `json:"state"`
	SourceVersion  string  `json:"sourceVersion"`
	RowsProcessed  int64   `json:"rowsProcessed"`
	RowsTotal      int64   `json:"rowsTotal"`
	EntriesWritten int64   `json:"entriesWritten"`
	StartedAt      *string `json:"startedAt"`
	UpdatedAt      *string `json:"updatedAt"`
	Error          *string `json:"error"`
}

func TestDictImportRoute(t *testing.T) {
	for _, tc := range []struct {
		name  string
		state service.DictImportState
	}{
		{"检查中快照", service.DictImportStateChecking},
		{"空闲快照", service.DictImportStateIdle},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc := &dictImportStub{snapshot: service.DictImportSnapshot{State: tc.state}}
			engine := newMockAPI(t, nil, nil, svc)
			rec := request(t, engine, "GET", "/api/v1/dictionary-import", "")
			require.Equal(t, http.StatusOK, rec.Code)
			var e httpresp.Envelope[dictImportData]
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &e))
			assert.Equal(t, "OK", e.Code)
			assert.Equal(t, "success", e.Message)

			data := e.Data
			assert.Equal(t, string(tc.state), data.State)
			assert.Empty(t, data.SourceVersion)
			assert.Zero(t, data.RowsProcessed)
			assert.Zero(t, data.RowsTotal)
			assert.Zero(t, data.EntriesWritten)
			assert.Nil(t, data.StartedAt)
			assert.Nil(t, data.UpdatedAt)
			assert.Nil(t, data.Error)
			assert.Equal(t, 1, svc.calls)
		})
	}
}
