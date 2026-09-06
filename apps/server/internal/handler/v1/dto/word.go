// Package dto 定义 /api/v1 的请求 / 响应 DTO 与转换函数：JSON 契约
// （camelCase 字段、空列表 []、空对象 {}）只由本包定义；Service 请求 /
// 结果类型不含 json 标签（docs/specs/backend/HTTP API 设计规范.md §1、§12，
// docs/backend/structure.md §4.3–§4.4）。
//
// 请求体的存在性 / 长度 / 取值约束以 huma 的 JSON Schema 标签声明
// （required / minLength / minimum / minItems / enum 等），替代 gin 的
// binding 标签；schema 生成与请求校验共用同一份标签。
package dto

import (
	"time"

	"github.com/dongwlin/lexi-loop/apps/server/internal/domain"
	"github.com/dongwlin/lexi-loop/apps/server/internal/service"
)

// NoData 是无额外数据端点的 data 形态（HTTP API 设计规范 §5.1：统一返回
// 空对象 {}）。
type NoData struct{}

// Meaning 是释义义项的 JSON 形态（dictionary/data-model.md §4，
// HTTP 契约与 JSONB 存储结构一致）。pos / translations 允许缺省。
type Meaning struct {
	Pos          string   `json:"pos" required:"false"`
	Translations []string `json:"translations" required:"false"`
}

// NewMeanings 把 service 层的释义转换为响应 DTO；空切片归一为 [] 而非 null
// （HTTP API 设计规范 §5）。
func NewMeanings(ms []domain.Meaning) []Meaning {
	out := make([]Meaning, 0, len(ms))
	for _, m := range ms {
		out = append(out, Meaning{Pos: m.Pos, Translations: m.Translations})
	}
	return out
}

// ToDomainMeanings 把请求 DTO 的释义转换为 service 层类型；nil 语义保持
// （表示清除自定义复习释义）。
func ToDomainMeanings(ms []Meaning) []domain.Meaning {
	if len(ms) == 0 {
		return nil
	}
	out := make([]domain.Meaning, 0, len(ms))
	for _, m := range ms {
		out = append(out, domain.Meaning{Pos: m.Pos, Translations: m.Translations})
	}
	return out
}

// ---- 导入（api/words.md §2）----

// ImportWordsRequest 是 POST /api/v1/words/import 的请求体。
type ImportWordsRequest struct {
	Words []ImportWordInput `json:"words" minItems:"1" nullable:"false"`
}

// ImportWordInput 是单个导入项：word 经服务端归一，count 为本次计入的
// 遇词次数。
type ImportWordInput struct {
	Word  string `json:"word" minLength:"1"`
	Count int    `json:"count" minimum:"1"`
}

// ImportWordsResponse 是导入用例的响应数据。
type ImportWordsResponse struct {
	Encounters int `json:"encounters"`
	Created    int `json:"created"`
	Updated    int `json:"updated"`
}

// NewImportWordsResponse 转换导入结果。
func NewImportWordsResponse(r *service.ImportWordsResult) ImportWordsResponse {
	return ImportWordsResponse{
		Encounters: r.Encounters,
		Created:    r.Created,
		Updated:    r.Updated,
	}
}

// ---- 列表与详情（api/words.md §3–§4）----

// ListWordsQuery 是 GET /api/v1/words 的查询参数；page / pageSize 缺省值
// 在 schema 中声明（1 / 20），越界值（<1 / >100）由 Handler 归一。
type ListWordsQuery struct {
	Page     int    `query:"page" default:"1"`
	PageSize int    `query:"pageSize" default:"20"`
	Search   string `query:"search"`
}

// WordListItem 是列表项（api/words.md §3 字段）。
type WordListItem struct {
	ID                     string     `json:"id"`
	Word                   string     `json:"word"`
	Phonetic               string     `json:"phonetic"`
	EffectiveReviewMeaning []Meaning  `json:"effectiveReviewMeaning" nullable:"false"`
	EncounterCount         int        `json:"encounterCount"`
	ReviewCount            int        `json:"reviewCount"`
	RememberCount          int        `json:"rememberCount"`
	ForgetCount            int        `json:"forgetCount"`
	CurrentStreak          int        `json:"currentStreak"`
	LastReviewedAt         *time.Time `json:"lastReviewedAt"`
}

// WordDetail 是单词详情（api/words.md §4 字段：列表字段 + definition 与
// meaningSource）。definition 为英文释义，MVP 数据源暂无该数据（V2 Enrich
// 后填充），恒为空串；meaningSource 标记三层取值命中的层。
type WordDetail struct {
	WordListItem
	Definition    string `json:"definition"`
	MeaningSource string `json:"meaningSource"`
}

// NewWordListItem 转换列表项。
func NewWordListItem(item *service.WordItem) WordListItem {
	return WordListItem{
		ID:                     item.UserWord.ID.String(),
		Word:                   item.Entry.Headword,
		Phonetic:               item.Phonetic,
		EffectiveReviewMeaning: NewMeanings(item.EffectiveMeaning),
		EncounterCount:         item.UserWord.EncounterCount,
		ReviewCount:            item.UserWord.ReviewCount,
		RememberCount:          item.UserWord.RememberCount,
		ForgetCount:            item.UserWord.ForgetCount,
		CurrentStreak:          item.UserWord.CurrentStreak,
		LastReviewedAt:         item.UserWord.LastReviewedAt,
	}
}

// NewWordDetail 转换单词详情。
func NewWordDetail(item *service.WordItem) WordDetail {
	return WordDetail{
		WordListItem:  NewWordListItem(item),
		MeaningSource: string(item.MeaningSource),
	}
}

// ListWordsResponse 是列表用例的响应数据（data.list + data.pagination）。
type ListWordsResponse struct {
	List       []WordListItem `json:"list" nullable:"false"`
	Pagination Pagination     `json:"pagination"`
}

// Pagination 是页码分页元数据（HTTP API 设计规范 §8.3）。
type Pagination struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"pageSize"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"totalPages"`
	HasMore    bool  `json:"hasMore"`
}

// NewListWordsResponse 组装列表响应：list 保证非 null（空列表为 []），
// totalPages = ceil(total / pageSize)，hasMore = page < totalPages。
func NewListWordsResponse(page, pageSize int, total int64, items []*service.WordItem) ListWordsResponse {
	list := make([]WordListItem, 0, len(items))
	for _, item := range items {
		list = append(list, NewWordListItem(item))
	}
	totalPages := (total + int64(pageSize) - 1) / int64(pageSize)
	return ListWordsResponse{
		List: list,
		Pagination: Pagination{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: int(totalPages),
			HasMore:    int64(page) < totalPages,
		},
	}
}

// ---- 更新复习释义（api/words.md §5）----

// UpdateReviewMeaningRequest 是 PATCH /api/v1/words/:id 的请求体。
// customReviewMeaning 可缺省或传 null，均表示清除自定义、回退词典层默认
// 复习释义。
type UpdateReviewMeaningRequest struct {
	CustomReviewMeaning []Meaning `json:"customReviewMeaning" required:"false"`
}
