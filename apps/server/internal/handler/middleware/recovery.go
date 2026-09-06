package middleware

import (
	"fmt"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"github.com/dongwlin/lexi-loop/apps/server/internal/handler/httpresp"
)

// Recovery 返回 panic 恢复中间件：未预期 panic 统一按内部错误经 httpresp
// 输出（500 + CodeInternal，message 不含 panic 值与堆栈，HTTP API 设计规范
// §4.2）；panic 值、堆栈与 request id 只写入服务端日志。
func Recovery(l zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}
			cause := fmt.Errorf("panic: %v", rec)
			e := l.Error()
			if id := FromContext(c); id != "" {
				e = e.Str("request_id", id)
			}
			e.AnErr("panic", cause).
				Str("stack", string(debug.Stack())).
				Msg("recovered from panic")

			// 响应已提交（panic 前已写出内容）时无法再改写状态与响应体，
			// 只能中断链路；未写出的请求统一经 httpresp 渲染内部错误。
			if !c.Writer.Written() {
				httpresp.AbortError(c, cause)
				return
			}
			c.Abort()
		}()
		c.Next()
	}
}
