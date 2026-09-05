package domain

import (
	"time"

	"github.com/google/uuid"
)

// ReviewStatus 是 review_sessions.status 的类型化状态
// （docs/review/data-model.md §4）。状态值供 Repo / Handler 等多层引用，
// 统一使用本常量，不散落字符串字面量。
type ReviewStatus string

const (
	ReviewStatusActive    ReviewStatus = "active"
	ReviewStatusCompleted ReviewStatus = "completed"
	ReviewStatusAbandoned ReviewStatus = "abandoned"
)

// ReviewResult 是 review_items.result 的类型化取值：pending 表示未作答，
// remembered / forgotten 是一次有效作答的结果（docs/review/data-model.md §2）。
type ReviewResult string

const (
	ReviewResultPending    ReviewResult = "pending"
	ReviewResultRemembered ReviewResult = "remembered"
	ReviewResultForgotten  ReviewResult = "forgotten"
)

// isAnswer 报告该结果是否为一次有效作答（pending 不是作答）。
func (r ReviewResult) isAnswer() bool {
	return r == ReviewResultRemembered || r == ReviewResultForgotten
}

// ReviewSession 对应 review_sessions：一轮复习（docs/review/data-model.md §1）。
// 冻结决策 D010：同一时间至多一个 active session；放弃旧轮次与创建新轮次
// 的串行化由 Service 经 Repo 的 advisory lock 完成，Domain 只固化状态机。
type ReviewSession struct {
	ID              uuid.UUID
	RequestedCount  int
	TotalCount      int
	RememberedCount int
	ForgottenCount  int
	Status          ReviewStatus
	StartedAt       time.Time
	CompletedAt     *time.Time
	CreatedAt       time.Time
}

// NewReviewSession 创建 active session：生成 UUID v7 主键与 UTC 时间戳。
// D009 截断（total = min(count, available)）由 Service 完成，Domain 校验
// requested ≥ 1、total ≥ 1 且 total ≤ requested。
func NewReviewSession(requestedCount, totalCount int, now time.Time) (*ReviewSession, error) {
	if requestedCount < 1 || totalCount < 1 || totalCount > requestedCount {
		return nil, ErrInvalidSessionCounts
	}
	id, err := newUUIDv7()
	if err != nil {
		return nil, err
	}
	started := now.UTC()
	return &ReviewSession{
		ID:             id,
		RequestedCount: requestedCount,
		TotalCount:     totalCount,
		Status:         ReviewStatusActive,
		StartedAt:      started,
		CreatedAt:      started,
	}, nil
}

// Complete 完成 session：回写 remembered_count / forgotten_count 与
// completed_at（docs/review/data-model.md §4）。仅 active 可完成，且
// remembered + forgotten 必须等于 total_count（全部 item 已作答）。
func (s *ReviewSession) Complete(rememberedCount, forgottenCount int, now time.Time) error {
	if s.Status != ReviewStatusActive {
		return ErrSessionNotActive
	}
	if rememberedCount < 0 || forgottenCount < 0 || rememberedCount+forgottenCount != s.TotalCount {
		return ErrSessionTotalsMismatch
	}
	completed := now.UTC()
	s.Status = ReviewStatusCompleted
	s.RememberedCount = rememberedCount
	s.ForgottenCount = forgottenCount
	s.CompletedAt = &completed
	return nil
}

// Abandon 放弃 session（docs/review/data-model.md §4）。仅 active 可放弃；
// abandoned 不是完成，completed_at 保持为空。
func (s *ReviewSession) Abandon(now time.Time) error {
	if s.Status != ReviewStatusActive {
		return ErrSessionNotActive
	}
	s.Status = ReviewStatusAbandoned
	return nil
}

// ReviewItem 对应 review_items：一轮里逐次的复习记录
// （docs/review/data-model.md §2）。同一 session 内 position 唯一、同一
// user_word 至多出现一次，由数据库唯一约束保证。
type ReviewItem struct {
	ID         uuid.UUID
	SessionID  uuid.UUID
	UserWordID uuid.UUID
	Position   int
	Result     ReviewResult
	CreatedAt  time.Time
	ReviewedAt *time.Time
}

// NewReviewItem 创建 pending 的复习 item：生成 UUID v7 主键与 UTC 时间戳。
func NewReviewItem(sessionID, userWordID uuid.UUID, position int, now time.Time) (*ReviewItem, error) {
	if position < 0 {
		return nil, ErrInvalidPosition
	}
	id, err := newUUIDv7()
	if err != nil {
		return nil, err
	}
	return &ReviewItem{
		ID:         id,
		SessionID:  sessionID,
		UserWordID: userWordID,
		Position:   position,
		Result:     ReviewResultPending,
		CreatedAt:  now.UTC(),
	}, nil
}

// Submit 提交作答：pending → remembered / forgotten 只能发生一次
// （docs/review/data-model.md §4）。并发的幂等短路由 Repo 的条件 UPDATE
// 保证（docs/backend/structure.md §5.2），本方法固化实体状态机不变式。
func (i *ReviewItem) Submit(result ReviewResult, now time.Time) error {
	if !result.isAnswer() {
		return ErrInvalidReviewResult
	}
	if i.Result != ReviewResultPending {
		return ErrItemAlreadyAnswered
	}
	reviewed := now.UTC()
	i.Result = result
	i.ReviewedAt = &reviewed
	return nil
}
