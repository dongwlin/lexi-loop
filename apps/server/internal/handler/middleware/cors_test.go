package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// corsTestRouter 构造挂载 CORS（白名单来自参数）与一个 GET 端点的测试路由。
func corsTestRouter(allowedOrigins []string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS(allowedOrigins))
	r.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})
	return r
}

// TestCORS_白名单请求按规范输出 验证前端 API 与认证集成规范 §11.2 的
// 后端 CORS 要求：预检放行允许的方法 / 请求头并缓存，实际请求回显
// Allow-Origin、暴露响应头并携带 Vary: Origin。
func TestCORS_白名单请求按规范输出(t *testing.T) {
	t.Parallel()

	r := corsTestRouter([]string{"https://app.lexiloop.dev", "http://localhost:5173"})

	t.Run("预检请求返回204与规范要求的方法和请求头", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/ping", nil)
		req.Header.Set("Origin", "https://app.lexiloop.dev")
		req.Header.Set("Access-Control-Request-Method", http.MethodPatch)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Equal(t, "https://app.lexiloop.dev", w.Header().Get("Access-Control-Allow-Origin"))
		allowMethods := w.Header().Get("Access-Control-Allow-Methods")
		for _, m := range []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"} {
			assert.Contains(t, allowMethods, m)
		}
		allowHeaders := w.Header().Get("Access-Control-Allow-Headers")
		for _, h := range []string{"Authorization", "Content-Type", "X-Refresh-Request-Id", "Idempotency-Key", "If-None-Match", "If-Match"} {
			assert.Contains(t, allowHeaders, h)
		}
		assert.Contains(t, w.Header().Values("Vary"), "Origin")
		assert.NotEmpty(t, w.Header().Get("Access-Control-Max-Age"))
	})

	t.Run("实际请求携带允许Origin并暴露响应头", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		req.Header.Set("Origin", "http://localhost:5173")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "http://localhost:5173", w.Header().Get("Access-Control-Allow-Origin"))
		// HTTP 头名大小写不敏感，lib 按规范形式输出 Etag。
		assert.Equal(t, "X-Request-Id,Etag,Retry-After", w.Header().Get("Access-Control-Expose-Headers"))
		assert.Contains(t, w.Header().Values("Vary"), "Origin")
	})

	t.Run("白名单外的Origin被403拒绝且无放行头", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		req.Header.Set("Origin", "https://evil.example.com")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("无Origin的请求不受影响", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ping", nil))

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
	})
}

// TestCORS_空白名单等价仅同源 验证白名单为空时不输出任何 CORS 头，跨源
// 请求无法获得放行，请求本身仍正常处理。
func TestCORS_空白名单等价仅同源(t *testing.T) {
	t.Parallel()

	r := corsTestRouter(nil)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
	assert.Empty(t, w.Header().Values("Vary"))
}

// TestCORS_通配符白名单在构造期panic 验证 "*" 不被当作允许任意 Origin，
// 而是在构造阶段以 panic 暴露配置错误。
func TestCORS_通配符白名单在构造期panic(t *testing.T) {
	t.Parallel()

	assert.Panics(t, func() { CORS([]string{"*"}) })
	assert.Panics(t, func() { CORS([]string{"https://app.lexiloop.dev", " * "}) })
}
