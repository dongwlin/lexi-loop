package v1

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/dongwlin/lexi-loop/apps/server/internal/domain"
	"github.com/dongwlin/lexi-loop/apps/server/internal/handler/httpresp"
	"github.com/dongwlin/lexi-loop/apps/server/internal/handler/v1/dto"
	"github.com/dongwlin/lexi-loop/apps/server/internal/service"
)

// ReviewHandler 是复习端点（docs/api/reviews.md）的 HTTP 适配层。
type ReviewHandler struct {
	svc *service.Review
}

// NewReviewHandler 构造 ReviewHandler。svc 为 nil 时仅用于离线 OpenAPI
// spec 生成（操作函数不会被调用）。
func NewReviewHandler(svc *service.Review) *ReviewHandler {
	return &ReviewHandler{svc: svc}
}

// Register 把复习操作注册到 huma API；路径自带 /api/v1 前缀，
// 契约见 docs/api/reviews.md §1。
func (h *ReviewHandler) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "start-review-session",
		Method:      http.MethodPost,
		Path:        "/api/v1/reviews",
		Summary:     "开始一轮复习（抽词 + 建 session）",
		Tags:        []string{"Reviews"},
		Errors:      []int{http.StatusBadRequest, http.StatusUnprocessableEntity},
	}, h.start)

	huma.Register(api, huma.Operation{
		OperationID: "submit-review-result",
		Method:      http.MethodPost,
		Path:        "/api/v1/reviews/{sessionId}/items/{itemId}",
		Summary:     "提交一个单词的复习结果（幂等）",
		Tags:        []string{"Reviews"},
		Errors:      []int{http.StatusBadRequest},
	}, h.submit)

	huma.Register(api, huma.Operation{
		OperationID: "get-review-session",
		Method:      http.MethodGet,
		Path:        "/api/v1/reviews/{id}",
		Summary:     "获取一轮复习的汇总与逐词结果",
		Tags:        []string{"Reviews"},
		Errors:      []int{http.StatusBadRequest, http.StatusNotFound},
	}, h.get)
}

// ---- 开始一轮复习（api/reviews.md §2）----

type startSessionInput struct {
	Body dto.StartSessionRequest
}

type startSessionOutput struct {
	Body httpresp.Envelope[dto.StartSessionResponse]
}

// start 处理 POST /api/v1/reviews。
func (h *ReviewHandler) start(ctx context.Context, input *startSessionInput) (*startSessionOutput, error) {
	res, err := h.svc.StartSession(ctx, service.StartSessionRequest{Count: input.Body.Count})
	if err != nil {
		return nil, httpresp.FromError(err)
	}
	return &startSessionOutput{Body: httpresp.OK(dto.NewStartSessionResponse(res))}, nil
}

// ---- 提交单词结果（api/reviews.md §3）----

type submitResultInput struct {
	SessionID uuid.UUID `path:"sessionId"`
	ItemID    uuid.UUID `path:"itemId"`
	Body      dto.SubmitResultRequest
}

// submit 处理 POST /api/v1/reviews/{sessionId}/items/{itemId}：幂等，
// 重复与错配请求按契约短路返回成功。
func (h *ReviewHandler) submit(ctx context.Context, input *submitResultInput) (*noDataOutput, error) {
	if err := h.svc.SubmitResult(ctx, service.SubmitResultRequest{
		SessionID: input.SessionID,
		ItemID:    input.ItemID,
		Result:    domain.ReviewResult(input.Body.Result),
	}); err != nil {
		return nil, httpresp.FromError(err)
	}
	return &noDataOutput{Body: httpresp.OK(dto.NoData{})}, nil
}

// ---- 获取一轮结果（api/reviews.md §5）----

type getSessionInput struct {
	ID uuid.UUID `path:"id"`
}

type getSessionOutput struct {
	Body httpresp.Envelope[dto.GetSessionResponse]
}

// get 处理 GET /api/v1/reviews/{id}。
func (h *ReviewHandler) get(ctx context.Context, input *getSessionInput) (*getSessionOutput, error) {
	res, err := h.svc.GetSession(ctx, service.GetSessionRequest{SessionID: input.ID})
	if err != nil {
		return nil, httpresp.FromError(err)
	}
	return &getSessionOutput{Body: httpresp.OK(dto.NewGetSessionResponse(res))}, nil
}
