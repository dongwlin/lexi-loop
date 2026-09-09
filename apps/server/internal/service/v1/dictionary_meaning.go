package v1

import (
	"context"

	"github.com/google/uuid"

	"github.com/dongwlin/lexi-loop/apps/server/internal/domain"
	"github.com/dongwlin/lexi-loop/apps/server/internal/repo"
)

// dictionaryDisplayEntry 将读取期补充与持久化词典字段分开。
type dictionaryDisplayEntry struct {
	*domain.DictionaryEntry
	supplement []domain.Meaning
}

func (e *dictionaryDisplayEntry) effectiveMeaning(custom []domain.Meaning) ([]domain.Meaning, domain.MeaningSource) {
	meaning, source := domain.EffectiveReviewMeaning(custom, e.ReviewMeanings, e.RawMeanings)
	if source == domain.MeaningSourceCustom || len(e.supplement) == 0 {
		return meaning, source
	}
	result := make([]domain.Meaning, 0, len(meaning)+len(e.supplement))
	result = append(result, meaning...)
	return append(result, e.supplement...), source
}

// findDisplayEntries 是详情、列表和复习共用的词典读取路径。
// 最多额外一次批量查询；不递归、不写库，保持调用方事务边界。
func findDisplayEntries(ctx context.Context, entryRepo *repo.DictionaryRepo, ids []uuid.UUID) (map[uuid.UUID]*dictionaryDisplayEntry, error) {
	entries, err := entryRepo.FindByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	result := make(map[uuid.UUID]*dictionaryDisplayEntry, len(entries))
	targets := make(map[uuid.UUID]string)
	seen := make(map[string]bool)
	headwords := make([]string, 0)
	for _, entry := range entries {
		result[entry.ID] = &dictionaryDisplayEntry{DictionaryEntry: entry}
		if target := domain.InflectionLemma(entry); target != "" {
			targets[entry.ID] = target
			if !seen[target] {
				seen[target] = true
				headwords = append(headwords, target)
			}
		}
	}
	bases, err := entryRepo.FindByHeadwords(ctx, headwords)
	if err != nil {
		return nil, err
	}
	semantics := make(map[string][]domain.Meaning, len(bases))
	for _, base := range bases {
		meanings, _ := domain.EffectiveReviewMeaning(nil, base.ReviewMeanings, base.RawMeanings)
		semantics[base.Headword] = domain.LexicalMeanings(meanings)
	}
	for id, target := range targets {
		result[id].supplement = semantics[target]
	}
	return result, nil
}
