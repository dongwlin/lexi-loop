package v1

import (
	"github.com/gin-gonic/gin"

	"github.com/dongwlin/lexi-loop/apps/server/internal/domain"
	"github.com/dongwlin/lexi-loop/apps/server/internal/handler/httpresp"
	"github.com/dongwlin/lexi-loop/apps/server/internal/handler/v1/dto"
	"github.com/dongwlin/lexi-loop/apps/server/internal/service"
)

// ReviewHandler 是复习端点（docs/api/reviews.md）的 HTTP 适配层。
type ReviewHandler struct {
	svc *service.Review
}

// NewReviewHandler 构造 ReviewHandler。
func NewReviewHandler(svc *service.Review) *ReviewHandler {
	return &ReviewHandler{svc: svc}
}

// Start 处理 POST /api/v1/reviews：开始一轮复习（抽词 + 建 session）。
func (h *ReviewHandler) Start(c *gin.Context) {
	var req dto.StartSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		invalidParam(c, err)
		return
	}
	res, err := h.svc.StartSession(c.Request.Context(), service.StartSessionRequest{Count: req.Count})
	if err != nil {
		httpresp.Error(c, err)
		return
	}
	httpresp.Success(c, dto.NewStartSessionResponse(res))
}

// Submit 处理 POST /api/v1/reviews/:sessionId/items/:itemId：提交一个单词
// 的复习结果（幂等，重复与错配请求按契约短路返回成功）。
func (h *ReviewHandler) Submit(c *gin.Context) {
	sessionID, ok := pathUUID(c, "sessionId")
	if !ok {
		return
	}
	itemID, ok := pathUUID(c, "itemId")
	if !ok {
		return
	}
	var req dto.SubmitResultRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		invalidParam(c, err)
		return
	}
	if err := h.svc.SubmitResult(c.Request.Context(), service.SubmitResultRequest{
		SessionID: sessionID,
		ItemID:    itemID,
		Result:    domain.ReviewResult(req.Result),
	}); err != nil {
		httpresp.Error(c, err)
		return
	}
	httpresp.Success(c, nil)
}

// Get 处理 GET /api/v1/reviews/:id：获取一轮复习的汇总与逐词结果。
func (h *ReviewHandler) Get(c *gin.Context) {
	sessionID, ok := pathUUID(c, "id")
	if !ok {
		return
	}
	res, err := h.svc.GetSession(c.Request.Context(), service.GetSessionRequest{SessionID: sessionID})
	if err != nil {
		httpresp.Error(c, err)
		return
	}
	httpresp.Success(c, dto.NewGetSessionResponse(res))
}
