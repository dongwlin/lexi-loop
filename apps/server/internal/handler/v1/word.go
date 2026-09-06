// Package v1 承载 /api/v1 版本化业务 Handler：只做参数绑定、基础校验、
// DTO 转换与 Service 调用，不写业务逻辑；响应一律经 httpresp 输出，
// apperr.Kind → HTTP 状态映射只在 httpresp（docs/backend/structure.md §4.4）。
package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/dongwlin/lexi-loop/apps/server/internal/apperr"
	"github.com/dongwlin/lexi-loop/apps/server/internal/handler/httpresp"
	"github.com/dongwlin/lexi-loop/apps/server/internal/handler/v1/dto"
	"github.com/dongwlin/lexi-loop/apps/server/internal/service"
)

// WordHandler 是生词库端点（docs/api/words.md）的 HTTP 适配层。
type WordHandler struct {
	svc *service.Word
}

// NewWordHandler 构造 WordHandler。
func NewWordHandler(svc *service.Word) *WordHandler {
	return &WordHandler{svc: svc}
}

// Import 处理 POST /api/v1/words/import：批量导入生词（累计遇词次数）。
func (h *WordHandler) Import(c *gin.Context) {
	var req dto.ImportWordsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		invalidParam(c, err)
		return
	}
	words := make([]service.ImportWordItem, len(req.Words))
	for i, w := range req.Words {
		words[i] = service.ImportWordItem{Word: w.Word, Count: w.Count}
	}
	res, err := h.svc.ImportWords(c.Request.Context(), service.ImportWordsRequest{Words: words})
	if err != nil {
		httpresp.Error(c, err)
		return
	}
	httpresp.Success(c, dto.NewImportWordsResponse(res))
}

// List 处理 GET /api/v1/words：生词库分页列表（含 search）。
// page / pageSize 的默认与上限（默认 1 / 20，上限 100）在此归一
// （HTTP API 设计规范 §8.5）。
func (h *WordHandler) List(c *gin.Context) {
	var query dto.ListWordsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		invalidParam(c, err)
		return
	}
	page, pageSize := query.Page, query.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	res, err := h.svc.ListWords(c.Request.Context(), service.ListWordsRequest{
		Page:     page,
		PageSize: pageSize,
		Search:   query.Search,
	})
	if err != nil {
		httpresp.Error(c, err)
		return
	}
	httpresp.Success(c, dto.NewListWordsResponse(page, pageSize, res.Total, res.Items))
}

// Get 处理 GET /api/v1/words/:id：单词详情。
func (h *WordHandler) Get(c *gin.Context) {
	id, ok := pathUUID(c, "id")
	if !ok {
		return
	}
	item, err := h.svc.GetWord(c.Request.Context(), id)
	if err != nil {
		httpresp.Error(c, err)
		return
	}
	httpresp.Success(c, dto.NewWordDetail(item))
}

// UpdateReviewMeaning 处理 PATCH /api/v1/words/:id：更新（或传 null 清除）
// 用户自定义复习释义。
func (h *WordHandler) UpdateReviewMeaning(c *gin.Context) {
	id, ok := pathUUID(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateReviewMeaningRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		invalidParam(c, err)
		return
	}
	if err := h.svc.UpdateReviewMeaning(c.Request.Context(), id, service.UpdateReviewMeaningRequest{
		CustomReviewMeaning: dto.ToDomainMeanings(req.CustomReviewMeaning),
	}); err != nil {
		httpresp.Error(c, err)
		return
	}
	httpresp.Success(c, nil)
}

// Delete 处理 DELETE /api/v1/words/:id：软删除生词。
func (h *WordHandler) Delete(c *gin.Context) {
	id, ok := pathUUID(c, "id")
	if !ok {
		return
	}
	if err := h.svc.DeleteWord(c.Request.Context(), id); err != nil {
		httpresp.Error(c, err)
		return
	}
	httpresp.Success(c, nil)
}

// pathUUID 解析路径参数中的 UUID；解析失败时输出 400 响应并返回 false
// （路径参数是 UUID v7，路径与 JSON 中都按字符串传输，docs/api/words.md §1）。
func pathUUID(c *gin.Context, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(name))
	if err != nil {
		invalidParam(c, err)
		return uuid.Nil, false
	}
	return id, true
}

// invalidParam 统一输出参数校验失败的失败响应（HTTP API 设计规范 §3.2、§4.1）。
func invalidParam(c *gin.Context, cause error) {
	httpresp.Error(c, apperr.New(apperr.InvalidArgument, apperr.CodeValidationFailed, "invalid parameter", cause))
}
