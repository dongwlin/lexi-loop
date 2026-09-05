// Package repo 提供无状态数据访问适配器：构造即用即弃、仅依赖 bun.IDB，
// 返回 domain 类型，schema ↔ domain 转换只在 repo 内部完成
// （docs/backend/structure.md §4.2）。数据访问错误按 SQLSTATE 归一化为
// errors.go 中的可识别错误；条件 UPDATE、行锁与 advisory lock 封装在
// repo 内，Service 只表达业务意图（docs/backend/structure.md §5）。
package repo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/dongwlin/lexi-loop/apps/server/internal/domain"
	"github.com/dongwlin/lexi-loop/apps/server/internal/repo/internal/schema"
)

// DictionaryRepo 查 / 写 dictionary_entries（structure.md §4.2）。
// 无状态：由 Service / Importer 方法内以当前 bun.IDB（含 bun.Tx）构造。
type DictionaryRepo struct {
	db bun.IDB
}

// NewDictionaryRepo 构造即用即弃的 DictionaryRepo。
func NewDictionaryRepo(db bun.IDB) *DictionaryRepo {
	return &DictionaryRepo{db: db}
}

// FindByHeadword 按 headword 精确查询词条（Lookup 的本地词典命中路径）。
func (r *DictionaryRepo) FindByHeadword(ctx context.Context, headword string) (*domain.DictionaryEntry, error) {
	row := schema.DictionaryEntry{}
	err := r.db.NewSelect().
		Model(&row).
		Where("headword = ?", headword).
		Scan(ctx)
	if err != nil {
		return nil, normalizeError("find dictionary entry by headword", err)
	}
	return toDomainEntry(&row), nil
}

// FindByID 按 ID 查询完整词条（GetWord 等需要全部词典字段的场景）。
func (r *DictionaryRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.DictionaryEntry, error) {
	row := schema.DictionaryEntry{}
	err := r.db.NewSelect().
		Model(&row).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, normalizeError("find dictionary entry by id", err)
	}
	return toDomainEntry(&row), nil
}

// FindByIDs 批量查询词条，返回命中的词条（缺失的 ID 不在结果中）。
// 供 Service 在抽样 / 列表后按词条 ID 批量取展示字段，避免逐条查询。
func (r *DictionaryRepo) FindByIDs(ctx context.Context, ids []uuid.UUID) ([]*domain.DictionaryEntry, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows := make([]schema.DictionaryEntry, 0)
	err := r.db.NewSelect().
		Model(&rows).
		Where("id IN (?)", bun.In(ids)).
		OrderExpr("id ASC").
		Scan(ctx)
	if err != nil {
		return nil, normalizeError("find dictionary entries by ids", err)
	}
	entries := make([]*domain.DictionaryEntry, len(rows))
	for i := range rows {
		entries[i] = toDomainEntry(&rows[i])
	}
	return entries, nil
}

// FindByLemma 按 lemma 查询词条候选（structure.md §4.1：lemma 候选查询
// 交给 Dictionary Repo；能否归并由 DictionaryService 按
// 「不确定不归并」裁决，docs/dictionary/normalization.md §3）。
func (r *DictionaryRepo) FindByLemma(ctx context.Context, lemma string) ([]*domain.DictionaryEntry, error) {
	rows := make([]schema.DictionaryEntry, 0)
	err := r.db.NewSelect().
		Model(&rows).
		Where("lemma = ?", lemma).
		OrderExpr("id ASC").
		Scan(ctx)
	if err != nil {
		return nil, normalizeError("find dictionary entries by lemma", err)
	}
	entries := make([]*domain.DictionaryEntry, len(rows))
	for i := range rows {
		entries[i] = toDomainEntry(&rows[i])
	}
	return entries, nil
}

// Insert 写入一个新词条。headword 唯一约束冲突时返回 ErrAlreadyExists
// （并发创建同词由 Service 按用例处理）。
func (r *DictionaryRepo) Insert(ctx context.Context, entry *domain.DictionaryEntry) error {
	_, err := r.db.NewInsert().
		Model(toSchemaEntry(entry)).
		Exec(ctx)
	return normalizeError("insert dictionary entry", err)
}

// UpsertBatch 批量写入词条：headword 冲突时更新 ECDICT 词典字段
// （重跑导入刷新同一来源数据、不产生重复行，structure.md §5.3）。
// review_meanings 是词典层的展示派生数据（可被后续整理流程单独设置），
// 仅在本次写入提供值时才覆盖，否则保留原值。
func (r *DictionaryRepo) UpsertBatch(ctx context.Context, entries []*domain.DictionaryEntry) error {
	if len(entries) == 0 {
		return nil
	}
	rows := make([]schema.DictionaryEntry, len(entries))
	for i, entry := range entries {
		rows[i] = *toSchemaEntry(entry)
	}
	_, err := r.db.NewInsert().
		Model(&rows).
		On("CONFLICT (headword) DO UPDATE").
		Set("lemma = EXCLUDED.lemma").
		Set("phonetic_uk = EXCLUDED.phonetic_uk").
		Set("phonetic_us = EXCLUDED.phonetic_us").
		Set("raw_meanings = EXCLUDED.raw_meanings").
		Set("review_meanings = COALESCE(EXCLUDED.review_meanings, de.review_meanings)").
		Set("exchange = EXCLUDED.exchange").
		Set("frequency = EXCLUDED.frequency").
		Set("tags = EXCLUDED.tags").
		Set("source = EXCLUDED.source").
		Set("source_version = EXCLUDED.source_version").
		Set("updated_at = EXCLUDED.updated_at").
		Exec(ctx)
	return normalizeError("upsert dictionary entries", err)
}

// ---- schema ↔ domain 转换（只发生在 repo 内） ----

func toDomainEntry(s *schema.DictionaryEntry) *domain.DictionaryEntry {
	return &domain.DictionaryEntry{
		ID:             s.ID,
		Headword:       s.Headword,
		Lemma:          s.Lemma,
		PhoneticUK:     s.PhoneticUK,
		PhoneticUS:     s.PhoneticUS,
		RawMeanings:    toDomainMeanings(s.RawMeanings.Val),
		ReviewMeanings: toDomainMeanings(s.ReviewMeanings.Val),
		Exchange:       nonEmptyMap(s.Exchange.Val),
		Frequency:      nonEmptyMap(s.Frequency.Val),
		Tags:           nonEmptySlice(s.Tags.Val),
		Source:         s.Source,
		SourceVersion:  s.SourceVersion,
		CreatedAt:      utcTime(s.CreatedAt),
		UpdatedAt:      utcTime(s.UpdatedAt),
	}
}

func toSchemaEntry(d *domain.DictionaryEntry) *schema.DictionaryEntry {
	return &schema.DictionaryEntry{
		ID:             d.ID,
		Headword:       d.Headword,
		Lemma:          d.Lemma,
		PhoneticUK:     d.PhoneticUK,
		PhoneticUS:     d.PhoneticUS,
		RawMeanings:    schema.JSONB[[]schema.Meaning]{Val: toSchemaMeanings(d.RawMeanings)},
		ReviewMeanings: schema.JSONB[[]schema.Meaning]{Val: toSchemaMeanings(d.ReviewMeanings)},
		Exchange:       schema.JSONB[map[string]string]{Val: d.Exchange},
		Frequency:      schema.JSONB[map[string]int]{Val: d.Frequency},
		Tags:           schema.JSONB[[]string]{Val: d.Tags},
		Source:         d.Source,
		SourceVersion:  d.SourceVersion,
		CreatedAt:      d.CreatedAt,
		UpdatedAt:      d.UpdatedAt,
	}
}

// toDomainMeanings / toSchemaMeanings 转换释义；空切片与 nil 视为同一
// 「未设置」语义（docs/dictionary/data-model.md §7）。
func toDomainMeanings(ms []schema.Meaning) []domain.Meaning {
	if len(ms) == 0 {
		return nil
	}
	out := make([]domain.Meaning, len(ms))
	for i, m := range ms {
		out[i] = domain.Meaning{Pos: m.Pos, Translations: m.Translations}
	}
	return out
}

func toSchemaMeanings(ms []domain.Meaning) []schema.Meaning {
	if len(ms) == 0 {
		return nil
	}
	out := make([]schema.Meaning, len(ms))
	for i, m := range ms {
		out[i] = schema.Meaning{Pos: m.Pos, Translations: m.Translations}
	}
	return out
}

// nonEmptyMap / nonEmptySlice 把空容器归一为 nil，保持 domain 侧
// 「未设置」的语义一致。
func nonEmptyMap[M ~map[K]V, K comparable, V any](m M) M {
	if len(m) == 0 {
		return nil
	}
	return m
}

func nonEmptySlice[S ~[]E, E any](s S) S {
	if len(s) == 0 {
		return nil
	}
	return s
}

// utcTime / utcTimePtr 把从数据库读出的时间归一为 UTC（domain 约定以
// UTC 记录时间戳），指针版本保持 nil 语义。
func utcTime(t time.Time) time.Time {
	return t.UTC()
}

func utcTimePtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	v := t.UTC()
	return &v
}
