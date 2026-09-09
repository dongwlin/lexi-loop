package v1

// WordService 集成测试：真实 PostgreSQL 上的生词库用例（docs/api/words.md），
// 含导入聚合、归一累计、软删除恢复、三层释义取值与分页搜索。

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dongwlin/lexi-loop/apps/server/internal/apperr"
	"github.com/dongwlin/lexi-loop/apps/server/internal/domain"
	"github.com/dongwlin/lexi-loop/apps/server/internal/repo"
	"github.com/dongwlin/lexi-loop/apps/server/internal/service"
)

func TestIntegration_WordService_ImportWords(t *testing.T) {
	ctx := context.Background()

	t.Run("首次导入创建词条与学习行并统计 created", func(t *testing.T) {
		resetTables(t)
		svc := newTestWord(t)
		wordA := uniqWord(t, "alpha")
		wordB := uniqWord(t, "beta")

		res, err := svc.ImportWords(ctx, service.ImportWordsRequest{Words: []service.ImportWordItem{
			{Word: "  " + wordA + " ", Count: 2},
			{Word: wordB, Count: 1},
		}})
		require.NoError(t, err)
		assert.Equal(t, 3, res.Encounters)
		assert.Equal(t, 2, res.Created)
		assert.Equal(t, 0, res.Updated)

		// 逐词结果与输入对应（word 归一为小写）。
		require.Len(t, res.Items, 2)
		assert.Equal(t, service.ImportWordOutcome{Word: wordA, Count: 2, Created: true}, res.Items[0])
		assert.Equal(t, service.ImportWordOutcome{Word: wordB, Count: 1, Created: true}, res.Items[1])

		// 词条与学习行：最小词条 lemma 保留原词，遇词次数按输入累计。
		entry, err := repo.NewDictionaryRepo(testDB).FindByHeadword(ctx, wordA)
		require.NoError(t, err)
		assert.Equal(t, wordA, entry.Lemma)

		userWordRepo := repo.NewUserWordRepo(testDB)
		words, total, err := userWordRepo.List(ctx, repo.ListParams{Page: 1, PageSize: 100})
		require.NoError(t, err)
		assert.Equal(t, int64(2), total)
		counts := map[uuid.UUID]int{}
		for _, w := range words {
			counts[w.DictionaryEntryID] = w.EncounterCount
		}
		assert.Equal(t, 2, counts[entry.ID])
	})

	t.Run("重复导入累计遇词次数且不新建行", func(t *testing.T) {
		resetTables(t)
		svc := newTestWord(t)
		word := uniqWord(t, "repeat")

		first, err := svc.ImportWords(ctx, importReq(word, 1))
		require.NoError(t, err)
		assert.Equal(t, 1, first.Created)

		second, err := svc.ImportWords(ctx, importReq(word, 3))
		require.NoError(t, err)
		assert.Equal(t, 0, second.Created)
		assert.Equal(t, 1, second.Updated)
		assert.Equal(t, 3, second.Encounters)
		assert.Equal(t, []service.ImportWordOutcome{{Word: word, Count: 3, Created: false}}, second.Items)

		words, total, err := repo.NewUserWordRepo(testDB).List(ctx, repo.ListParams{Page: 1, PageSize: 100})
		require.NoError(t, err)
		require.Len(t, words, 1)
		assert.Equal(t, int64(1), total)
		assert.Equal(t, 4, words[0].EncounterCount, "1 + 3 = 4")
	})

	t.Run("同一请求内聚合重复输入与归一同词条的不同词形", func(t *testing.T) {
		resetTables(t)
		svc := newTestWord(t)
		base := uniqWord(t, "constrain")
		inflected := uniqWord(t, "constrained")
		mustInsertEntry(t, testEntry(base, base, map[string]string{"0": base}, nil, nil))
		mustInsertEntry(t, testEntry(inflected, inflected, map[string]string{"0": base}, nil, nil))

		res, err := svc.ImportWords(ctx, service.ImportWordsRequest{Words: []service.ImportWordItem{
			{Word: base, Count: 1},
			{Word: base, Count: 2},
			{Word: inflected, Count: 3},
		}})
		require.NoError(t, err)
		assert.Equal(t, 6, res.Encounters)
		assert.Equal(t, 1, res.Created, "两种输入归一到同一词条，只建一行学习数据")

		// 逐词结果是单词粒度：归一到同一新建词条的输入词形都记 created，
		// 与行粒度的聚合统计（created=1）计量单位不同（api/words.md §2）。
		require.Len(t, res.Items, 2)
		assert.Equal(t, service.ImportWordOutcome{Word: base, Count: 3, Created: true}, res.Items[0])
		assert.Equal(t, service.ImportWordOutcome{Word: inflected, Count: 3, Created: true}, res.Items[1])

		words, _, err := repo.NewUserWordRepo(testDB).List(ctx, repo.ListParams{Page: 1, PageSize: 100})
		require.NoError(t, err)
		require.Len(t, words, 1)
		assert.Equal(t, 6, words[0].EncounterCount)
	})

	t.Run("重新导入恢复已软删除的词条", func(t *testing.T) {
		resetTables(t)
		svc := newTestWord(t)
		word := uniqWord(t, "restore")

		first, err := svc.ImportWords(ctx, importReq(word, 1))
		require.NoError(t, err)
		require.Equal(t, 1, first.Created)

		items, err := svc.ListWords(ctx, service.ListWordsRequest{Page: 1, PageSize: 100})
		require.NoError(t, err)
		require.Len(t, items.Items, 1)
		wordID := items.Items[0].UserWord.ID
		require.NoError(t, svc.DeleteWord(ctx, wordID))

		second, err := svc.ImportWords(ctx, importReq(word, 2))
		require.NoError(t, err)
		assert.Equal(t, 0, second.Created)
		assert.Equal(t, 1, second.Updated)
		assert.Equal(t, []service.ImportWordOutcome{{Word: word, Count: 2, Created: false}}, second.Items,
			"恢复软删除在逐词结果中记 updated")

		restored, err := repo.NewUserWordRepo(testDB).FindByID(ctx, wordID)
		require.NoError(t, err)
		assert.Nil(t, restored.DeletedAt, "应恢复软删除而非新建行")
		assert.Equal(t, 3, restored.EncounterCount, "历史遇词次数继续累计")
	})

	t.Run("空请求与非法输入返回参数错误", func(t *testing.T) {
		resetTables(t)
		svc := newTestWord(t)

		for name, req := range map[string]service.ImportWordsRequest{
			"空列表":      {Words: []service.ImportWordItem{}},
			"计数为零":     {Words: []service.ImportWordItem{{Word: "abc", Count: 0}}},
			"归一后为空字符串": {Words: []service.ImportWordItem{{Word: "   ", Count: 1}}},
		} {
			t.Run(name, func(t *testing.T) {
				_, err := svc.ImportWords(ctx, req)
				require.Error(t, err)
				var appErr *apperr.Error
				require.True(t, errors.As(err, &appErr))
				assert.Equal(t, apperr.InvalidArgument, appErr.Kind)
				assert.Equal(t, apperr.CodeValidationFailed, appErr.Code)
			})
		}
	})

	t.Run("并发导入同一新词只创建一个词条", func(t *testing.T) {
		resetTables(t)
		svc := newTestWord(t)
		word := uniqWord(t, "race")

		type result struct {
			res *service.ImportWordsResult
			err error
		}
		results := make(chan result, 2)
		for i := 0; i < 2; i++ {
			go func() {
				res, err := svc.ImportWords(context.Background(), importReq(word, 1))
				results <- result{res, err}
			}()
		}
		var created, updated int
		for i := 0; i < 2; i++ {
			r := <-results
			require.NoError(t, r.err)
			created += r.res.Created
			updated += r.res.Updated
		}
		assert.Equal(t, 1, created, "headword 唯一约束裁决并发创建")
		assert.Equal(t, 1, updated)

		entries, err := repo.NewDictionaryRepo(testDB).FindByLemma(ctx, word)
		require.NoError(t, err)
		assert.Len(t, entries, 1)

		words, _, err := repo.NewUserWordRepo(testDB).List(ctx, repo.ListParams{Page: 1, PageSize: 100})
		require.NoError(t, err)
		require.Len(t, words, 1)
		assert.Equal(t, 2, words[0].EncounterCount, "两个请求的遇词次数都累计")
	})
}

func TestIntegration_WordService_ListWords(t *testing.T) {
	ctx := context.Background()

	t.Run("分页按新导入在前排序并返回总数", func(t *testing.T) {
		resetTables(t)
		svc := newTestWord(t)
		for _, suffix := range []string{"a", "b", "c", "d", "e"} {
			_, err := svc.ImportWords(ctx, importReq(uniqWord(t, suffix), 1))
			require.NoError(t, err)
		}

		res, err := svc.ListWords(ctx, service.ListWordsRequest{Page: 2, PageSize: 2})
		require.NoError(t, err)
		assert.Equal(t, int64(5), res.Total)
		require.Len(t, res.Items, 2)

		// created_at 倒序：第 2 页应是第 3、4 新的两行。
		all, err := svc.ListWords(ctx, service.ListWordsRequest{Page: 1, PageSize: 100})
		require.NoError(t, err)
		assert.Equal(t, all.Items[2:4], res.Items)
	})

	t.Run("search 匹配单词与释义", func(t *testing.T) {
		resetTables(t)
		svc := newTestWord(t)
		word := uniqWord(t, "ambiguous")
		// 预置带释义的词条，再走导入建立学习行。
		entry := testEntry(word, word, nil,
			[]domain.Meaning{{Pos: "adjective", Translations: []string{"模棱两可的"}}}, nil)
		mustInsertEntry(t, entry)
		_, err := svc.ImportWords(ctx, importReq(word, 1))
		require.NoError(t, err)
		other := uniqWord(t, "plain")
		_, err = svc.ImportWords(ctx, importReq(other, 1))
		require.NoError(t, err)

		// uniqWord 生成的词共享用例名前缀，按各自独有的尾部子串搜索。
		byHeadword, err := svc.ListWords(ctx, service.ListWordsRequest{Page: 1, PageSize: 100, Search: "ambiguous"})
		require.NoError(t, err)
		require.Len(t, byHeadword.Items, 1)
		assert.Equal(t, word, byHeadword.Items[0].Entry.Headword)

		byMeaning, err := svc.ListWords(ctx, service.ListWordsRequest{Page: 1, PageSize: 100, Search: "模棱两可"})
		require.NoError(t, err)
		require.Len(t, byMeaning.Items, 1)
		assert.Equal(t, entry.ID, byMeaning.Items[0].Entry.ID)

		none, err := svc.ListWords(ctx, service.ListWordsRequest{Page: 1, PageSize: 100, Search: "不存在的搜索词xyz"})
		require.NoError(t, err)
		assert.Empty(t, none.Items)
		assert.Equal(t, int64(0), none.Total)
	})

	t.Run("列表过滤软删除的生词", func(t *testing.T) {
		resetTables(t)
		svc := newTestWord(t)
		keep := uniqWord(t, "keep")
		drop := uniqWord(t, "drop")
		_, err := svc.ImportWords(ctx, importReq(keep, 1))
		require.NoError(t, err)
		_, err = svc.ImportWords(ctx, importReq(drop, 1))
		require.NoError(t, err)

		items, err := svc.ListWords(ctx, service.ListWordsRequest{Page: 1, PageSize: 100})
		require.NoError(t, err)
		require.Len(t, items.Items, 2)
		var dropID string
		for _, item := range items.Items {
			if item.Entry.Headword == drop {
				dropID = item.UserWord.ID.String()
			}
		}
		require.NotEmpty(t, dropID)
		require.NoError(t, svc.DeleteWord(ctx, mustParseUUID(t, dropID)))

		after, err := svc.ListWords(ctx, service.ListWordsRequest{Page: 1, PageSize: 100})
		require.NoError(t, err)
		assert.Equal(t, int64(1), after.Total)
		assert.Equal(t, keep, after.Items[0].Entry.Headword)
	})
}

func TestIntegration_WordService_GetWord(t *testing.T) {
	ctx := context.Background()

	t.Run("详情组合学习字段与词典字段并按三层取值", func(t *testing.T) {
		resetTables(t)
		svc := newTestWord(t)
		word := uniqWord(t, "detail")
		raw := []domain.Meaning{{Pos: "adjective", Translations: []string{"原始释义"}}}
		review := []domain.Meaning{{Pos: "adjective", Translations: []string{"词典层复习释义"}}}
		mustInsertEntry(t, testEntry(word, word, nil, raw, review))
		_, err := svc.ImportWords(ctx, importReq(word, 3))
		require.NoError(t, err)

		list, err := svc.ListWords(ctx, service.ListWordsRequest{Page: 1, PageSize: 100})
		require.NoError(t, err)
		require.Len(t, list.Items, 1)
		id := list.Items[0].UserWord.ID

		// 未自定义：回退词典层 review_meanings；音标优先英式。
		detail, err := svc.GetWord(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, domain.MeaningSourceReview, detail.MeaningSource)
		assert.Equal(t, review, detail.EffectiveMeaning)
		assert.Equal(t, "/uk/", detail.Phonetic)
		assert.Equal(t, 3, detail.UserWord.EncounterCount)

		// 派生指标（D011 动态计算）：新词 review=0 → mastery 50、
		// weight = 1 + log2(3+1) + 0.5×4 + 2 = 7（新词无时间因素，确定性成立）。
		assert.Equal(t, 50.0, detail.MasteryScore)
		assert.InDelta(t, 7.0, detail.ReviewWeight, 1e-9)

		// 自定义后：取 custom 层。
		custom := []domain.Meaning{{Pos: "adjective", Translations: []string{"我的记忆方式"}}}
		require.NoError(t, svc.UpdateReviewMeaning(ctx, id, service.UpdateReviewMeaningRequest{CustomReviewMeaning: custom}))
		customized, err := svc.GetWord(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, domain.MeaningSourceCustom, customized.MeaningSource)
		assert.Equal(t, custom, customized.EffectiveMeaning)
	})

	t.Run("词条缺失复习释义时回退原始释义", func(t *testing.T) {
		resetTables(t)
		svc := newTestWord(t)
		word := uniqWord(t, "rawonly")
		raw := []domain.Meaning{{Pos: "noun", Translations: []string{"原始释义"}}}
		mustInsertEntry(t, testEntry(word, word, nil, raw, nil))
		_, err := svc.ImportWords(ctx, importReq(word, 1))
		require.NoError(t, err)

		list, err := svc.ListWords(ctx, service.ListWordsRequest{Page: 1, PageSize: 100})
		require.NoError(t, err)
		detail, err := svc.GetWord(ctx, list.Items[0].UserWord.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.MeaningSourceRaw, detail.MeaningSource)
		assert.Equal(t, raw, detail.EffectiveMeaning)
	})

	t.Run("派生指标按当前时间动态计算", func(t *testing.T) {
		resetTables(t)
		svc := newTestWord(t)
		word := uniqWord(t, "metrics")
		_, err := svc.ImportWords(ctx, importReq(word, 1))
		require.NoError(t, err)
		list, err := svc.ListWords(ctx, service.ListWordsRequest{Page: 1, PageSize: 100})
		require.NoError(t, err)
		require.Len(t, list.Items, 1)
		id := list.Items[0].UserWord.ID

		// 模拟已有复习记录：remember 1 / forget 2，1 小时前复习过。
		w, err := repo.NewUserWordRepo(testDB).FindByID(ctx, id)
		require.NoError(t, err)
		w.ReviewCount, w.RememberCount, w.ForgetCount = 3, 1, 2
		w.CurrentStreak = -1
		reviewedAt := time.Now().Add(-time.Hour)
		w.LastReviewedAt = &reviewedAt
		w.UpdatedAt = time.Now()
		require.NoError(t, repo.NewUserWordRepo(testDB).UpdateReviewCounters(ctx, w))

		detail, err := svc.GetWord(ctx, id)
		require.NoError(t, err)
		// mastery = (1+1)/(3+2)×100 = 40；weight 与领域公式按当前时刻一致
		// （两次取 now 的间隔远小于容差）。
		assert.InDelta(t, 40.0, detail.MasteryScore, 1e-9)
		assert.InDelta(t, w.ReviewWeight(time.Now()), detail.ReviewWeight, 0.01)

		again, err := svc.ListWords(ctx, service.ListWordsRequest{Page: 1, PageSize: 100})
		require.NoError(t, err)
		assert.InDelta(t, detail.ReviewWeight, again.Items[0].ReviewWeight, 0.01,
			"列表与详情同口径动态计算")
	})

	t.Run("不存在或已软删除时返回未找到", func(t *testing.T) {
		resetTables(t)
		svc := newTestWord(t)
		word := uniqWord(t, "gone")
		_, err := svc.ImportWords(ctx, importReq(word, 1))
		require.NoError(t, err)
		list, err := svc.ListWords(ctx, service.ListWordsRequest{Page: 1, PageSize: 100})
		require.NoError(t, err)
		id := list.Items[0].UserWord.ID

		require.NoError(t, svc.DeleteWord(ctx, id))

		for name, target := range map[string]uuid.UUID{
			"不存在的 ID": uuid.Must(uuid.NewV7()),
			"已软删除":    id,
		} {
			t.Run(name, func(t *testing.T) {
				_, err := svc.GetWord(ctx, target)
				require.Error(t, err)
				var appErr *apperr.Error
				require.True(t, errors.As(err, &appErr))
				assert.Equal(t, apperr.NotFound, appErr.Kind)
				assert.Equal(t, apperr.CodeNotFound, appErr.Code)
				assert.Equal(t, "word not found", appErr.Message)
			})
		}
	})
}

func TestIntegration_WordService_UpdateReviewMeaning(t *testing.T) {
	ctx := context.Background()

	setup := func(t *testing.T) (*Word, uuid.UUID) {
		resetTables(t)
		svc := newTestWord(t)
		word := uniqWord(t, "meaning")
		mustInsertEntry(t, testEntry(word, word, nil,
			[]domain.Meaning{{Pos: "noun", Translations: []string{"原始释义"}}}, nil))
		_, err := svc.ImportWords(ctx, importReq(word, 1))
		require.NoError(t, err)
		list, err := svc.ListWords(ctx, service.ListWordsRequest{Page: 1, PageSize: 100})
		require.NoError(t, err)
		return svc, list.Items[0].UserWord.ID
	}

	t.Run("设置后清除回退到词典层", func(t *testing.T) {
		svc, id := setup(t)
		custom := []domain.Meaning{{Pos: "noun", Translations: []string{"自定义"}}}
		require.NoError(t, svc.UpdateReviewMeaning(ctx, id, service.UpdateReviewMeaningRequest{CustomReviewMeaning: custom}))

		stored, err := repo.NewUserWordRepo(testDB).FindByID(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, custom, stored.CustomReviewMeaning)

		// nil 表示清除（契约：传 null 清除自定义）。
		require.NoError(t, svc.UpdateReviewMeaning(ctx, id, service.UpdateReviewMeaningRequest{}))
		cleared, err := repo.NewUserWordRepo(testDB).FindByID(ctx, id)
		require.NoError(t, err)
		assert.Nil(t, cleared.CustomReviewMeaning)
	})

	t.Run("目标不存在时返回未找到", func(t *testing.T) {
		svc, _ := setup(t)
		err := svc.UpdateReviewMeaning(ctx, uuid.Must(uuid.NewV7()), service.UpdateReviewMeaningRequest{
			CustomReviewMeaning: []domain.Meaning{{Pos: "noun", Translations: []string{"x"}}},
		})
		require.Error(t, err)
		var appErr *apperr.Error
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, apperr.NotFound, appErr.Kind)
	})
}

func TestIntegration_WordService_DeleteWord(t *testing.T) {
	ctx := context.Background()

	t.Run("软删除保留行并从列表消失", func(t *testing.T) {
		resetTables(t)
		svc := newTestWord(t)
		word := uniqWord(t, "delete")
		_, err := svc.ImportWords(ctx, importReq(word, 1))
		require.NoError(t, err)
		list, err := svc.ListWords(ctx, service.ListWordsRequest{Page: 1, PageSize: 100})
		require.NoError(t, err)
		id := list.Items[0].UserWord.ID

		require.NoError(t, svc.DeleteWord(ctx, id))

		stored, err := repo.NewUserWordRepo(testDB).FindByID(ctx, id)
		require.NoError(t, err, "软删除不物理删行")
		assert.NotNil(t, stored.DeletedAt)

		err = svc.DeleteWord(ctx, id)
		require.Error(t, err, "重复删除视为未找到")
		var appErr *apperr.Error
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, apperr.NotFound, appErr.Kind)
	})

	t.Run("目标不存在时返回未找到", func(t *testing.T) {
		resetTables(t)
		svc := newTestWord(t)
		err := svc.DeleteWord(ctx, uuid.Must(uuid.NewV7()))
		require.Error(t, err)
		var appErr *apperr.Error
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, apperr.NotFound, appErr.Kind)
	})
}

// ---- 用例专属辅助 ----

// newTestWord 构造共享数据库上的 WordService。
func newTestWord(t *testing.T) *Word {
	t.Helper()
	db := requireTestDB(t)
	return NewWord(db, NewDictionary(db))
}

// importReq 构造单词条导入请求。
func importReq(word string, count int) service.ImportWordsRequest {
	return service.ImportWordsRequest{Words: []service.ImportWordItem{{Word: word, Count: count}}}
}
