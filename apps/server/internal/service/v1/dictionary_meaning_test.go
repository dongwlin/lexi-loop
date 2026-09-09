package v1

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dongwlin/lexi-loop/apps/server/internal/domain"
	"github.com/dongwlin/lexi-loop/apps/server/internal/repo"
	"github.com/dongwlin/lexi-loop/apps/server/internal/service"
)

func TestIntegration_InflectionMeanings(t *testing.T) {
	ctx := context.Background()
	for _, tt := range []struct{ name, word, base, relation, semantic string }{
		{"过去式", "oversaw", "oversee", "oversee的过去式", "监督；监管；管理"},
		{"复数", "children", "child", "child的复数", "孩子"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			resetTables(t)
			raw := []domain.Meaning{{Pos: "v", Translations: []string{tt.relation}}}
			semantic := []domain.Meaning{{Translations: []string{tt.semantic}}}
			entry := testEntry(tt.word, tt.word, nil, raw, nil)
			mustInsertEntry(t, entry)
			mustInsertEntry(t, testEntry(tt.base, tt.base, nil, semantic, nil))
			wordSvc := newTestWord(t)
			_, err := wordSvc.ImportWords(ctx, importReq(tt.word, 1))
			require.NoError(t, err)
			list, err := wordSvc.ListWords(ctx, service.ListWordsRequest{Page: 1, PageSize: 20})
			require.NoError(t, err)
			require.Len(t, list.Items, 1)
			id := list.Items[0].UserWord.ID
			detail, err := wordSvc.GetWord(ctx, id)
			require.NoError(t, err)
			want := append(append([]domain.Meaning{}, raw...), semantic...)
			assert.Equal(t, want, detail.EffectiveMeaning)
			assert.Equal(t, domain.MeaningSourceRaw, detail.MeaningSource)
			assert.Equal(t, list.Items[0].EffectiveMeaning, detail.EffectiveMeaning)
			assert.Equal(t, tt.word, detail.Entry.Headword)
			assert.Equal(t, raw, detail.Entry.RawMeanings)
			review := mustStart(t, newTestReview(t), 1)
			assert.Equal(t, detail.EffectiveMeaning, review.Items[0].EffectiveMeaning)
			assert.Equal(t, tt.word, review.Items[0].Headword)
			custom := []domain.Meaning{{Translations: []string{"自己的释义"}}}
			require.NoError(t, wordSvc.UpdateReviewMeaning(ctx, id, service.UpdateReviewMeaningRequest{CustomReviewMeaning: custom}))
			detail, err = wordSvc.GetWord(ctx, id)
			require.NoError(t, err)
			assert.Equal(t, custom, detail.EffectiveMeaning)
			assert.Equal(t, domain.MeaningSourceCustom, detail.MeaningSource)
			customReview := mustStart(t, newTestReview(t), 1)
			assert.Equal(t, custom, customReview.Items[0].EffectiveMeaning)
			require.NoError(t, wordSvc.UpdateReviewMeaning(ctx, id, service.UpdateReviewMeaningRequest{}))
			restored, err := wordSvc.GetWord(ctx, id)
			require.NoError(t, err)
			assert.Equal(t, want, restored.EffectiveMeaning)
			stored, err := repo.NewDictionaryRepo(testDB).FindByID(ctx, entry.ID)
			require.NoError(t, err)
			assert.Equal(t, raw, stored.RawMeanings)
			assert.Empty(t, stored.ReviewMeanings)
		})
	}
}

func TestIntegration_InflectionFallbacks(t *testing.T) {
	ctx := context.Background()
	for _, tt := range []struct {
		name         string
		baseMeanings []domain.Meaning
		missing      bool
	}{
		{name: "原形不存在", missing: true},
		{name: "原形释义为空"},
		{name: "原形释义只有空白", baseMeanings: []domain.Meaning{{Translations: []string{" "}}}},
		{name: "循环引用", baseMeanings: []domain.Meaning{{Translations: []string{"oversaw的过去式"}}}},
		{name: "原形仍指向其他词", baseMeanings: []domain.Meaning{{Translations: []string{"see的过去式"}}}},
		{name: "原形关系存在多个目标", baseMeanings: []domain.Meaning{{Translations: []string{"good的比较级", "well的比较级"}}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			resetTables(t)
			raw := []domain.Meaning{{Translations: []string{"oversee的过去式"}}}
			entry := testEntry("oversaw", "oversee", map[string]string{"0": "oversee"}, raw, nil)
			mustInsertEntry(t, entry)
			if !tt.missing {
				mustInsertEntry(t, testEntry("oversee", "oversee", nil, tt.baseMeanings, nil))
			}
			got, err := findDisplayEntries(ctx, repo.NewDictionaryRepo(testDB), []uuid.UUID{entry.ID})
			require.NoError(t, err)
			meanings, source := got[entry.ID].effectiveMeaning(nil)
			assert.Equal(t, raw, meanings)
			assert.Equal(t, domain.MeaningSourceRaw, source)
		})
	}
}

func TestIntegration_InflectionStructuredAndReviewMeanings(t *testing.T) {
	resetTables(t)
	ctx := context.Background()
	raw := []domain.Meaning{{Translations: []string{"oversee的过去式"}}}
	semantic := []domain.Meaning{{Pos: "v", Translations: []string{"监管"}}}
	entry := testEntry("oversaw", "oversee", map[string]string{"0": "oversee"}, raw, raw)
	mustInsertEntry(t, entry)
	mustInsertEntry(t, testEntry("oversee", "oversee", nil, []domain.Meaning{{Translations: []string{"完整原始释义"}}}, semantic))
	got, err := findDisplayEntries(ctx, repo.NewDictionaryRepo(testDB), []uuid.UUID{entry.ID})
	require.NoError(t, err)
	meanings, source := got[entry.ID].effectiveMeaning(nil)
	assert.Equal(t, append(append([]domain.Meaning{}, raw...), semantic...), meanings)
	assert.Equal(t, domain.MeaningSourceReview, source)
	// 再次使用同一读模型不会累加；原字段保持不变。
	again, _ := got[entry.ID].effectiveMeaning(nil)
	assert.Equal(t, meanings, again)
	assert.Equal(t, raw, got[entry.ID].ReviewMeanings)
}

func TestIntegration_InflectionLegacyAndMixedBase(t *testing.T) {
	for _, tt := range []struct {
		name, word, base, relation string
		baseMeanings, supplement   []domain.Meaning
	}{
		{"旧关系词条", "worked", "work", `work的过去式\nwork的过去分词`, []domain.Meaning{{Translations: []string{"工作"}}}, []domain.Meaning{{Translations: []string{"工作"}}}},
		{"旧原形只有关系", "worked", "work", "work的过去式", []domain.Meaning{{Translations: []string{`act的过去式\nact的过去分词`}}}, nil},
		{"混合原形不带第二层关系", "laid", "lay", "lay的过去式和过去分词", []domain.Meaning{{Pos: "v", Translations: []string{"放置"}}, {Translations: []string{"lie的过去式"}}}, []domain.Meaning{{Pos: "v", Translations: []string{"放置"}}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			resetTables(t)
			ctx := context.Background()
			raw := []domain.Meaning{{Translations: []string{tt.relation}}}
			mustInsertEntry(t, testEntry(tt.word, tt.word, nil, raw, nil))
			mustInsertEntry(t, testEntry(tt.base, tt.base, nil, tt.baseMeanings, nil))
			svc := newTestWord(t)
			_, err := svc.ImportWords(ctx, importReq(tt.word, 1))
			require.NoError(t, err)
			list, err := svc.ListWords(ctx, service.ListWordsRequest{Page: 1, PageSize: 20})
			require.NoError(t, err)
			require.Len(t, list.Items, 1)
			detail, err := svc.GetWord(ctx, list.Items[0].UserWord.ID)
			require.NoError(t, err)
			want := append(append([]domain.Meaning{}, raw...), tt.supplement...)
			assert.Equal(t, want, list.Items[0].EffectiveMeaning)
			assert.Equal(t, want, detail.EffectiveMeaning)
			review := mustStart(t, newTestReview(t), 1)
			assert.Equal(t, want, review.Items[0].EffectiveMeaning)
			base, err := repo.NewDictionaryRepo(testDB).FindByHeadword(ctx, tt.base)
			require.NoError(t, err)
			assert.Equal(t, tt.baseMeanings, base.RawMeanings)
		})
	}
}

func TestIntegration_InflectionLegacyLexicalNewlines(t *testing.T) {
	for _, source := range []domain.MeaningSource{domain.MeaningSourceRaw, domain.MeaningSourceReview} {
		t.Run(string(source), func(t *testing.T) {
			resetTables(t)
			ctx := context.Background()
			relation := []domain.Meaning{{Translations: []string{"work的过去式"}}}
			entry := testEntry("worked", "worked", nil, relation, nil)
			if source == domain.MeaningSourceReview {
				entry.ReviewMeanings = relation
			}
			mustInsertEntry(t, entry)
			legacy := []domain.Meaning{{Translations: []string{`工作\n运转`}}}
			mustInsertEntry(t, testEntry("work", "work", nil, legacy, nil))
			svc := newTestWord(t)
			_, err := svc.ImportWords(ctx, importReq("worked", 1))
			require.NoError(t, err)
			list, err := svc.ListWords(ctx, service.ListWordsRequest{Page: 1, PageSize: 20})
			require.NoError(t, err)
			require.Len(t, list.Items, 1)
			id := list.Items[0].UserWord.ID
			detail, err := svc.GetWord(ctx, id)
			require.NoError(t, err)
			want := []domain.Meaning{{Translations: []string{"work的过去式"}}, {Translations: []string{"工作\n运转"}}}
			assert.Equal(t, want, list.Items[0].EffectiveMeaning)
			assert.Equal(t, want, detail.EffectiveMeaning)
			assert.Equal(t, source, detail.MeaningSource)
			review := mustStart(t, newTestReview(t), 1)
			assert.Equal(t, want, review.Items[0].EffectiveMeaning)
			require.NoError(t, svc.UpdateReviewMeaning(ctx, id, service.UpdateReviewMeaningRequest{CustomReviewMeaning: legacy}))
			custom, err := svc.GetWord(ctx, id)
			require.NoError(t, err)
			assert.Equal(t, legacy, custom.EffectiveMeaning, "用户自定义不做解码")
			base, err := repo.NewDictionaryRepo(testDB).FindByHeadword(ctx, "work")
			require.NoError(t, err)
			assert.Equal(t, legacy, base.RawMeanings, "不改数据库")
		})
	}
}
