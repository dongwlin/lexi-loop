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
)

// newLoggedRouter 构造挂载 RequestID + Logger 的测试路由，请求日志写入 buf。
func newLoggedRouter(buf *bytes.Buffer) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID())
	r.Use(Logger(zerolog.New(buf)))
	return r
}

// decodeLog 把单行 JSON 日志解析为字段 map。
func decodeLog(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	var fields map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &fields))
	return fields
}

// TestLogger_请求完成记录字段 验证 200 请求记录 method / path（含 query）/
// status / bytes / client_ip / latency 与 request_id 链路字段。
func TestLogger_请求完成记录字段(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	r := newLoggedRouter(&buf)
	r.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	req := httptest.NewRequest(http.MethodGet, "/ping?page=2", nil)
	req.RemoteAddr = "203.0.113.7:1234"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	fields := decodeLog(t, &buf)
	assert.Equal(t, "info", fields["level"])
	assert.Equal(t, "GET", fields["method"])
	assert.Equal(t, "/ping?page=2", fields["path"])
	assert.InDelta(t, http.StatusOK, fields["status"], 0)
	assert.GreaterOrEqual(t, fields["bytes"], float64(len("pong")))
	assert.Equal(t, "203.0.113.7", fields["client_ip"])
	assert.Greater(t, fields["latency"], float64(0))
	assert.Equal(t, w.Header().Get(HeaderRequestID), fields["request_id"])
	assert.Equal(t, "request", fields["message"])
}

// TestLogger_级别随状态码 验证 >= 500 记 ERROR、>= 400 记 WARN、其余 INFO。
func TestLogger_级别随状态码(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name   string
		status int
		level  string
	}{
		{name: "2xx记info", status: http.StatusCreated, level: "info"},
		{name: "4xx记warn", status: http.StatusNotFound, level: "warn"},
		{name: "5xx记error", status: http.StatusInternalServerError, level: "error"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			r := newLoggedRouter(&buf)
			r.GET("/echo", func(c *gin.Context) {
				c.Status(tt.status)
			})

			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/echo", nil))

			require.Equal(t, tt.status, w.Code)
			fields := decodeLog(t, &buf)
			assert.Equal(t, tt.level, fields["level"])
			assert.InDelta(t, tt.status, fields["status"], 0)
		})
	}
}
