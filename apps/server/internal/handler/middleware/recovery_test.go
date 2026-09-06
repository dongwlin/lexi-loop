package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dongwlin/lexi-loop/apps/server/internal/apperr"
	"github.com/dongwlin/lexi-loop/apps/server/internal/handler/httpresp"
)

// TestRecovery_未写出响应时输出内部错误 验证 panic 被恢复为 500 的统一
// 响应：CodeInternal + 安全 message，不向客户端暴露 panic 值与堆栈；
// panic 详情进服务端日志并携带 request_id。
func TestRecovery_未写出响应时输出内部错误(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	var buf bytes.Buffer
	r := gin.New()
	r.Use(RequestID())
	r.Use(Recovery(zerolog.New(&buf)))
	r.GET("/boom", func(c *gin.Context) {
		panic("boom-secret")
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/boom", nil))

	require.Equal(t, http.StatusInternalServerError, w.Code)
	var resp httpresp.Response
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, apperr.CodeInternal, resp.Code)
	assert.Equal(t, apperr.Internal(nil).Message, resp.Message)
	assert.NotContains(t, w.Body.String(), "boom-secret", "panic 值不得进入响应")
	assert.NotContains(t, w.Body.String(), "goroutine", "堆栈不得进入响应")
	assert.NotEmpty(t, w.Header().Get(HeaderRequestID), "panic 响应仍应携带 request id")

	fields := decodeLog(t, &buf)
	assert.Equal(t, "error", fields["level"])
	assert.Contains(t, fields["panic"], "boom-secret")
	assert.NotEmpty(t, fields["stack"])
	assert.Equal(t, w.Header().Get(HeaderRequestID), fields["request_id"])
}

// TestRecovery_响应已提交时不再改写 验证 panic 前已写出响应时不再追加
// 内部错误响应体，仅中断链路，已完成的部分响应保持原样。
func TestRecovery_响应已提交时不再改写(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(Recovery(zerolog.Nop()))
	r.GET("/boom-late", func(c *gin.Context) {
		c.String(http.StatusOK, "partial")
		panic("late boom")
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/boom-late", nil))

	assert.Equal(t, http.StatusOK, w.Code, "响应已提交，状态不得被改写")
	assert.Equal(t, "partial", w.Body.String())
	assert.NotContains(t, w.Body.String(), "internal server error")
}
