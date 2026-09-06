package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// HeaderRequestID 是请求唯一标识的响应头（HTTP API 设计规范 §4.3：只放
// 响应头不放响应体，由服务端生成；跨服务调用时透传）。
const HeaderRequestID = "X-Request-Id"

// generatedIDLen 是自生成 request id 的长度：按规范建议使用短 UUID
// （随机 UUID 截取），实例内排查足够且便于日志阅读。
const generatedIDLen = 8

// maxIncomingIDLen 限制透传请求头的最大长度，防止超长值污染日志与响应头。
const maxIncomingIDLen = 128

// ctxKeyRequestID 是 request id 在 gin.Context 中的存储键；包外只经
// FromContext 读取。
const ctxKeyRequestID = "request_id"

// RequestID 返回 request id 中间件：优先透传客户端 / 上游携带的
// X-Request-Id（去首尾空白、超长截断，空白视为未携带），否则生成短 UUID；
// 同时写入响应头与 gin.Context。应最先挂载，使 panic / 中断的响应也携带 id。
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := strings.TrimSpace(c.GetHeader(HeaderRequestID))
		if id == "" {
			id = uuid.NewString()[:generatedIDLen]
		} else if len(id) > maxIncomingIDLen {
			id = id[:maxIncomingIDLen]
		}
		c.Set(ctxKeyRequestID, id)
		// 进入后续链路前写响应头，后续任何阶段（含提前 abort）的响应都携带 id。
		c.Header(HeaderRequestID, id)
		c.Next()
	}
}

// FromContext 返回当前请求的 request id；未挂载 RequestID 时为空字符串。
func FromContext(c *gin.Context) string {
	if v, ok := c.Get(ctxKeyRequestID); ok {
		id, _ := v.(string)
		return id
	}
	return ""
}
