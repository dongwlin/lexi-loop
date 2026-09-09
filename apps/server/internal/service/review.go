package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/dongwlin/lexi-loop/apps/server/internal/domain"
)

// Review 是复习用例接口；事务与抽样细节由实现包负责。
type Review interface {
	StartSession(context.Context, StartSessionRequest) (*StartSessionResult, error)
	SubmitResult(context.Context, SubmitResultRequest) error
	AbandonSession(context.Context, AbandonSessionRequest) error
	GetSession(context.Context, GetSessionRequest) (*GetSessionResult, error)
}

// ---- 用例请求 / 结果类型（无 json 标签，JSON 契约由版本化 DTO 定义）----

// StartSessionRequest 是开始一轮复习用例的请求。
type StartSessionRequest struct {
	Count int
}

// StartSessionItem 是本轮抽中单词的展示读模型（api/reviews.md §2 items）。
type StartSessionItem struct {
	ItemID           uuid.UUID
	Headword         string
	Phonetic         string
	EffectiveMeaning []domain.Meaning
	MeaningSource    domain.MeaningSource
}

// StartSessionResult 是开始一轮复习用例的结果。
type StartSessionResult struct {
	SessionID      uuid.UUID
	RequestedCount int
	TotalCount     int
	Items          []StartSessionItem
}

// SubmitResultRequest 是提交单词复习结果用例的请求。
type SubmitResultRequest struct {
	SessionID uuid.UUID
	ItemID    uuid.UUID
	// Result 必须是 remembered / forgotten 的有效作答。
	Result domain.ReviewResult
}

// GetSessionRequest 是获取一轮复习结果用例的请求。
type GetSessionRequest struct {
	SessionID uuid.UUID
}

// AbandonSessionRequest 是放弃一轮复习用例的请求。
type AbandonSessionRequest struct {
	SessionID uuid.UUID
}

// GetSessionItem 是逐词结果读模型（api/reviews.md §5 items）。
type GetSessionItem struct {
	Headword string
	Result   domain.ReviewResult
}

// GetSessionResult 是获取一轮复习结果用例的结果。
type GetSessionResult struct {
	SessionID   uuid.UUID
	Status      domain.ReviewStatus
	Total       int
	Remembered  int
	Forgotten   int
	CompletedAt *time.Time
	Items       []GetSessionItem
}
