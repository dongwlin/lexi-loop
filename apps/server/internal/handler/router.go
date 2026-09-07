// Package handler 是 HTTP 层组装入口：全局中间件挂载、huma API 构造与
// /api/v1 业务操作注册、OpenAPI spec 离线生成（docs/backend/structure.md
// §4.4）。
package handler

import (
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humagin"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"github.com/dongwlin/lexi-loop/apps/server/internal/handler/httpresp"
	"github.com/dongwlin/lexi-loop/apps/server/internal/handler/middleware"
	"github.com/dongwlin/lexi-loop/apps/server/internal/handler/v1"
)

// Options 是 RegisterRoutes 的横切依赖与配置，由组合根（internal/app）传入。
type Options struct {
	// Log 是请求日志与 panic 恢复使用的 logger（infra/logger 构造）。
	Log zerolog.Logger

	// CORSAllowedOrigins 是允许跨源访问的前端 Origin 白名单；为空时
	// middleware.CORS 不输出 CORS 头（等价仅同源可用）。
	CORSAllowedOrigins []string
}

// humaConfig 构造 huma API 的 OpenAPI 元数据：运行时路由与离线 spec 生成
// 共用，保证两处的标题 / 版本 / 描述一致。不设置 OpenAPIPath / DocsPath /
// SchemasPath——spec 只离线生成落盘，不挂载到线上系统（docs/specs/backend/
// Go 技术栈.md「API 文档」），也因此不引入向响应体注入 $schema 的
// SchemaLinkTransformer，响应体外壳保持纯净。
func humaConfig() huma.Config {
	return huma.Config{
		OpenAPI: &huma.OpenAPI{
			OpenAPI: "3.1.0",
			Info: &huma.Info{
				Title:       "LexiLoop API",
				Version:     "1.0.0",
				Description: "LexiLoop（词环）英语生词复习系统 /api/v1 接口。spec 由 `lexi-loop openapi` 离线生成至 docs/openapi/，请勿手改。",
			},
		},
		Formats:       huma.DefaultFormats,
		DefaultFormat: "application/json",
	}
}

// newAPI 在给定 gin 路由上构造 huma API：先装配统一错误模型（校验失败
// 422 降为 400 等，见 httpresp.UseHumaError），再挂载适配器。业务操作
// 路径自带 /api/v1 前缀，humagin 把 {param} 转换为 gin 的 :param。
func newAPI(r *gin.Engine) huma.API {
	httpresp.UseHumaError()
	return humagin.New(r, humaConfig())
}

// RegisterRoutes 是 HTTP 路由挂载的唯一入口：构造并挂载全局中间件，
// 注册 /api/v1 业务操作。版本化 Handler 与横切依赖由组合根
// （internal/app）构造后传入，路径契约见 docs/api/words.md、
// docs/api/reviews.md 与 docs/api/meta.md。
//
// 全局中间件为 handler/middleware 的自定义实现，顺序：
// request_id（最先挂载，panic / 中断的响应也携带 id）→
// recovery（兜住后续所有环节的 panic）→
// cors（预检与非法 Origin 的请求不进请求日志）→
// logger。
func RegisterRoutes(r *gin.Engine, opts Options, wordH *v1.WordHandler, reviewH *v1.ReviewHandler, versionH *v1.VersionHandler) {
	r.Use(middleware.RequestID())
	r.Use(middleware.Recovery(opts.Log))
	r.Use(middleware.CORS(opts.CORSAllowedOrigins))
	r.Use(middleware.Logger(opts.Log))

	api := newAPI(r)
	wordH.Register(api)
	reviewH.Register(api)
	versionH.Register(api)
}
