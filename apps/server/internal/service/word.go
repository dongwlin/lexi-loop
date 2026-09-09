package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/dongwlin/lexi-loop/apps/server/internal/domain"
)

// Word 是生词库用例接口，供 Handler 注入实现或测试替身。
type Word interface {
	ImportWords(context.Context, ImportWordsRequest) (*ImportWordsResult, error)
	ListWords(context.Context, ListWordsRequest) (*ListWordsResult, error)
	GetWord(context.Context, uuid.UUID) (*WordItem, error)
	UpdateReviewMeaning(context.Context, uuid.UUID, UpdateReviewMeaningRequest) error
	DeleteWord(context.Context, uuid.UUID) error
}

// ---- 用例请求 / 结果类型（无 json 标签，JSON 契约由版本化 DTO 定义）----

// ImportWordItem 是批量导入的单项输入。
type ImportWordItem struct {
	Word string
	// Count 本次计入的遇词次数（首次导入为 1，重复导入累计）。
	Count int
}

// ImportWordsRequest 是批量导入用例的请求。
type ImportWordsRequest struct {
	Words []ImportWordItem
}

// ImportWordsResult 是批量导入用例的结果（api/words.md §2 响应字段）。
type ImportWordsResult struct {
	// Encounters 本次计入的遇词次数合计。
	Encounters int
	// Created 新建的 user_words 词条数。
	Created int
	// Updated 已存在、遇词次数被累计的词条数。
	Updated int
	// Items 逐词结果（与聚合后的输入单词一一对应），导入反馈的数据来源。
	Items []ImportWordOutcome
}

// ImportWordOutcome 是单个输入单词的导入结果。
type ImportWordOutcome struct {
	// Word 归一后的单词。
	Word string
	// Count 该词本次计入的遇词次数。
	Count int
	// Created 报告该词命中的 user_words 行是否本次新建（false = 已存在并累计，
	// 含软删除恢复）。
	Created bool
}

// ListWordsRequest 是生词库列表用例的请求（api/words.md §3）。
type ListWordsRequest struct {
	Page     int
	PageSize int
	Search   string
}

// ListWordsResult 是生词库列表用例的结果。
type ListWordsResult struct {
	Items []*WordItem
	// Total 为过滤后的总记录数（分页元数据由 Handler DTO 计算）。
	Total int64
}

// WordItem 是列表 / 详情共用的展示读模型：学习字段 + 词典字段 +
// 按「custom ?? review ?? raw」三层取值后的复习释义（D007）。
type WordItem struct {
	// UserWord 是学习状态来源；Entry 是词典字段来源（D006 两表分离）。
	UserWord *domain.UserWord
	Entry    *domain.DictionaryEntry
	// EffectiveMeaning / MeaningSource 为三层取值结果，三层皆空时为零值。
	EffectiveMeaning []domain.Meaning
	MeaningSource    domain.MeaningSource
	// Phonetic 为展示音标（单值字段，优先英式、缺失回退美式）。
	Phonetic string
	// MasteryScore / ReviewWeight 为动态计算的派生指标（D011 不落库，
	// 公式权威 docs/review/algorithm.md §3–§6），组装读模型时按当前时间
	// 计算；JSON 契约的舍入由 DTO 层处理。
	MasteryScore float64
	ReviewWeight float64
}

// UpdateReviewMeaningRequest 是更新复习释义用例的请求。
type UpdateReviewMeaningRequest struct {
	// CustomReviewMeaning 为要写入的自定义复习释义；nil 表示清除自定义、
	// 回退词典层默认复习释义（api/words.md §5）。
	CustomReviewMeaning []domain.Meaning
}
