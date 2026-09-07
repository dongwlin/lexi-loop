package handler

// 词典导入进度端点的组件测试：快照是进程内存态，纯单元测试即可覆盖
// 响应契约（idle 零值 → 各字段缺省映射为 null / 0）；状态机行为本身
// 由 service 包的集成测试覆盖。Handler 的 DictImport 只读快照，
// nil db / nil importer 不参与该路径。

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "github.com/dongwlin/lexi-loop/apps/server/internal/handler/v1"
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
	engine := gin.New()
	// 未调用 StartAutoImport 的服务保持 idle：csvPath 指向不存在的文件，
	// db / importer 为 nil（快照路径不触达）。
	svc := service.NewDictImport(nil, nil, "/nonexistent/ecdict.csv", true, zerolog.Nop())
	RegisterRoutes(engine, Options{Log: zerolog.Nop()},
		v1.NewWordHandler(nil), v1.NewReviewHandler(nil), v1.NewVersionHandler(),
		v1.NewDictImportHandler(svc))

	rec := doJSON(t, engine, "GET", "/api/v1/dictionary-import", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	e := decodeEnvelope(t, rec.Body.Bytes())
	assert.Equal(t, "OK", e.Code)
	assert.Equal(t, "success", e.Message)

	var data dictImportData
	require.NoError(t, json.Unmarshal(e.Data, &data))
	assert.Equal(t, "idle", data.State, "未启动导入时为 idle")
	assert.Empty(t, data.SourceVersion)
	assert.Zero(t, data.RowsProcessed)
	assert.Zero(t, data.RowsTotal)
	assert.Zero(t, data.EntriesWritten)
	assert.Nil(t, data.StartedAt)
	assert.Nil(t, data.UpdatedAt)
	assert.Nil(t, data.Error)
}
