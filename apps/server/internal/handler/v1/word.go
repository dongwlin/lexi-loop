// Package v1 承载 /api/v1 版本化业务操作：请求绑定、基础校验与请求体
// schema 由 huma 按 DTO 标签声明式完成，操作函数只做 DTO 转换与 Service
// 调用，不写业务逻辑；成功外壳与错误模型一律经 httpresp 输出，
// apperr.Kind → HTTP 状态映射只在 httpresp（docs/backend/structure.md §4.4）。
package v1

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/dongwlin/lexi-loop/apps/server/internal/handler/httpresp"
	"github.com/dongwlin/lexi-loop/apps/server/internal/handler/v1/dto"
	"github.com/dongwlin/lexi-loop/apps/server/internal/service"
)

// noDataOutput 是无额外数据端点（更新 / 删除 / 提交）的统一输出：
// data 渲染为空对象 {}（HTTP API 设计规范 §5.1）。
type noDataOutput struct {
	Body httpresp.Envelope[dto.NoData]
}

// WordHandler 是生词库端点（docs/api/words.md）的 HTTP 适配层。
type WordHandler struct {
	svc service.Word
}

// NewWordHandler 构造 WordHandler。svc 为 nil 时仅用于离线 OpenAPI spec
// 生成（操作函数不会被调用）。
func NewWordHandler(svc service.Word) *WordHandler {
	return &WordHandler{svc: svc}
}

// Register 把生词库操作注册到 huma API；路径自带 /api/v1 前缀，
// 契约见 docs/api/words.md §1。
func (h *WordHandler) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "import-words",
		Method:      http.MethodPost,
		Path:        "/api/v1/words/import",
		Summary:     "批量导入生词（累计遇词次数）",
		Tags:        []string{"Words"},
		Errors:      []int{http.StatusBadRequest},
	}, h.importWords)

	huma.Register(api, huma.Operation{
		OperationID: "list-words",
		Method:      http.MethodGet,
		Path:        "/api/v1/words",
		Summary:     "生词库分页列表（含搜索）",
		Tags:        []string{"Words"},
		Errors:      []int{http.StatusBadRequest},
	}, h.list)

	huma.Register(api, huma.Operation{
		OperationID: "get-word",
		Method:      http.MethodGet,
		Path:        "/api/v1/words/{id}",
		Summary:     "单词详情",
		Tags:        []string{"Words"},
		Errors:      []int{http.StatusBadRequest, http.StatusNotFound},
	}, h.get)

	huma.Register(api, huma.Operation{
		OperationID: "update-word-review-meaning",
		Method:      http.MethodPatch,
		Path:        "/api/v1/words/{id}",
		Summary:     "更新用户自定义复习释义（传 null 清除）",
		Tags:        []string{"Words"},
		Errors:      []int{http.StatusBadRequest, http.StatusNotFound},
	}, h.updateReviewMeaning)

	huma.Register(api, huma.Operation{
		OperationID: "delete-word",
		Method:      http.MethodDelete,
		Path:        "/api/v1/words/{id}",
		Summary:     "删除生词（软删除）",
		Tags:        []string{"Words"},
		Errors:      []int{http.StatusBadRequest, http.StatusNotFound},
	}, h.delete)
}

// ---- 导入（api/words.md §2）----

type importWordsInput struct {
	Body dto.ImportWordsRequest
}

type importWordsOutput struct {
	Body httpresp.Envelope[dto.ImportWordsResponse]
}

// importWords 处理 POST /api/v1/words/import。
func (h *WordHandler) importWords(ctx context.Context, input *importWordsInput) (*importWordsOutput, error) {
	words := make([]service.ImportWordItem, len(input.Body.Words))
	for i, w := range input.Body.Words {
		words[i] = service.ImportWordItem{Word: w.Word, Count: w.Count}
	}
	res, err := h.svc.ImportWords(ctx, service.ImportWordsRequest{Words: words})
	if err != nil {
		return nil, httpresp.FromError(err)
	}
	return &importWordsOutput{Body: httpresp.OK(dto.NewImportWordsResponse(res))}, nil
}

// ---- 列表与详情（api/words.md §3–§4）----

type listWordsInput struct {
	dto.ListWordsQuery
}

type listWordsOutput struct {
	Body httpresp.Envelope[dto.ListWordsResponse]
}

// list 处理 GET /api/v1/words。page / pageSize 的缺省由 DTO 的 schema
// default 声明，越界值（<1 / >100）在此归一（HTTP API 设计规范 §8.5）。
func (h *WordHandler) list(ctx context.Context, input *listWordsInput) (*listWordsOutput, error) {
	page, pageSize := input.Page, input.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	res, err := h.svc.ListWords(ctx, service.ListWordsRequest{
		Page:     page,
		PageSize: pageSize,
		Search:   input.Search,
	})
	if err != nil {
		return nil, httpresp.FromError(err)
	}
	return &listWordsOutput{Body: httpresp.OK(dto.NewListWordsResponse(page, pageSize, res.Total, res.Items))}, nil
}

// wordPathInput 是 /api/v1/words/{id} 的路径参数：UUID v7，按字符串传输
// （docs/api/words.md §1），解析失败由 huma 校验降级为 400。
type wordPathInput struct {
	ID uuid.UUID `path:"id"`
}

type getWordOutput struct {
	Body httpresp.Envelope[dto.WordDetail]
}

// get 处理 GET /api/v1/words/{id}。
func (h *WordHandler) get(ctx context.Context, input *wordPathInput) (*getWordOutput, error) {
	item, err := h.svc.GetWord(ctx, input.ID)
	if err != nil {
		return nil, httpresp.FromError(err)
	}
	return &getWordOutput{Body: httpresp.OK(dto.NewWordDetail(item))}, nil
}

// ---- 更新复习释义（api/words.md §5）----

type updateReviewMeaningInput struct {
	ID   uuid.UUID `path:"id"`
	Body dto.UpdateReviewMeaningRequest
}

// updateReviewMeaning 处理 PATCH /api/v1/words/{id}：只写
// user_words.custom_review_meaning，绝不修改 dictionary_entries。
func (h *WordHandler) updateReviewMeaning(ctx context.Context, input *updateReviewMeaningInput) (*noDataOutput, error) {
	if err := h.svc.UpdateReviewMeaning(ctx, input.ID, service.UpdateReviewMeaningRequest{
		CustomReviewMeaning: dto.ToDomainMeanings(input.Body.CustomReviewMeaning),
	}); err != nil {
		return nil, httpresp.FromError(err)
	}
	return &noDataOutput{Body: httpresp.OK(dto.NoData{})}, nil
}

// delete 处理 DELETE /api/v1/words/{id}：软删除。
func (h *WordHandler) delete(ctx context.Context, input *wordPathInput) (*noDataOutput, error) {
	if err := h.svc.DeleteWord(ctx, input.ID); err != nil {
		return nil, httpresp.FromError(err)
	}
	return &noDataOutput{Body: httpresp.OK(dto.NoData{})}, nil
}
