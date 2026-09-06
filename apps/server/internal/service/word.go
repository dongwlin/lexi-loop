package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/dongwlin/lexi-loop/apps/server/internal/apperr"
	"github.com/dongwlin/lexi-loop/apps/server/internal/domain"
	"github.com/dongwlin/lexi-loop/apps/server/internal/repo"
)

// Word 提供生词库用例（docs/api/words.md）：批量导入（累计遇词次数）、
// 分页列表、详情、更新自定义复习释义与软删除。词典解析委托
// DictionaryService，用户自定义只写 user_words（D007）。
type Word struct {
	db         *bun.DB
	dictionary *Dictionary
}

// NewWord 构造 WordService。
func NewWord(db *bun.DB, dictionary *Dictionary) *Word {
	return &Word{db: db, dictionary: dictionary}
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

// ImportWords 批量导入生词（api/words.md §2）：单个请求一个事务；
// 先在内存中按归一结果校验并聚合计数（前端已聚合，服务端幂等地再做一次；
// UpsertEncounters 要求同一批次内 dictionary_entry_id 不重复），事务内逐词
// 解析词条（不同输入可能归一到同一词条，再按词条聚合）后一次性
// ON CONFLICT 累计遇词次数并恢复软删除（D004）。任一项失败整批回滚。
func (s *Word) ImportWords(ctx context.Context, req ImportWordsRequest) (*ImportWordsResult, error) {
	counts := make(map[string]int, len(req.Words))
	order := make([]string, 0, len(req.Words))
	result := &ImportWordsResult{}
	for _, item := range req.Words {
		if item.Count < 1 {
			return nil, apperr.New(apperr.InvalidArgument, apperr.CodeValidationFailed, "word count must be positive", nil)
		}
		word := domain.NormalizeWord(item.Word)
		if word == "" {
			return nil, apperr.New(apperr.InvalidArgument, apperr.CodeValidationFailed, "word is required", nil)
		}
		if _, seen := counts[word]; !seen {
			order = append(order, word)
		}
		counts[word] += item.Count
		result.Encounters += item.Count
	}
	if len(order) == 0 {
		return nil, apperr.New(apperr.InvalidArgument, apperr.CodeValidationFailed, "words is required", nil)
	}

	// 词条解析使用 headword 唯一约束处理并发（structure.md §5.3）；整批
	// 事务可重放，冲突时由 runReplayableTx 整体重试后重查。
	err := runReplayableTx(ctx, s.db, func(ctx context.Context, tx bun.Tx) error {
		entryCounts := make(map[uuid.UUID]int, len(order))
		entryOrder := make([]uuid.UUID, 0, len(order))
		wordEntries := make(map[string]uuid.UUID, len(order))
		for _, word := range order {
			entry, err := s.dictionary.lookup(ctx, tx, word)
			if err != nil {
				return err
			}
			if _, seen := entryCounts[entry.ID]; !seen {
				entryOrder = append(entryOrder, entry.ID)
			}
			entryCounts[entry.ID] += counts[word]
			wordEntries[word] = entry.ID
		}

		now := time.Now()
		words := make([]*domain.UserWord, len(entryOrder))
		for i, entryID := range entryOrder {
			w, err := domain.NewUserWord(entryID, entryCounts[entryID], now)
			if err != nil {
				return err
			}
			words[i] = w
		}
		results, err := repo.NewUserWordRepo(tx).UpsertEncounters(ctx, words)
		if err != nil {
			return err
		}
		var created, updated int
		entryCreated := make(map[uuid.UUID]bool, len(results))
		for _, r := range results {
			entryCreated[r.DictionaryEntryID] = r.Created
			if r.Created {
				created++
			} else {
				updated++
			}
		}
		result.Created, result.Updated = created, updated

		// 逐词结果按输入顺序输出，单词粒度（api/words.md §2）：命中词条
		// 本次新建则该词记 created（归一到同一新建词条的多个词形都记
		// created），已存在 / 恢复则记 updated；与行粒度的聚合统计计量
		// 单位不同。
		result.Items = make([]ImportWordOutcome, 0, len(order))
		for _, word := range order {
			result.Items = append(result.Items, ImportWordOutcome{
				Word:    word,
				Count:   counts[word],
				Created: entryCreated[wordEntries[word]],
			})
		}
		return nil
	})
	if err != nil {
		return nil, internalError(err)
	}
	return result, nil
}

// ListWords 分页查询生词库（api/words.md §3）：过滤软删除，search 同时
// 匹配单词与各层释义；新导入的词在前。
func (s *Word) ListWords(ctx context.Context, req ListWordsRequest) (*ListWordsResult, error) {
	words, total, err := repo.NewUserWordRepo(s.db).List(ctx, repo.ListParams{
		Page:     req.Page,
		PageSize: req.PageSize,
		Search:   req.Search,
	})
	if err != nil {
		return nil, apperr.Internal(err)
	}

	items := make([]*WordItem, 0, len(words))
	if len(words) == 0 {
		return &ListWordsResult{Items: items, Total: total}, nil
	}

	entries, err := s.findEntries(ctx, words)
	if err != nil {
		return nil, err
	}
	for _, w := range words {
		items = append(items, newWordItem(w, entries[w.DictionaryEntryID]))
	}
	return &ListWordsResult{Items: items, Total: total}, nil
}

// GetWord 查询单词详情（api/words.md §4）。软删除的词从生词库视角已
// 不存在（api/words.md §3：被删除的词不再出现），与列表口径一致返回
// 404；复习历史不受影响（review_items 仍指向保留的行）。
func (s *Word) GetWord(ctx context.Context, id uuid.UUID) (*WordItem, error) {
	w, err := repo.NewUserWordRepo(s.db).FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, wordNotFound(err)
		}
		return nil, apperr.Internal(err)
	}
	if w.DeletedAt != nil {
		return nil, wordNotFound(nil)
	}
	entries, err := s.findEntries(ctx, []*domain.UserWord{w})
	if err != nil {
		return nil, err
	}
	return newWordItem(w, entries[w.DictionaryEntryID]), nil
}

// UpdateReviewMeaning 更新用户自定义复习释义（api/words.md §5，D007）：
// 只写 user_words.custom_review_meaning，绝不修改 dictionary_entries。
// 单条带业务条件的 UPDATE，无需显式事务（架构规范 §9.2）。
func (s *Word) UpdateReviewMeaning(ctx context.Context, id uuid.UUID, req UpdateReviewMeaningRequest) error {
	err := repo.NewUserWordRepo(s.db).UpdateCustomReviewMeaning(ctx, id, req.CustomReviewMeaning, time.Now())
	if errors.Is(err, repo.ErrNotFound) {
		return wordNotFound(err)
	}
	return internalError(err)
}

// DeleteWord 软删除生词（api/words.md §6，D004）：置 deleted_at，不物理
// 删行，历史复习记录与 session 统计保持完整；重新导入时恢复并继续累计。
func (s *Word) DeleteWord(ctx context.Context, id uuid.UUID) error {
	err := repo.NewUserWordRepo(s.db).SoftDelete(ctx, id, time.Now())
	if errors.Is(err, repo.ErrNotFound) {
		return wordNotFound(err)
	}
	return internalError(err)
}

// findEntries 按 user_words 的词条 ID 批量取词典字段，返回按词条 ID
// 索引的映射。user_words.dictionary_entry_id 外键保证词条存在；缺失属于
// 数据损坏，按内部错误处理。
func (s *Word) findEntries(ctx context.Context, words []*domain.UserWord) (map[uuid.UUID]*domain.DictionaryEntry, error) {
	entryIDs := make([]uuid.UUID, 0, len(words))
	seen := make(map[uuid.UUID]struct{}, len(words))
	for _, w := range words {
		if _, ok := seen[w.DictionaryEntryID]; !ok {
			seen[w.DictionaryEntryID] = struct{}{}
			entryIDs = append(entryIDs, w.DictionaryEntryID)
		}
	}
	entries, err := repo.NewDictionaryRepo(s.db).FindByIDs(ctx, entryIDs)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	byID := make(map[uuid.UUID]*domain.DictionaryEntry, len(entries))
	for _, entry := range entries {
		byID[entry.ID] = entry
	}
	for _, w := range words {
		if byID[w.DictionaryEntryID] == nil {
			return nil, apperr.Internal(fmt.Errorf("service: dictionary entry %s missing for user word %s", w.DictionaryEntryID, w.ID))
		}
	}
	return byID, nil
}

// newWordItem 组装列表 / 详情共用的展示读模型。
func newWordItem(userWord *domain.UserWord, entry *domain.DictionaryEntry) *WordItem {
	meaning, source := domain.EffectiveReviewMeaning(userWord.CustomReviewMeaning, entry.ReviewMeanings, entry.RawMeanings)
	return &WordItem{
		UserWord:         userWord,
		Entry:            entry,
		EffectiveMeaning: meaning,
		MeaningSource:    source,
		Phonetic:         displayPhonetic(entry),
		// 派生指标按当前时间计算（D011）：列表 / 详情的每次读取都是最新值。
		MasteryScore: userWord.MasteryScore(),
		ReviewWeight: userWord.ReviewWeight(time.Now()),
	}
}

// displayPhonetic 选择展示音标：API 契约的 phonetic 为单值字段，
// 优先英式，缺失时回退美式。
func displayPhonetic(entry *domain.DictionaryEntry) string {
	if entry.PhoneticUK != "" {
		return entry.PhoneticUK
	}
	return entry.PhoneticUS
}

// wordNotFound 统一生词不存在的应用错误（api/words.md §4 的契约 message）。
func wordNotFound(cause error) error {
	return apperr.New(apperr.NotFound, apperr.CodeNotFound, "word not found", cause)
}
