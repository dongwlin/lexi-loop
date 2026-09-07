package handler

// 版本信息端点的组件测试：无入参、恒成功且不依赖数据库，作为纯单元测试
// 运行（-short 同样执行）。业务 Handler 传 nil Service 构造完整路由（与
// 离线 spec 生成相同，仅注册不触达），版本端点本身无 Service 依赖。

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "github.com/dongwlin/lexi-loop/apps/server/internal/handler/v1"
	"github.com/dongwlin/lexi-loop/apps/server/internal/infra/buildinfo"
)

type versionData struct {
	Version   string `json:"version"`
	BuildTime string `json:"buildTime"`
	GoVersion string `json:"goVersion"`
}

func TestVersionRoute(t *testing.T) {
	engine := gin.New()
	RegisterRoutes(engine, Options{Log: zerolog.Nop()},
		v1.NewWordHandler(nil), v1.NewReviewHandler(nil), v1.NewVersionHandler(),
		v1.NewDictImportHandler(nil))

	rec := doJSON(t, engine, "GET", "/api/v1/version", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	e := decodeEnvelope(t, rec.Body.Bytes())
	assert.Equal(t, "OK", e.Code)
	assert.Equal(t, "success", e.Message)

	var data versionData
	require.NoError(t, json.Unmarshal(e.Data, &data))
	// 测试环境无 ldflags 注入，锁定「开发构建」契约（infra/buildinfo）。
	assert.Equal(t, "dev", data.Version)
	assert.Empty(t, data.BuildTime)
	assert.Equal(t, buildinfo.GoVersion(), data.GoVersion)
}
