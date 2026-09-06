package dto

import (
	"time"

	"github.com/dongwlin/lexi-loop/apps/server/internal/service"
)

// ---- 开始一轮复习（api/reviews.md §2）----

// StartSessionRequest 是 POST /api/v1/reviews 的请求体：本轮想复习的数量，
// 超过可复习词数时由服务端按 min(count, available) 截断（D009）。
type StartSessionRequest struct {
	Count int `json:"count" minimum:"1"`
}

// SessionWordItem 是本轮抽中单词的展示项。
type SessionWordItem struct {
	ItemID                 string    `json:"itemId"`
	Word                   string    `json:"word"`
	Phonetic               string    `json:"phonetic"`
	EffectiveReviewMeaning []Meaning `json:"effectiveReviewMeaning" nullable:"false"`
}

// StartSessionResponse 是开始一轮复习的响应数据。
type StartSessionResponse struct {
	SessionID      string            `json:"sessionId"`
	RequestedCount int               `json:"requestedCount"`
	TotalCount     int               `json:"totalCount"`
	Items          []SessionWordItem `json:"items" nullable:"false"`
}

// NewStartSessionResponse 转换开始一轮的结果。
func NewStartSessionResponse(r *service.StartSessionResult) StartSessionResponse {
	items := make([]SessionWordItem, 0, len(r.Items))
	for _, item := range r.Items {
		items = append(items, SessionWordItem{
			ItemID:                 item.ItemID.String(),
			Word:                   item.Headword,
			Phonetic:               item.Phonetic,
			EffectiveReviewMeaning: NewMeanings(item.EffectiveMeaning),
		})
	}
	return StartSessionResponse{
		SessionID:      r.SessionID.String(),
		RequestedCount: r.RequestedCount,
		TotalCount:     r.TotalCount,
		Items:          items,
	}
}

// ---- 提交单词结果（api/reviews.md §3）----

// SubmitResultRequest 是 POST /api/v1/reviews/:sessionId/items/:itemId 的
// 请求体：result 只允许 remembered / forgotten。
type SubmitResultRequest struct {
	Result string `json:"result" enum:"remembered,forgotten"`
}

// ---- 获取一轮结果（api/reviews.md §5）----

// SessionResultItem 是逐词结果项。
type SessionResultItem struct {
	Word   string `json:"word"`
	Result string `json:"result"`
}

// GetSessionResponse 是一轮复习的汇总与逐词结果；completedAt 未完成时为
// null（HTTP API 设计规范 §5.4：明确存在「未设置」语义时使用 null）。
type GetSessionResponse struct {
	SessionID   string              `json:"sessionId"`
	Status      string              `json:"status"`
	Total       int                 `json:"total"`
	Remembered  int                 `json:"remembered"`
	Forgotten   int                 `json:"forgotten"`
	CompletedAt *time.Time          `json:"completedAt"`
	Items       []SessionResultItem `json:"items" nullable:"false"`
}

// NewGetSessionResponse 转换获取一轮结果的数据。
func NewGetSessionResponse(r *service.GetSessionResult) GetSessionResponse {
	items := make([]SessionResultItem, 0, len(r.Items))
	for _, item := range r.Items {
		items = append(items, SessionResultItem{
			Word:   item.Headword,
			Result: string(item.Result),
		})
	}
	return GetSessionResponse{
		SessionID:   r.SessionID.String(),
		Status:      string(r.Status),
		Total:       r.Total,
		Remembered:  r.Remembered,
		Forgotten:   r.Forgotten,
		CompletedAt: r.CompletedAt,
		Items:       items,
	}
}
