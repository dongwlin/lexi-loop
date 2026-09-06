package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// Logger 返回基于 zerolog 的请求日志中间件：每请求一行，记录 method /
// path（含 query）/ status / bytes / client_ip / latency，并携带
// X-Request-Id 链路字段。状态 >= 500 记 ERROR、>= 400 记 WARN，其余 INFO。
func Logger(l zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		status := c.Writer.Status()
		var e *zerolog.Event
		switch {
		case status >= http.StatusInternalServerError:
			e = l.Error()
		case status >= http.StatusBadRequest:
			e = l.Warn()
		default:
			e = l.Info()
		}
		if id := FromContext(c); id != "" {
			e = e.Str("request_id", id)
		}
		path := c.Request.URL.Path
		if raw := c.Request.URL.RawQuery; raw != "" {
			path += "?" + raw
		}
		size := c.Writer.Size()
		if size < 0 {
			size = 0
		}
		e.Str("method", c.Request.Method).
			Str("path", path).
			Int("status", status).
			Int("bytes", size).
			Str("client_ip", c.ClientIP()).
			Dur("latency", time.Since(start)).
			Msg("request")
	}
}
