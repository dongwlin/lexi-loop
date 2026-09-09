package v1

// 版本信息端点的单元测试：只注册 v1 Handler，不启动数据库。

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dongwlin/lexi-loop/apps/server/internal/handler/httpresp"
	"github.com/dongwlin/lexi-loop/apps/server/internal/infra/buildinfo"
)

type versionData struct {
	Version   string `json:"version"`
	BuildTime string `json:"buildTime"`
	GoVersion string `json:"goVersion"`
}

func TestVersionRoute(t *testing.T) {
	engine := newMockAPI(t, nil, nil, nil)
	rec := request(t, engine, "GET", "/api/v1/version", "")
	require.Equal(t, http.StatusOK, rec.Code)
	var e httpresp.Envelope[versionData]
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &e))
	assert.Equal(t, "OK", e.Code)
	assert.Equal(t, "success", e.Message)

	data := e.Data
	// 测试环境无 ldflags 注入，锁定「开发构建」契约（infra/buildinfo）。
	assert.Equal(t, "dev", data.Version)
	assert.Empty(t, data.BuildTime)
	assert.Equal(t, buildinfo.GoVersion(), data.GoVersion)
}
