package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestRequestID_生成透传与截断 覆盖 request id 的来源规则：未携带时生成
// 短 UUID、透传上游 id（去首尾空白）、超长截断；同时验证响应头与上下文
// 取值一致。
func TestRequestID_生成透传与截断(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name   string
		header string
		want   string // "generated" 表示期望新生成的短 id，否则期望响应头恰为该值
	}{
		{name: "无请求头生成短UUID", header: "", want: "generated"},
		{name: "纯空白视为未携带", header: "   ", want: "generated"},
		{name: "透传上游id并去首尾空白", header: "  upstream-trace-id  ", want: "upstream-trace-id"},
		{name: "超长透传截断", header: strings.Repeat("a", 200), want: strings.Repeat("a", maxIncomingIDLen)},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := gin.New()
			r.Use(RequestID())
			r.GET("/ping", func(c *gin.Context) {
				c.String(http.StatusOK, FromContext(c))
			})

			req := httptest.NewRequest(http.MethodGet, "/ping", nil)
			if tt.header != "" {
				req.Header.Set(HeaderRequestID, tt.header)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			got := w.Header().Get(HeaderRequestID)
			assert.Equal(t, got, w.Body.String(), "上下文取值应与响应头一致")
			if tt.want == "generated" {
				assert.Len(t, got, generatedIDLen)
			} else {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

// TestRequestID_未挂载时FromContext为空 保证 FromContext 对未挂载
// RequestID 的路由不误报。
func TestRequestID_未挂载时FromContext为空(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, FromContext(c))
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ping", nil))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, w.Body.String())
}
