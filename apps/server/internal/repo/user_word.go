package repo

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/dongwlin/lexi-loop/apps/server/internal/domain"
	"github.com/dongwlin/lexi-loop/apps/server/internal/repo/internal/schema"
)

// ListParams 是生词库列表的查询参数（docs/api/words.md §3）。
type ListParams struct {
	// Page 页码，从 1 开始；<1 按 1 处理。
	Page int
	// PageSize 每页条数；<1 按默认 20 处理（上限由 Handler / Service 校验）。
	PageSize int
	// Search 关键词，匹配 headword 或任一层释义；空串不过滤。
	Search string
}

// ListDefaultPageSize 是 Page / PageSize 未提供时的默认每页条数。
const ListDefaultPageSize = 20

// EncounterInput 是 ImportWords 一次批量导入的单项（docs/api/words.md §2）。
type EncounterInput struct {
	DictionaryEntryID uuid.UUID
	// Count 本次计入的遇词次数（首次导入为 1，重复导入累计）。
	Count int
}

// EncounterResult 是单项 upsert 的结果：Created 报告本次是否新建了
// user_words 行（对应 API 响应的 created / updated 统计）。
type EncounterResult struct {
	DictionaryEntryID uuid.UUID
	Created           bool
}

// UserWordRepo 查 / 写 user_words，含软删除（D004）与遇词累计 upsert。
// 无状态：由 Service 方法内以当前 bun.IDB（含 bun.Tx）构造。
type UserWordRepo struct {
	db bun.IDB
}

// NewUserWordRepo 构造即用即弃的 UserWordRepo。
func NewUserWordRepo(db bun.IDB) *UserWordRepo {
	return &UserWordRepo{db: db}
}

// FindByID 按 ID 查询学习行（无论是否已软删除；生词库视角的过滤由
// 调用方按用例决定）。
func (r *UserWordRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.UserWord, error) {
	row := schema.UserWord{}
	err := r.db.NewSelect().
		Model(&row).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, normalizeError("find user word by id", err)
	}
	return toDomainUserWord(&row), nil
}

// LockByID 按 ID 锁定学习行（SELECT ... FOR UPDATE）。SubmitResult 在
// 条件更新 item 之后锁定对应 user_word 再累计，锁顺序为
// review_session → review_item → user_word（structure.md §5.2）。
func (r *UserWordRepo) LockByID(ctx context.Context, id uuid.UUID) (*domain.UserWord, error) {
	row := schema.UserWord{}
	err := r.db.NewSelect().
		Model(&row).
		Where("id = ?", id).
		For("UPDATE").
		Scan(ctx)
	if err != nil {
		return nil, normalizeError("lock user word by id", err)
	}
	return toDomainUserWord(&row), nil
}

// List 分页查询未删除的生词（deleted_at IS NULL，docs/api/words.md §3），
// 按 created_at 倒序（新导入的词在前），返回当页与未过滤的总数。
// search 同时匹配 headword 与任一层释义（词典字段经 JOIN 过滤，
// 每个生词至多关联一个词条，JOIN 不产生重复行）。
func (r *UserWordRepo) List(ctx context.Context, params ListParams) ([]*domain.UserWord, int64, error) {
	page := params.Page
	if page < 1 {
		page = 1
	}
	pageSize := params.PageSize
	if pageSize < 1 {
		pageSize = ListDefaultPageSize
	}

	total, err := r.listQuery(params).
		Count(ctx)
	if err != nil {
		return nil, 0, normalizeError("count user words", err)
	}
	if total == 0 {
		return nil, 0, nil
	}

	rows := make([]schema.UserWord, 0, pageSize)
	err = r.listQuery(params).
		OrderExpr("uw.created_at DESC, uw.id DESC").
		Limit(pageSize).
		Offset((page-1)*pageSize).
		Scan(ctx, &rows)
	if err != nil {
		return nil, 0, normalizeError("list user words", err)
	}

	words := make([]*domain.UserWord, len(rows))
	for i := range rows {
		words[i] = toDomainUserWord(&rows[i])
	}
	return words, int64(total), nil
}

// listQuery 组装 List 的公共查询：JOIN 词典表 + 过滤软删除 + search 条件。
func (r *UserWordRepo) listQuery(params ListParams) *bun.SelectQuery {
	q := r.db.NewSelect().
		Model((*schema.UserWord)(nil)).
		ColumnExpr("uw.*").
		Join("LEFT JOIN dictionary_entries AS de ON de.id = uw.dictionary_entry_id").
		Where("uw.deleted_at IS NULL")
	if params.Search != "" {
		pattern := "%" + escapeLike(params.Search) + "%"
		q = q.Where(
			"(de.headword ILIKE ? OR de.raw_meanings::text ILIKE ? OR de.review_meanings::text ILIKE ? OR uw.custom_review_meaning::text ILIKE ?)",
			pattern, pattern, pattern, pattern,
		)
	}
	return q
}

// ListActive 查询全部未删除的生词（加权抽样的候选集，
// docs/api/reviews.md §2）。数量截断 min(count, available) 由 Service 完成。
func (r *UserWordRepo) ListActive(ctx context.Context) ([]*domain.UserWord, error) {
	rows := make([]schema.UserWord, 0)
	err := r.db.NewSelect().
		Model(&rows).
		Where("deleted_at IS NULL").
		OrderExpr("id ASC").
		Scan(ctx)
	if err != nil {
		return nil, normalizeError("list active user words", err)
	}
	words := make([]*domain.UserWord, len(rows))
	for i := range rows {
		words[i] = toDomainUserWord(&rows[i])
	}
	return words, nil
}

// UpsertEncounters 批量累计遇词次数（docs/api/words.md §2）：
// INSERT ... ON CONFLICT (dictionary_entry_id) DO UPDATE，重复导入累计
// encounter_count 并恢复软删除（deleted_at 置 NULL），不新建行、不丢历史。
// 返回逐项的 created / updated 结果；调用方须保证同一批次内
// dictionary_entry_id 不重复（PostgreSQL 不允许同语句二次冲突）。
func (r *UserWordRepo) UpsertEncounters(ctx context.Context, words []*domain.UserWord) ([]EncounterResult, error) {
	if len(words) == 0 {
		return nil, nil
	}
	rows := make([]schema.UserWord, len(words))
	for i, w := range words {
		rows[i] = *toSchemaUserWord(w)
	}

	var upserted []struct {
		DictionaryEntryID uuid.UUID `bun:"dictionary_entry_id"`
		Created           bool      `bun:"created"`
	}
	// xmax = 0 仅对本次插入的行成立，冲突更新的行 xmax 为当前事务 ID，
	// 以此区分 created / updated。
	err := r.db.NewInsert().
		Model(&rows).
		On("CONFLICT (dictionary_entry_id) DO UPDATE").
		Set("encounter_count = uw.encounter_count + EXCLUDED.encounter_count").
		Set("deleted_at = NULL").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("dictionary_entry_id, (xmax = 0) AS created").
		Scan(ctx, &upserted)
	if err != nil {
		return nil, normalizeError("upsert user word encounters", err)
	}

	results := make([]EncounterResult, len(upserted))
	for i, row := range upserted {
		results[i] = EncounterResult{DictionaryEntryID: row.DictionaryEntryID, Created: row.Created}
	}
	return results, nil
}

// UpdateCustomReviewMeaning 更新用户自定义复习释义（docs/api/words.md §5，
// D007：用户自定义只写 user_words）：meanings 为 nil 表示清除自定义
// （custom_review_meaning 置 NULL）。已软删除或不存在的行返回 ErrNotFound。
func (r *UserWordRepo) UpdateCustomReviewMeaning(ctx context.Context, id uuid.UUID, meanings []domain.Meaning, now time.Time) error {
	var value any
	if len(meanings) > 0 {
		value = schema.JSONB[[]schema.Meaning]{Val: toSchemaMeanings(meanings)}
	}
	res, err := r.db.NewUpdate().
		Model((*schema.UserWord)(nil)).
		Set("custom_review_meaning = ?", value).
		Set("updated_at = ?", now.UTC()).
		Where("id = ?", id).
		Where("deleted_at IS NULL").
		Exec(ctx)
	if err != nil {
		return normalizeError("update custom review meaning", err)
	}
	return requireAffected(res, "update custom review meaning")
}

// SoftDelete 软删除生词：置 deleted_at（D004），不物理删行、保留复习历史。
// 已删除或不存在的行返回 ErrNotFound。
func (r *UserWordRepo) SoftDelete(ctx context.Context, id uuid.UUID, now time.Time) error {
	res, err := r.db.NewUpdate().
		Model((*schema.UserWord)(nil)).
		Set("deleted_at = ?", now.UTC()).
		Set("updated_at = ?", now.UTC()).
		Where("id = ?", id).
		Where("deleted_at IS NULL").
		Exec(ctx)
	if err != nil {
		return normalizeError("soft delete user word", err)
	}
	return requireAffected(res, "soft delete user word")
}

// UpdateReviewCounters 写回词级累计字段（ApplyReview 之后调用，
// docs/review/data-model.md §5）。目标行不存在时返回 ErrNotFound。
func (r *UserWordRepo) UpdateReviewCounters(ctx context.Context, w *domain.UserWord) error {
	res, err := r.db.NewUpdate().
		Model((*schema.UserWord)(nil)).
		Set("review_count = ?", w.ReviewCount).
		Set("remember_count = ?", w.RememberCount).
		Set("forget_count = ?", w.ForgetCount).
		Set("current_streak = ?", w.CurrentStreak).
		Set("last_reviewed_at = ?", w.LastReviewedAt).
		Set("updated_at = ?", w.UpdatedAt).
		Where("id = ?", w.ID).
		Exec(ctx)
	if err != nil {
		return normalizeError("update review counters", err)
	}
	return requireAffected(res, "update review counters")
}

// requireAffected 把「未命中任何行」归一化为 ErrNotFound（条件 UPDATE
// 的业务条件不满足或行不存在，docs/specs/backend/Go 单体应用架构规范.md §4.5）。
func requireAffected(res sql.Result, op string) error {
	affected, err := res.RowsAffected()
	if err != nil {
		return normalizeError(op, err)
	}
	if affected == 0 {
		return fmt.Errorf("repo: %s: %w", op, ErrNotFound)
	}
	return nil
}

// ---- schema ↔ domain 转换（只发生在 repo 内） ----

func toDomainUserWord(s *schema.UserWord) *domain.UserWord {
	return &domain.UserWord{
		ID:                  s.ID,
		DictionaryEntryID:   s.DictionaryEntryID,
		EncounterCount:      s.EncounterCount,
		ReviewCount:         s.ReviewCount,
		RememberCount:       s.RememberCount,
		ForgetCount:         s.ForgetCount,
		CurrentStreak:       s.CurrentStreak,
		LastReviewedAt:      utcTimePtr(s.LastReviewedAt),
		CustomReviewMeaning: toDomainMeanings(s.CustomReviewMeaning.Val),
		DeletedAt:           utcTimePtr(s.DeletedAt),
		CreatedAt:           utcTime(s.CreatedAt),
		UpdatedAt:           utcTime(s.UpdatedAt),
	}
}

func toSchemaUserWord(d *domain.UserWord) *schema.UserWord {
	return &schema.UserWord{
		ID:                  d.ID,
		DictionaryEntryID:   d.DictionaryEntryID,
		EncounterCount:      d.EncounterCount,
		ReviewCount:         d.ReviewCount,
		RememberCount:       d.RememberCount,
		ForgetCount:         d.ForgetCount,
		CurrentStreak:       d.CurrentStreak,
		LastReviewedAt:      utcTimePtr(d.LastReviewedAt),
		CustomReviewMeaning: schema.JSONB[[]schema.Meaning]{Val: toSchemaMeanings(d.CustomReviewMeaning)},
		DeletedAt:           utcTimePtr(d.DeletedAt),
		CreatedAt:           d.CreatedAt,
		UpdatedAt:           d.UpdatedAt,
	}
}
