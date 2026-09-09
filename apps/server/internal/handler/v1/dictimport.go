package v1

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/dongwlin/lexi-loop/apps/server/internal/handler/httpresp"
	"github.com/dongwlin/lexi-loop/apps/server/internal/handler/v1/dto"
	"github.com/dongwlin/lexi-loop/apps/server/internal/service"
)

// DictImportHandler 是词典导入进度端点（docs/api/meta.md §3）的 HTTP
// 适配层：读取 DictImport 服务的进程内进度快照，不产生任何副作用。
type DictImportHandler struct {
	dictImport service.DictImport
}

// NewDictImportHandler 构造 DictImportHandler。
func NewDictImportHandler(dictImport service.DictImport) *DictImportHandler {
	return &DictImportHandler{dictImport: dictImport}
}

type getDictImportOutput struct {
	Body httpresp.Envelope[dto.DictImportStatus]
}

// Register 把词典导入进度操作注册到 huma API；路径自带 /api/v1 前缀。
// 端点恒成功（进度快照即时可读），不声明 Errors——huma 因此自动追加的
// default 错误响应由 spec 归一化移除（handler/openapi.go）。
func (h *DictImportHandler) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "get-dictionary-import",
		Method:      http.MethodGet,
		Path:        "/api/v1/dictionary-import",
		Summary:     "词典导入进度",
		Tags:        []string{"Meta"},
	}, h.get)
}

// get 处理 GET /api/v1/dictionary-import。
func (h *DictImportHandler) get(_ context.Context, _ *struct{}) (*getDictImportOutput, error) {
	return &getDictImportOutput{Body: httpresp.OK(toDictImportStatus(h.dictImport.Snapshot()))}, nil
}

// toDictImportStatus 把 Service 快照转换为响应 DTO：时间为 RFC3339 UTC
// 字符串，零值 / 空错误映射为 null。
func toDictImportStatus(s service.DictImportSnapshot) dto.DictImportStatus {
	status := dto.DictImportStatus{
		State:          string(s.State),
		SourceVersion:  s.SourceVersion,
		RowsProcessed:  s.RowsProcessed,
		RowsTotal:      s.RowsTotal,
		EntriesWritten: s.EntriesWritten,
	}
	if !s.StartedAt.IsZero() {
		t := s.StartedAt.Format(time.RFC3339)
		status.StartedAt = &t
	}
	if !s.UpdatedAt.IsZero() {
		t := s.UpdatedAt.Format(time.RFC3339)
		status.UpdatedAt = &t
	}
	if s.ErrorMessage != "" {
		e := s.ErrorMessage
		status.Error = &e
	}
	return status
}
