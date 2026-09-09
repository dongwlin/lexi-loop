package app

import (
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/uptrace/bun"

	"github.com/dongwlin/lexi-loop/apps/server/internal/handler"
	handlerv1 "github.com/dongwlin/lexi-loop/apps/server/internal/handler/v1"
	"github.com/dongwlin/lexi-loop/apps/server/internal/importer/ecdict"
	"github.com/dongwlin/lexi-loop/apps/server/internal/infra/config"
	servicev1 "github.com/dongwlin/lexi-loop/apps/server/internal/service/v1"
)

// newEngine 创建 Gin Engine：组合根显式构造具体 Service 与版本化 Handler
// （Handler 依赖 service 接口），全局中间件与 /api/v1 业务路由统一经 handler.RegisterRoutes
// 挂载，基础设施端点直接挂在根路径。词典自动导入服务随引擎一并组装，
// 由调用方在 HTTP 服务启动后拉起（返回值即该服务）。
func newEngine(cfg *config.Config, db *bun.DB, log zerolog.Logger) (*gin.Engine, *servicev1.DictImport) {
	dictionarySvc := servicev1.NewDictionary(db)
	wordSvc := servicev1.NewWord(db, dictionarySvc)
	reviewSvc := servicev1.NewReview(db, servicev1.NewWeightedSampler(servicev1.NewTimeSeededSource()))
	dictImportSvc := servicev1.NewDictImport(db, ecdict.NewImporter(db, ecdict.DefaultBatchSize), cfg.Dict.CSVPath, cfg.Dict.AutoCheck, log)

	wordHandler := handlerv1.NewWordHandler(wordSvc)
	reviewHandler := handlerv1.NewReviewHandler(reviewSvc)
	versionHandler := handlerv1.NewVersionHandler()
	dictImportHandler := handlerv1.NewDictImportHandler(dictImportSvc)

	engine := gin.New()
	handler.RegisterRoutes(engine, handler.Options{
		Log:                log,
		CORSAllowedOrigins: cfg.HTTP.CORSAllowedOrigins,
	}, wordHandler, reviewHandler, versionHandler, dictImportHandler)
	// 健康检查属基础设施端点，不参与业务版本（docs/specs/backend/HTTP API 设计规范.md §2.4）。
	engine.GET("/healthz", handleHealthz)
	return engine, dictImportSvc
}
