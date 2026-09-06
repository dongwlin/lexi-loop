// Package handler 是 HTTP 层组装入口：全局中间件挂载、/api/v1 业务路由组
// （docs/backend/structure.md §4.4）。
package handler

import (
	"github.com/gin-gonic/gin"

	v1 "github.com/dongwlin/lexi-loop/apps/server/internal/handler/v1"
)

// RegisterRoutes 是 HTTP 路由挂载的唯一入口：构造并挂载全局中间件，
// 建立 /api/v1 业务路由组。版本化 Handler 由组合根（internal/app）构造后
// 传入，路径契约见 docs/api/words.md 与 docs/api/reviews.md。
//
// 全局中间件暂用 gin 内置实现；
// TODO: 替换为 handler/middleware 的自定义实现（recovery / logger / request_id / cors）。
func RegisterRoutes(r *gin.Engine, wordH *v1.WordHandler, reviewH *v1.ReviewHandler) {
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

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
