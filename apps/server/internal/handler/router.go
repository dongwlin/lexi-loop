// Package handler 是 HTTP 层组装入口：全局中间件挂载、/api/v1 业务路由组
// （docs/backend/structure.md §4.4）。
package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"github.com/dongwlin/lexi-loop/apps/server/internal/handler/middleware"
	v1 "github.com/dongwlin/lexi-loop/apps/server/internal/handler/v1"
)

// Options 是 RegisterRoutes 的横切依赖与配置，由组合根（internal/app）传入。
type Options struct {
	// Log 是请求日志与 panic 恢复使用的 logger（infra/logger 构造）。
	Log zerolog.Logger

	// CORSAllowedOrigins 是允许跨源访问的前端 Origin 白名单；为空时
	// middleware.CORS 不输出 CORS 头（等价仅同源可用）。
	CORSAllowedOrigins []string
}

// RegisterRoutes 是 HTTP 路由挂载的唯一入口：构造并挂载全局中间件，
// 建立 /api/v1 业务路由组。版本化 Handler 与横切依赖由组合根
// （internal/app）构造后传入，路径契约见 docs/api/words.md 与
// docs/api/reviews.md。
//
// 全局中间件为 handler/middleware 的自定义实现，顺序：
// request_id（最先挂载，panic / 中断的响应也携带 id）→
// recovery（兜住后续所有环节的 panic）→
// cors（预检与非法 Origin 的请求不进请求日志）→
// logger。
func RegisterRoutes(r *gin.Engine, opts Options, wordH *v1.WordHandler, reviewH *v1.ReviewHandler) {
	r.Use(middleware.RequestID())
	r.Use(middleware.Recovery(opts.Log))
	r.Use(middleware.CORS(opts.CORSAllowedOrigins))
	r.Use(middleware.Logger(opts.Log))

	v1Group := r.Group("/api/v1")
	{
		words := v1Group.Group("/words")
		words.POST("/import", wordH.Import)
		words.GET("", wordH.List)
		words.GET("/:id", wordH.Get)
		words.PATCH("/:id", wordH.UpdateReviewMeaning)
		words.DELETE("/:id", wordH.Delete)

		reviews := v1Group.Group("/reviews")
		reviews.POST("", reviewH.Start)
		reviews.POST("/:sessionId/items/:itemId", reviewH.Submit)
		reviews.GET("/:id", reviewH.Get)
	}
}
