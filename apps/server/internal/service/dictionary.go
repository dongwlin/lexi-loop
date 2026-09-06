package service

import (
	"context"
	"errors"
	"time"

	"github.com/uptrace/bun"

	"github.com/dongwlin/lexi-loop/apps/server/internal/apperr"
	"github.com/dongwlin/lexi-loop/apps/server/internal/domain"
	"github.com/dongwlin/lexi-loop/apps/server/internal/repo"
)

// Dictionary 提供词典查询用例。MVP 职责是纯字符串归一 → lemma 解析 →
// 本地词典 Lookup → 未命中时创建最小词条；在线 Provider 的 Lookup 分支
// 与 Enrich 均为 V2 能力，ECDICT 始终不是运行时 Provider
// （docs/backend/structure.md §4.3、docs/dictionary/enrichment.md）。
type Dictionary struct {
	db *bun.DB
}

// NewDictionary 构造 DictionaryService。
func NewDictionary(db *bun.DB) *Dictionary {
	return &Dictionary{db: db}
}

// Lookup 把一个用户输入的词解析为可用的词条（enrichment.md §2）：
// 归一 → 按 headword 查本地词典库 → 命中则按词形变化表尝试归并到原形
// 词条 → 未命中则创建仅含 headword 的最小词条（MVP 无在线分支）。
// 对外用例使用 s.db；需要加入调用方事务时经 lookup(ctx, tx, word) 复用
// 同一实现（structure.md §4.3：跨 Service 不得开启嵌套事务）。
func (s *Dictionary) Lookup(ctx context.Context, word string) (*domain.DictionaryEntry, error) {
	return s.lookup(ctx, s.db, word)
}

// lookup 是 Lookup 的内部实现：db 由调用方传入（*bun.DB 或 bun.Tx），
// 使 ImportWords 能把词条解析并入自己的事务。
func (s *Dictionary) lookup(ctx context.Context, db bun.IDB, word string) (*domain.DictionaryEntry, error) {
	normalized := domain.NormalizeWord(word)
	if normalized == "" {
		return nil, apperr.New(apperr.InvalidArgument, apperr.CodeValidationFailed, "word is required", nil)
	}

	entryRepo := repo.NewDictionaryRepo(db)
	entry, err := entryRepo.FindByHeadword(ctx, normalized)
	if err != nil {
		if !errors.Is(err, repo.ErrNotFound) {
			return nil, err
		}
		// 本地未命中：MVP 一律创建最小词条，lemma 不确定时保留原词
		//（enrichment.md §2、normalization.md §3、data-model.md §10）。
		created, err := domain.NewDictionaryEntry(normalized, "", time.Now())
		if err != nil {
			return nil, err
		}
		// 并发创建同词由 headword 唯一约束裁决（ErrAlreadyExists）；
		// 整批用例可重放，由调用方对整个事务做有上限重试后重查
		//（structure.md §5.3）。
		if err := entryRepo.Insert(ctx, created); err != nil {
			return nil, err
		}
		return created, nil
	}

	// 命中：借助词形变化表把多数规则变化归一到原形（normalization.md §2）。
	// 「不确定不归并」：变化表缺失、指向自身，或原形词条不存在时保留原词，
	// 宁可多一个词条也不错误合并（normalization.md §3）。
	if base, err := s.resolveLemma(ctx, entryRepo, entry); err != nil {
		return nil, err
	} else if base != nil {
		return base, nil
	}
	return entry, nil
}

// resolveLemma 从词条的词形变化表解析原形词条（exchange 键 "0" 为原形）。
// 能唯一确定且原形词条存在时返回该词条；否则返回 (nil, nil) 表示保留原词。
func (s *Dictionary) resolveLemma(ctx context.Context, entryRepo *repo.DictionaryRepo, entry *domain.DictionaryEntry) (*domain.DictionaryEntry, error) {
	lemma := entry.Exchange["0"]
	if lemma == "" || lemma == entry.Headword {
		return nil, nil
	}
	base, err := entryRepo.FindByHeadword(ctx, lemma)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			// 原形词条不在词典库中：归并目标缺失，保留原词。
			return nil, nil
		}
		return nil, err
	}
	return base, nil
}
