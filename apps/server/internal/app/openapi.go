package app

import (
	"github.com/dongwlin/lexi-loop/apps/server/internal/handler"
)

// BuildOpenAPISpec 离线构造 /api/v1 的 OpenAPI 3.1 spec 产物（JSON 与
// YAML）：不启动 HTTP、不连接数据库。组合根只提供入口，API 构造与
// schema 生成在 handler（docs/backend/structure.md §3：cmd 经 app 使用
// 组合根能力）。
func BuildOpenAPISpec() (jsonSpec, yamlSpec []byte, err error) {
	return handler.BuildOpenAPISpec()
}
