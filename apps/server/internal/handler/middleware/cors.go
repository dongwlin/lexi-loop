// Package middleware 承载 HTTP 横切中间件（Handler 子包，不参与版本化）。
// 定义与挂载分离：本包只定义，由 handler/router.go 构造并挂载。
package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// corsMaxAge 是预检结果（Access-Control-Max-Age）的缓存时长。
const corsMaxAge = 12 * time.Hour

// CORS 返回 CORS 中间件：allowedOrigins 是显式的前端 Origin 白名单——私有
// API 不得对任意 Origin 开放（前端 API 与认证集成规范 §11.2）；允许的方法 /
// 请求头 / 暴露响应头按同节的后端 CORS 要求设置。前端凭证走 Bearer 头且
// credentials: 'omit'，因此不开 Allow-Credentials。
//
// 白名单为空时不输出任何 CORS 头，等价于仅同源可用。含通配符 "*" 的白名单
// 视为配置错误，在构造阶段 panic 暴露（gin-contrib/cors 会把 "*" 当作允许
// 任意 Origin）；其余非法配置由 cors.New 在构造阶段 panic。
func CORS(allowedOrigins []string) gin.HandlerFunc {
	if len(allowedOrigins) == 0 {
		return func(c *gin.Context) { c.Next() }
	}
	for _, origin := range allowedOrigins {
		if strings.TrimSpace(origin) == "*" {
			panic(`middleware: cors allowlist must not contain wildcard "*": configure explicit frontend origins`)
		}
	}
	return cors.New(cors.Config{
		AllowOrigins: allowedOrigins,
		AllowMethods: []string{
			http.MethodGet, http.MethodPost, http.MethodPut,
			http.MethodPatch, http.MethodDelete, http.MethodOptions,
		},
		AllowHeaders: []string{
			"Authorization", "Content-Type", "X-Refresh-Request-Id",
			"Idempotency-Key", "If-None-Match", "If-Match",
		},
		ExposeHeaders:    []string{HeaderRequestID, "ETag", "Retry-After"},
		AllowCredentials: false,
		MaxAge:           corsMaxAge,
	})
}
