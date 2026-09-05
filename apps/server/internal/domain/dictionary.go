// Package domain 承载不依赖外部资源的领域模型与纯业务规则
// （docs/backend/structure.md §4.1）。结构体不携带 bun struct tag /
// bun.BaseModel，方法只操作自身字段；实体工厂生成 UUID v7 并以 UTC
// 记录时间戳，同一用例需共享的 now 由 Service 显式传入。
package domain

import (
	"cmp"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Meaning 是词典释义的值对象，其 JSON 形态即 dictionary_entries 的
// raw_meanings / review_meanings 与 user_words 的 custom_review_meaning
// 列的 JSONB 结构（docs/dictionary/data-model.md §4）。
type Meaning struct {
	Pos          string   `json:"pos"`
	Translations []string `json:"translations"`
}

// MeaningSource 标记 effective review meaning 取自三层取值中的哪一层
// （docs/dictionary/data-model.md §7）。三层皆空时为零值 ""。
type MeaningSource string

const (
	MeaningSourceCustom MeaningSource = "custom" // user_words.custom_review_meaning
	MeaningSourceReview MeaningSource = "review" // dictionary_entries.review_meanings
	MeaningSourceRaw    MeaningSource = "raw"    // dictionary_entries.raw_meanings
)

// EffectiveReviewMeaning 按「custom ?? review ?? raw」三层取复习释义
// （docs/dictionary/data-model.md §7），空切片视为未设置。
// 三层皆无释义时返回 (nil, "")。
func EffectiveReviewMeaning(custom, review, raw []Meaning) ([]Meaning, MeaningSource) {
	switch {
	case len(custom) > 0:
		return custom, MeaningSourceCustom
	case len(review) > 0:
		return review, MeaningSourceReview
	case len(raw) > 0:
		return raw, MeaningSourceRaw
	default:
		return nil, ""
	}
}

// DictionaryEntry 对应 dictionary_entries：全局共享的客观词典数据，
// 任何用户都不应直接修改（docs/dictionary/data-model.md §1–§2）。
// 用户自定义只写 UserWord.CustomReviewMeaning（冻结决策 D006 / D007）。
type DictionaryEntry struct {
	ID             uuid.UUID
	Headword       string
	Lemma          string
	PhoneticUK     string
	PhoneticUS     string
	RawMeanings    []Meaning
	ReviewMeanings []Meaning
	Exchange       map[string]string // 词形变化表（ECDICT exchange；键为变化类型代码，"0" 为原形），lemma 解析依据 normalization.md §2
	Frequency      map[string]int    // 词频数据（frequency / bnc_frequency / coca_frequency，data-model.md §8）
	Tags           []string
	Source         string
	SourceVersion  string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// NewDictionaryEntry 创建词条：生成 UUID v7 主键与 UTC 时间戳。
// lemma 为空时按「无法确定 lemma 的输入保留原词」取 headword
// （docs/dictionary/normalization.md §3）。
func NewDictionaryEntry(headword, lemma string, now time.Time) (*DictionaryEntry, error) {
	if headword == "" {
		return nil, ErrHeadwordRequired
	}
	id, err := newUUIDv7()
	if err != nil {
		return nil, err
	}
	created := now.UTC()
	return &DictionaryEntry{
		ID:        id,
		Headword:  headword,
		Lemma:     cmp.Or(lemma, headword),
		CreatedAt: created,
		UpdatedAt: created,
	}, nil
}

// newUUIDv7 是各实体工厂共用的主键生成入口；主键统一为 UUID v7，
// 由 Domain 工厂在创建实体时生成（docs/backend/structure.md §2）。
func newUUIDv7() (uuid.UUID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, fmt.Errorf("domain: generate uuid v7: %w", err)
	}
	return id, nil
}
