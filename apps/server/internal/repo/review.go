package repo

import (
	"context"
	"hash/fnv"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/dongwlin/lexi-loop/apps/server/internal/domain"
	"github.com/dongwlin/lexi-loop/apps/server/internal/repo/internal/schema"
)

// activeSessionLockKey 是「全库至多一个 active session」流程串行化用的
// 事务级 advisory lock 键（structure.md §5.1、review/data-model.md §4、D010）。
// MVP 单用户全局一个键；V3 引入用户后按 user_id 派生。取值由固定字符串
// FNV-1a 派生，只要求部署期内稳定且唯一。
var activeSessionLockKey = func() int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte("lexi-loop:review:active-session"))
	return int64(h.Sum64())
}()

// ReviewItemWord 是逐词结果与所属词条 headword 的组合读模型
// （GetSession 响应的 items 需要「单词 + 结果」，
// docs/api/reviews.md §5）。
type ReviewItemWord struct {
	Item     *domain.ReviewItem
	Headword string
}

// ReviewRepo 查 / 写 review_sessions / review_items；并发所需的
// advisory lock、行锁与条件 UPDATE 封装在本 repo（structure.md §5）。
// 无状态：由 ReviewService 方法内以当前 bun.IDB（含 bun.Tx）构造。
type ReviewRepo struct {
	db bun.IDB
}

// NewReviewRepo 构造即用即弃的 ReviewRepo。
func NewReviewRepo(db bun.IDB) *ReviewRepo {
	return &ReviewRepo{db: db}
}

// AcquireActiveSessionLock 获取事务级 advisory lock，串行化
// 「放弃旧 active session → 创建新 session 与全部 items」的整个流程。
// 键为全局固定值；同一事务内重复获取幂等，事务结束自动释放。
func (r *ReviewRepo) AcquireActiveSessionLock(ctx context.Context) error {
	_, err := r.db.NewRaw("SELECT pg_advisory_xact_lock(?)", activeSessionLockKey).Exec(ctx)
	return normalizeError("acquire active session lock", err)
}

// LockActiveSession 查找并锁定当前唯一的 active session
// （SELECT ... FOR UPDATE）。不存在时返回 ErrNotFound。
func (r *ReviewRepo) LockActiveSession(ctx context.Context) (*domain.ReviewSession, error) {
	row := schema.ReviewSession{}
	err := r.db.NewSelect().
		Model(&row).
		Where("status = ?", string(domain.ReviewStatusActive)).
		For("UPDATE").
		Scan(ctx)
	if err != nil {
		return nil, normalizeError("lock active review session", err)
	}
	return toDomainSession(&row), nil
}

// GetSessionByID 按 ID 读取 session（GetSession 用例，不加锁）。
func (r *ReviewRepo) GetSessionByID(ctx context.Context, id uuid.UUID) (*domain.ReviewSession, error) {
	row := schema.ReviewSession{}
	err := r.db.NewSelect().
		Model(&row).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, normalizeError("find review session by id", err)
	}
	return toDomainSession(&row), nil
}

// LockSessionByID 按 ID 锁定 session（SELECT ... FOR UPDATE）。SubmitResult
// 先锁定 session 串行化同一轮的并发提交（structure.md §5.2 锁顺序第一步）。
// 不存在时返回 ErrNotFound。
func (r *ReviewRepo) LockSessionByID(ctx context.Context, id uuid.UUID) (*domain.ReviewSession, error) {
	row := schema.ReviewSession{}
	err := r.db.NewSelect().
		Model(&row).
		Where("id = ?", id).
		For("UPDATE").
		Scan(ctx)
	if err != nil {
		return nil, normalizeError("lock review session by id", err)
	}
	return toDomainSession(&row), nil
}

// InsertSession 写入新 session。全库已有 active session 时由部分唯一索引
// 返回 ErrAlreadyExists（D010 最终防线；正常流程已被 advisory lock 串行化）。
func (r *ReviewRepo) InsertSession(ctx context.Context, session *domain.ReviewSession) error {
	_, err := r.db.NewInsert().
		Model(toSchemaSession(session)).
		Exec(ctx)
	return normalizeError("insert review session", err)
}

// UpdateSession 回写 session 状态（Abandon / Complete 之后调用）。
// Session 行已由调用方锁定；行不存在时返回 ErrNotFound。
func (r *ReviewRepo) UpdateSession(ctx context.Context, session *domain.ReviewSession) error {
	res, err := r.db.NewUpdate().
		Model((*schema.ReviewSession)(nil)).
		Set("status = ?", string(session.Status)).
		Set("remembered_count = ?", session.RememberedCount).
		Set("forgotten_count = ?", session.ForgottenCount).
		Set("completed_at = ?", session.CompletedAt).
		Where("id = ?", session.ID).
		Exec(ctx)
	if err != nil {
		return normalizeError("update review session", err)
	}
	return requireAffected(res, "update review session")
}

// InsertItems 批量写入 pending 的复习 items（一个事务内一次性创建本轮
// 全部 items，structure.md §5.1）。
func (r *ReviewRepo) InsertItems(ctx context.Context, items []*domain.ReviewItem) error {
	if len(items) == 0 {
		return nil
	}
	rows := make([]schema.ReviewItem, len(items))
	for i, item := range items {
		rows[i] = *toSchemaItem(item)
	}
	_, err := r.db.NewInsert().
		Model(&rows).
		Exec(ctx)
	return normalizeError("insert review items", err)
}

// LockItem 按 session_id + item_id 锁定 item（SELECT ... FOR UPDATE）。
// 归属不匹配或不存在时返回 ErrNotFound（structure.md §5.2 锁顺序第二步）。
func (r *ReviewRepo) LockItem(ctx context.Context, sessionID, itemID uuid.UUID) (*domain.ReviewItem, error) {
	row := schema.ReviewItem{}
	err := r.db.NewSelect().
		Model(&row).
		Where("id = ?", itemID).
		Where("session_id = ?", sessionID).
		For("UPDATE").
		Scan(ctx)
	if err != nil {
		return nil, normalizeError("lock review item", err)
	}
	return toDomainItem(&row), nil
}

// UpdateItemIfPending 以条件 UPDATE 提交作答（docs/api/reviews.md §3）：
// 只有 result 仍为 pending 且归属匹配的 item 会被更新并计数一次，
// 返回其 user_word_id。未命中任何行（不存在、归属不匹配或已作答）时
// 返回 ErrNotFound；幂等 / 归属语义由调用方在锁定 item 后裁决。
func (r *ReviewRepo) UpdateItemIfPending(ctx context.Context, sessionID, itemID uuid.UUID, result domain.ReviewResult, now time.Time) (uuid.UUID, error) {
	var row struct {
		UserWordID uuid.UUID `bun:"user_word_id"`
	}
	err := r.db.NewUpdate().
		Model((*schema.ReviewItem)(nil)).
		Set("result = ?", string(result)).
		Set("reviewed_at = ?", now.UTC()).
		Where("id = ?", itemID).
		Where("session_id = ?", sessionID).
		Where("result = ?", string(domain.ReviewResultPending)).
		Returning("user_word_id").
		Scan(ctx, &row)
	if err != nil {
		return uuid.Nil, normalizeError("update review item if pending", err)
	}
	return row.UserWordID, nil
}

// CountResults 统计 session 内各状态的 item 数量：SubmitResult 用 pending
// 判断本轮是否已全部作答，Complete 的汇总用 remembered / forgotten
// （docs/api/reviews.md §5）。
func (r *ReviewRepo) CountResults(ctx context.Context, sessionID uuid.UUID) (remembered, forgotten, pending int, err error) {
	var rows []struct {
		Result string `bun:"result"`
		Count  int64  `bun:"count"`
	}
	err = r.db.NewSelect().
		Model((*schema.ReviewItem)(nil)).
		Column("result").
		ColumnExpr("count(*) AS count").
		Where("session_id = ?", sessionID).
		GroupExpr("result").
		Scan(ctx, &rows)
	if err != nil {
		return 0, 0, 0, normalizeError("count review item results", err)
	}
	for _, row := range rows {
		switch domain.ReviewResult(row.Result) {
		case domain.ReviewResultRemembered:
			remembered = int(row.Count)
		case domain.ReviewResultForgotten:
			forgotten = int(row.Count)
		case domain.ReviewResultPending:
			pending = int(row.Count)
		}
	}
	return remembered, forgotten, pending, nil
}

// ListSessionItems 按 position 顺序读取 session 的全部 items 并带出
// 所属词条 headword（GetSession 的逐词结果，docs/api/reviews.md §5）。
func (r *ReviewRepo) ListSessionItems(ctx context.Context, sessionID uuid.UUID) ([]ReviewItemWord, error) {
	rows := make([]schema.ReviewItemWithHeadword, 0)
	err := r.db.NewSelect().
		Model(&rows).
		ColumnExpr("ri.*").
		ColumnExpr("de.headword").
		Join("JOIN user_words AS uw ON uw.id = ri.user_word_id").
		Join("JOIN dictionary_entries AS de ON de.id = uw.dictionary_entry_id").
		Where("ri.session_id = ?", sessionID).
		OrderExpr("ri.position ASC").
		Scan(ctx)
	if err != nil {
		return nil, normalizeError("list review session items", err)
	}
	items := make([]ReviewItemWord, len(rows))
	for i := range rows {
		items[i] = ReviewItemWord{
			Item:     toDomainItem(&rows[i].ReviewItem),
			Headword: rows[i].Headword,
		}
	}
	return items, nil
}

// ---- schema ↔ domain 转换（只发生在 repo 内） ----

func toDomainSession(s *schema.ReviewSession) *domain.ReviewSession {
	return &domain.ReviewSession{
		ID:              s.ID,
		RequestedCount:  s.RequestedCount,
		TotalCount:      s.TotalCount,
		RememberedCount: s.RememberedCount,
		ForgottenCount:  s.ForgottenCount,
		Status:          domain.ReviewStatus(s.Status),
		StartedAt:       utcTime(s.StartedAt),
		CompletedAt:     utcTimePtr(s.CompletedAt),
		CreatedAt:       utcTime(s.CreatedAt),
	}
}

func toSchemaSession(d *domain.ReviewSession) *schema.ReviewSession {
	return &schema.ReviewSession{
		ID:              d.ID,
		RequestedCount:  d.RequestedCount,
		TotalCount:      d.TotalCount,
		RememberedCount: d.RememberedCount,
		ForgottenCount:  d.ForgottenCount,
		Status:          string(d.Status),
		StartedAt:       d.StartedAt,
		CompletedAt:     d.CompletedAt,
		CreatedAt:       d.CreatedAt,
	}
}

func toDomainItem(s *schema.ReviewItem) *domain.ReviewItem {
	return &domain.ReviewItem{
		ID:         s.ID,
		SessionID:  s.SessionID,
		UserWordID: s.UserWordID,
		Position:   s.Position,
		Result:     domain.ReviewResult(s.Result),
		CreatedAt:  utcTime(s.CreatedAt),
		ReviewedAt: utcTimePtr(s.ReviewedAt),
	}
}

func toSchemaItem(d *domain.ReviewItem) *schema.ReviewItem {
	return &schema.ReviewItem{
		ID:         d.ID,
		SessionID:  d.SessionID,
		UserWordID: d.UserWordID,
		Position:   d.Position,
		Result:     string(d.Result),
		CreatedAt:  d.CreatedAt,
		ReviewedAt: d.ReviewedAt,
	}
}
