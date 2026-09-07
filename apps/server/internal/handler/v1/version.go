package v1

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/dongwlin/lexi-loop/apps/server/internal/handler/httpresp"
	"github.com/dongwlin/lexi-loop/apps/server/internal/handler/v1/dto"
	"github.com/dongwlin/lexi-loop/apps/server/internal/infra/buildinfo"
)

// VersionHandler 是版本信息端点（docs/api/meta.md）的 HTTP 适配层，
// 无 Service 依赖：数据直接来自 infra/buildinfo 的构建期注入值。
type VersionHandler struct{}

// NewVersionHandler 构造 VersionHandler。
func NewVersionHandler() *VersionHandler {
	return &VersionHandler{}
}

type getVersionOutput struct {
	Body httpresp.Envelope[dto.VersionInfo]
}

// Register 把版本信息操作注册到 huma API；路径自带 /api/v1 前缀，契约见
// docs/api/meta.md §1。端点无入参且恒成功，不声明 Errors——huma 因此自动
// 追加的 default 错误响应由 spec 归一化移除（handler/openapi.go）。
func (h *VersionHandler) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "get-version",
		Method:      http.MethodGet,
		Path:        "/api/v1/version",
		Summary:     "版本信息",
		Tags:        []string{"Meta"},
	}, h.get)
}

// get 处理 GET /api/v1/version。
func (h *VersionHandler) get(_ context.Context, _ *struct{}) (*getVersionOutput, error) {
	return &getVersionOutput{Body: httpresp.OK(dto.VersionInfo{
		Version:   buildinfo.Version,
		BuildTime: buildinfo.BuildTime,
		GoVersion: buildinfo.GoVersion(),
	})}, nil
}
