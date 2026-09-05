// Package handler 是 HTTP 层组装入口：全局中间件挂载、/api/v1 业务路由组。
package handler

import (
	"github.com/gin-gonic/gin"
)

// RegisterRoutes 是 HTTP 路由挂载的唯一入口（docs/backend/structure.md §4.4）。
//
// MVP 骨架阶段先用 gin 内置中间件占位；业务 Handler 实现后：
//   - 中间件替换为 handler/middleware 下的自定义实现；
//   - 在 /api/v1 组内注册 v1.WordHandler / v1.ReviewHandler，
//     路径契约见 docs/api/words.md 与 docs/api/reviews.md。
//
// 届时本函数将接收具体依赖（日志、Service、版本化 Handler）并在 app 组合根
// 中完成传参（internal/app/provider.go）。
func RegisterRoutes(r *gin.Engine) {
	// 全局中间件 —— 骨架阶段暂用 gin 内置实现。
	// TODO: 替换为 handler/middleware 的自定义实现（recovery / logger / request_id / cors）。
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	// /api/v1 业务路由挂载点。挂载以下路由时使用：
	//
	//	GET    /api/v1/words           生词库列表
	//	POST   /api/v1/words/import    批量导入生词
	//	GET    /api/v1/words/:id       单词详情
	//	PATCH  /api/v1/words/:id       更新复习释义
	//	DELETE /api/v1/words/:id       删除生词（软删除）
	//	POST   /api/v1/reviews         开始一轮复习
	//	POST   /api/v1/reviews/:sessionId/items/:itemId  提交单词复习结果
	//	GET    /api/v1/reviews/:id     获取一轮复习结果
	r.Group("/api/v1")
}
