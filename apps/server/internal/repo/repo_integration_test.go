package repo

// 集成测试：共享 PostgreSQL 容器与迁移后的 schema 由 main_test.go 的包级
// TestMain 准备，Docker 不可用或传 -short 时跳过。
// 覆盖 Go 单体应用架构规范 §11.2 的 Repo 层要求：真实 PostgreSQL 上的
// 数据操作、schema ↔ domain 往返转换、约束归一化错误，以及 structure.md
// §5 的锁与条件更新边界（advisory lock、FOR UPDATE、条件 UPDATE）。

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dongwlin/lexi-loop/apps/server/internal/domain"
	"github.com/dongwlin/lexi-loop/apps/server/internal/repo/internal/schema"
)

// hw 生成用例内唯一的 headword（集成用例共享数据库，按用例名隔离）。
func hw(t *testing.T, suffix string) string {
	t.Helper()
	return strings.ToLower(t.Name()) + "-" + suffix
}

// microNow 返回截断到微秒的 UTC 当前时间，且内部位表示与 pgx 的
// timestamptz 解码路径（time.Unix(sec, nsec) + UTC）一致：timestamptz
// 精度为微秒，截断保证时间可精确断言，同源构造保证 assert.Equal 的
// DeepEqual 不会因 time.Time 内部表示差异而假失败。
func microNow() time.Time {
	now := time.Now()
	return time.Unix(now.Unix(), int64(now.Nanosecond())).UTC().Truncate(time.Microsecond)
}

func seedEntry(t *testing.T, headword string) *schema.DictionaryEntry {
	t.Helper()
	now := microNow()
	row := &schema.DictionaryEntry{
		ID:        uuid.Must(uuid.NewV7()),
		Headword:  headword,
		Lemma:     headword,
		CreatedAt: now,
		UpdatedAt: now,
	}
	_, err := testDB.NewInsert().Model(row).Exec(context.Background())
	require.NoError(t, err, "seed dictionary entry %s", headword)
	return row
}

// seedWord 建一对词条 + 生词学习行（user_words.dictionary_entry_id 唯一，
// 每个生词需要自己的词条）。
func seedWord(t *testing.T, suffix string) (*schema.DictionaryEntry, *schema.UserWord) {
	t.Helper()
	entry := seedEntry(t, hw(t, suffix))
	now := microNow()
	uw := &schema.UserWord{
		ID:                uuid.Must(uuid.NewV7()),
		DictionaryEntryID: entry.ID,
		EncounterCount:    1,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	_, err := testDB.NewInsert().Model(uw).Exec(context.Background())
	require.NoError(t, err, "seed user word")
	return entry, uw
}

// seedUserWordAt 同 seedWord 的生词部分，但使用显式 created_at（列表排序用例）。
func seedUserWordAt(t *testing.T, entryID uuid.UUID, createdAt time.Time) *schema.UserWord {
	t.Helper()
	uw := &schema.UserWord{
		ID:                uuid.Must(uuid.NewV7()),
		DictionaryEntryID: entryID,
		EncounterCount:    1,
		CreatedAt:         createdAt,
		UpdatedAt:         createdAt,
	}
	_, err := testDB.NewInsert().Model(uw).Exec(context.Background())
	require.NoError(t, err, "seed user word")
	return uw
}

func seedSession(t *testing.T, requested, total int, status domain.ReviewStatus) *schema.ReviewSession {
	t.Helper()
	now := microNow()
	row := &schema.ReviewSession{
		ID:             uuid.Must(uuid.NewV7()),
		RequestedCount: requested,
		TotalCount:     total,
		Status:         string(status),
		StartedAt:      now,
		CreatedAt:      now,
	}
	_, err := testDB.NewInsert().Model(row).Exec(context.Background())
	require.NoError(t, err, "seed review session")
	return row
}

func seedItem(t *testing.T, sessionID, userWordID uuid.UUID, position int) *schema.ReviewItem {
	t.Helper()
	row := &schema.ReviewItem{
		ID:         uuid.Must(uuid.NewV7()),
		SessionID:  sessionID,
		UserWordID: userWordID,
		Position:   position,
		Result:     string(domain.ReviewResultPending),
		CreatedAt:  microNow(),
	}
	_, err := testDB.NewInsert().Model(row).Exec(context.Background())
	require.NoError(t, err, "seed review item")
	return row
}

func TestIntegration_DictionaryRepo_RoundTrip(t *testing.T) {
	db := requireTestDB(t)
	resetTables(t)
	ctx := context.Background()
	now := microNow()
	repo := NewDictionaryRepo(db)

	t.Run("完整词条写入读回不丢数据", func(t *testing.T) {
		entry, err := domain.NewDictionaryEntry("ambiguous", "ambiguous", now)
		require.NoError(t, err)
		entry.PhoneticUK = "/æmˈbɪɡjuəs/"
		entry.PhoneticUS = "/æmˈbɪɡjuəs/"
		entry.RawMeanings = []domain.Meaning{
			{Pos: "adjective", Translations: []string{"模棱两可的", "含糊不清的"}},
			{Pos: "noun", Translations: []string{"歧义"}},
		}
		entry.ReviewMeanings = []domain.Meaning{
			{Pos: "adjective", Translations: []string{"模棱两可的"}},
		}
		entry.Exchange = map[string]string{"0": "ambiguous", "p": "ambiguated"}
		entry.Frequency = map[string]int{"frequency": 3, "bnc": 4, "coca": 5}
		entry.Tags = []string{"cet4", "cet6", "ky"}
		entry.Source = "ecdict"
		entry.SourceVersion = "2025"
		require.NoError(t, repo.Insert(ctx, entry))

		got, err := repo.FindByHeadword(ctx, "ambiguous")
		require.NoError(t, err)
		assert.Equal(t, entry, got, "词条往返转换应无损失")
	})

	t.Run("最小词条的未设置字段读回为 nil", func(t *testing.T) {
		entry, err := domain.NewDictionaryEntry("minimal", "", now)
		require.NoError(t, err)
		require.NoError(t, repo.Insert(ctx, entry))

		got, err := repo.FindByID(ctx, entry.ID)
		require.NoError(t, err)
		assert.Equal(t, "minimal", got.Lemma, "lemma 为空时回退 headword")
		assert.Nil(t, got.RawMeanings)
		assert.Nil(t, got.ReviewMeanings)
		assert.Nil(t, got.Exchange)
		assert.Nil(t, got.Frequency)
		assert.Nil(t, got.Tags)
		assert.Empty(t, got.PhoneticUK)
	})
}

func TestIntegration_DictionaryRepo_Errors(t *testing.T) {
	db := requireTestDB(t)
	resetTables(t)
	ctx := context.Background()
	repo := NewDictionaryRepo(db)

	t.Run("未命中返回 ErrNotFound", func(t *testing.T) {
		_, err := repo.FindByHeadword(ctx, "no-such-word")
		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("headword 重复返回 ErrAlreadyExists", func(t *testing.T) {
		headword := hw(t, "dup")
		entry, err := domain.NewDictionaryEntry(headword, "", microNow())
		require.NoError(t, err)
		require.NoError(t, repo.Insert(ctx, entry))

		dup, err := domain.NewDictionaryEntry(headword, "", microNow())
		require.NoError(t, err)
		err = repo.Insert(ctx, dup)
		assert.ErrorIs(t, err, ErrAlreadyExists)
	})

	t.Run("按 lemma 查询词条候选", func(t *testing.T) {
		lemma := hw(t, "derive")
		for _, form := range []string{"derived", "deriving"} {
			entry, err := domain.NewDictionaryEntry(form, lemma, microNow())
			require.NoError(t, err)
			require.NoError(t, repo.Insert(ctx, entry))
		}
		unrelated, err := domain.NewDictionaryEntry(hw(t, "unrelated"), "", microNow())
		require.NoError(t, err)
		require.NoError(t, repo.Insert(ctx, unrelated))

		got, err := repo.FindByLemma(ctx, lemma)
		require.NoError(t, err)
		require.Len(t, got, 2, "只返回 lemma 匹配的词条")
		for _, entry := range got {
			assert.Equal(t, lemma, entry.Lemma)
		}
	})

	t.Run("按 ID 批量查询只返回命中项", func(t *testing.T) {
		first, err := domain.NewDictionaryEntry(hw(t, "id-a"), "", microNow())
		require.NoError(t, err)
		require.NoError(t, repo.Insert(ctx, first))
		second, err := domain.NewDictionaryEntry(hw(t, "id-b"), "", microNow())
		require.NoError(t, err)
		require.NoError(t, repo.Insert(ctx, second))

		got, err := repo.FindByIDs(ctx, []uuid.UUID{first.ID, second.ID, uuid.Must(uuid.NewV7())})
		require.NoError(t, err)
		assert.Len(t, got, 2)

		empty, err := repo.FindByIDs(ctx, nil)
		require.NoError(t, err)
		assert.Empty(t, empty)
	})
}

func TestIntegration_DictionaryRepo_UpsertBatch(t *testing.T) {
	db := requireTestDB(t)
	resetTables(t)
	ctx := context.Background()
	now := microNow()
	repo := NewDictionaryRepo(db)

	alpha, err := domain.NewDictionaryEntry(hw(t, "alpha"), "", now)
	require.NoError(t, err)
	alpha.RawMeanings = []domain.Meaning{{Pos: "noun", Translations: []string{"初版释义"}}}
	beta, err := domain.NewDictionaryEntry(hw(t, "beta"), "", now)
	require.NoError(t, err)
	require.NoError(t, repo.UpsertBatch(ctx, []*domain.DictionaryEntry{alpha, beta}))

	// 模拟后续整理流程单独写入词典层复习释义（raw UPDATE，repo 不提供该写路径）。
	reviewMeanings := `[{"pos":"noun","translations":["整理后的复习释义"]}]`
	_, err = db.NewUpdate().
		Model((*schema.DictionaryEntry)(nil)).
		Set("review_meanings = ?", reviewMeanings).
		Where("headword = ?", alpha.Headword).
		Exec(ctx)
	require.NoError(t, err)

	// 重跑导入：更新 ECDICT 字段，不新增行，且不覆盖已有 review_meanings。
	alphaUpdated, err := domain.NewDictionaryEntry(alpha.Headword, "", now)
	require.NoError(t, err)
	alphaUpdated.PhoneticUK = "/ˈæmfə/"
	betaUpdated, err := domain.NewDictionaryEntry(beta.Headword, "", now)
	require.NoError(t, err)
	require.NoError(t, repo.UpsertBatch(ctx, []*domain.DictionaryEntry{alphaUpdated, betaUpdated}))

	got, err := repo.FindByHeadword(ctx, alpha.Headword)
	require.NoError(t, err)
	assert.Equal(t, "/ˈæmfə/", got.PhoneticUK, "重跑导入应更新词典字段")
	assert.Equal(t, []domain.Meaning{{Pos: "noun", Translations: []string{"整理后的复习释义"}}},
		got.ReviewMeanings, "导入未提供 review_meanings 时应保留已有值")

	total, err := db.NewSelect().Model((*schema.DictionaryEntry)(nil)).Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, total, "重跑导入不产生重复行")

	// 本次写入提供 review_meanings 时覆盖旧值。
	alphaWithReview, err := domain.NewDictionaryEntry(alpha.Headword, "", now)
	require.NoError(t, err)
	alphaWithReview.ReviewMeanings = []domain.Meaning{{Pos: "noun", Translations: []string{"新版复习释义"}}}
	require.NoError(t, repo.UpsertBatch(ctx, []*domain.DictionaryEntry{alphaWithReview}))
	got, err = repo.FindByHeadword(ctx, alpha.Headword)
	require.NoError(t, err)
	assert.Equal(t, []domain.Meaning{{Pos: "noun", Translations: []string{"新版复习释义"}}}, got.ReviewMeanings)
}

func TestIntegration_UserWordRepo_UpsertEncounters(t *testing.T) {
	db := requireTestDB(t)
	resetTables(t)
	ctx := context.Background()
	now := microNow()
	repo := NewUserWordRepo(db)

	entryA := seedEntry(t, hw(t, "alpha"))
	entryB := seedEntry(t, hw(t, "beta"))

	first, err := domain.NewUserWord(entryA.ID, 1, now)
	require.NoError(t, err)
	second, err := domain.NewUserWord(entryB.ID, 2, now)
	require.NoError(t, err)

	t.Run("首次导入新建并计入遇词次数", func(t *testing.T) {
		results, err := repo.UpsertEncounters(ctx, []*domain.UserWord{first, second})
		require.NoError(t, err)
		require.Len(t, results, 2)
		assert.Equal(t, EncounterResult{DictionaryEntryID: entryA.ID, Created: true}, results[0])
		assert.Equal(t, EncounterResult{DictionaryEntryID: entryB.ID, Created: true}, results[1])

		got, err := repo.FindByID(ctx, first.ID)
		require.NoError(t, err)
		assert.Equal(t, 1, got.EncounterCount)
	})

	t.Run("重复导入累计且不新建行", func(t *testing.T) {
		again, err := domain.NewUserWord(entryA.ID, 2, now)
		require.NoError(t, err)
		results, err := repo.UpsertEncounters(ctx, []*domain.UserWord{again})
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.False(t, results[0].Created)

		got, err := repo.FindByID(ctx, first.ID)
		require.NoError(t, err)
		assert.Equal(t, first.ID, got.ID, "累计应作用在原行上")
		assert.Equal(t, 3, got.EncounterCount)
	})

	t.Run("重新导入恢复软删除的词条", func(t *testing.T) {
		require.NoError(t, repo.SoftDelete(ctx, second.ID, now))
		restored, err := domain.NewUserWord(entryB.ID, 1, now)
		require.NoError(t, err)
		results, err := repo.UpsertEncounters(ctx, []*domain.UserWord{restored})
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.False(t, results[0].Created)

		got, err := repo.FindByID(ctx, second.ID)
		require.NoError(t, err)
		assert.Nil(t, got.DeletedAt, "重新导入应把 deleted_at 置回 NULL")
		assert.Equal(t, 3, got.EncounterCount, "恢复后继续累计")
	})

	t.Run("同一批次内重复词条报错", func(t *testing.T) {
		a, err := domain.NewUserWord(entryA.ID, 1, now)
		require.NoError(t, err)
		b, err := domain.NewUserWord(entryA.ID, 1, now)
		require.NoError(t, err)
		_, err = repo.UpsertEncounters(ctx, []*domain.UserWord{a, b})
		assert.Error(t, err, "调用方须按 api/words.md 聚合重复单词")
	})
}

func TestIntegration_UserWordRepo_UpdateMeaningAndDelete(t *testing.T) {
	db := requireTestDB(t)
	resetTables(t)
	ctx := context.Background()
	now := microNow()
	repo := NewUserWordRepo(db)

	_, uw := seedWord(t, "target")
	custom := []domain.Meaning{{Pos: "adjective", Translations: []string{"我的记忆方式"}}}

	t.Run("设置与清除自定义复习释义", func(t *testing.T) {
		require.NoError(t, repo.UpdateCustomReviewMeaning(ctx, uw.ID, custom, now))
		got, err := repo.FindByID(ctx, uw.ID)
		require.NoError(t, err)
		assert.Equal(t, custom, got.CustomReviewMeaning)

		require.NoError(t, repo.UpdateCustomReviewMeaning(ctx, uw.ID, nil, now))
		got, err = repo.FindByID(ctx, uw.ID)
		require.NoError(t, err)
		assert.Nil(t, got.CustomReviewMeaning, "传 nil 表示清除自定义")
	})

	t.Run("软删除后不再可更新且重复删除报错", func(t *testing.T) {
		require.NoError(t, repo.SoftDelete(ctx, uw.ID, now))
		got, err := repo.FindByID(ctx, uw.ID)
		require.NoError(t, err)
		require.NotNil(t, got.DeletedAt, "软删除保留行")

		err = repo.UpdateCustomReviewMeaning(ctx, uw.ID, custom, now)
		assert.ErrorIs(t, err, ErrNotFound)

		err = repo.SoftDelete(ctx, uw.ID, now)
		assert.ErrorIs(t, err, ErrNotFound, "已删除的行不再命中条件更新")
	})

	t.Run("不存在的行返回 ErrNotFound", func(t *testing.T) {
		err := repo.UpdateCustomReviewMeaning(ctx, uuid.Must(uuid.NewV7()), custom, now)
		assert.ErrorIs(t, err, ErrNotFound)
		err = repo.SoftDelete(ctx, uuid.Must(uuid.NewV7()), now)
		assert.ErrorIs(t, err, ErrNotFound)
	})
}

func TestIntegration_UserWordRepo_UpdateReviewCounters(t *testing.T) {
	db := requireTestDB(t)
	resetTables(t)
	ctx := context.Background()
	now := microNow()
	repo := NewUserWordRepo(db)

	_, uw := seedWord(t, "counters")

	t.Run("ApplyReview 后写回累计字段", func(t *testing.T) {
		w, err := repo.FindByID(ctx, uw.ID)
		require.NoError(t, err)
		require.NoError(t, w.ApplyReview(domain.ReviewResultRemembered, now))
		require.NoError(t, w.ApplyReview(domain.ReviewResultForgotten, now.Add(time.Second)))
		require.NoError(t, repo.UpdateReviewCounters(ctx, w))

		got, err := repo.FindByID(ctx, uw.ID)
		require.NoError(t, err)
		assert.Equal(t, 2, got.ReviewCount)
		assert.Equal(t, 1, got.RememberCount)
		assert.Equal(t, 1, got.ForgetCount)
		assert.Equal(t, -1, got.CurrentStreak)
		require.NotNil(t, got.LastReviewedAt)
		assert.True(t, w.LastReviewedAt.Equal(*got.LastReviewedAt))
	})

	t.Run("目标行不存在返回 ErrNotFound", func(t *testing.T) {
		missing := &domain.UserWord{ID: uuid.Must(uuid.NewV7())}
		err := repo.UpdateReviewCounters(ctx, missing)
		assert.ErrorIs(t, err, ErrNotFound)
	})
}

func TestIntegration_UserWordRepo_List(t *testing.T) {
	db := requireTestDB(t)
	resetTables(t)
	ctx := context.Background()
	base := microNow()
	repo := NewUserWordRepo(db)

	// 四条生词：alpha 只在 headword 命中；beta 命中 raw 层释义；delta 命中
	// review 层释义；gamma 命中 custom 层释义。
	alpha, err := domain.NewDictionaryEntry(hw(t, "alpha"), "", base.Add(-3*time.Hour))
	require.NoError(t, err)
	beta, err := domain.NewDictionaryEntry(hw(t, "beta"), "", base.Add(-2*time.Hour))
	require.NoError(t, err)
	beta.RawMeanings = []domain.Meaning{{Pos: "noun", Translations: []string{"田野样本"}}}
	gamma, err := domain.NewDictionaryEntry(hw(t, "gamma"), "", base.Add(-time.Hour))
	require.NoError(t, err)
	delta, err := domain.NewDictionaryEntry(hw(t, "delta"), "", base)
	require.NoError(t, err)
	delta.ReviewMeanings = []domain.Meaning{{Pos: "noun", Translations: []string{"复习层样本"}}}
	for _, entry := range []*domain.DictionaryEntry{alpha, beta, gamma, delta} {
		require.NoError(t, NewDictionaryRepo(db).Insert(ctx, entry))
	}

	alphaUW := seedUserWordAt(t, alpha.ID, base.Add(-3*time.Hour))
	betaUW := seedUserWordAt(t, beta.ID, base.Add(-2*time.Hour))
	gammaUW := seedUserWordAt(t, gamma.ID, base.Add(-time.Hour))
	deltaUW := seedUserWordAt(t, delta.ID, base)
	require.NoError(t, repo.UpdateCustomReviewMeaning(ctx, gammaUW.ID,
		[]domain.Meaning{{Pos: "noun", Translations: []string{"自定义层样本"}}}, base))

	t.Run("分页与 created_at 倒序", func(t *testing.T) {
		words, total, err := repo.List(ctx, ListParams{Page: 1, PageSize: 2})
		require.NoError(t, err)
		assert.Equal(t, int64(4), total)
		require.Len(t, words, 2)
		assert.Equal(t, deltaUW.ID, words[0].ID, "最新的词在前")
		assert.Equal(t, gammaUW.ID, words[1].ID)

		words, total, err = repo.List(ctx, ListParams{Page: 2, PageSize: 2})
		require.NoError(t, err)
		assert.Equal(t, int64(4), total)
		require.Len(t, words, 2)
		assert.Equal(t, betaUW.ID, words[0].ID)
		assert.Equal(t, alphaUW.ID, words[1].ID)
	})

	t.Run("search 按 headword 与各层释义匹配", func(t *testing.T) {
		cases := []struct {
			name      string
			search    string
			wantTotal int64
		}{
			{"headword 前缀命中", "alpha", 1},
			{"headword 大小写不敏感", "ALPHA", 1},
			{"raw 层释义命中", "田野样本", 1},
			{"review 层释义命中", "复习层样本", 1},
			{"custom 层释义命中", "自定义层样本", 1},
			{"多行命中", "样本", 3},
			{"无命中", "不存在的词", 0},
		}
		for _, tt := range cases {
			t.Run(tt.name, func(t *testing.T) {
				_, total, err := repo.List(ctx, ListParams{Page: 1, PageSize: 20, Search: tt.search})
				require.NoError(t, err)
				assert.Equal(t, tt.wantTotal, total)
			})
		}
	})

	t.Run("软删除的词不再出现", func(t *testing.T) {
		require.NoError(t, repo.SoftDelete(ctx, alphaUW.ID, microNow()))
		_, total, err := repo.List(ctx, ListParams{Page: 1, PageSize: 20})
		require.NoError(t, err)
		assert.Equal(t, int64(3), total)
	})
}

func TestIntegration_UserWordRepo_ListActive(t *testing.T) {
	db := requireTestDB(t)
	resetTables(t)
	ctx := context.Background()
	repo := NewUserWordRepo(db)

	_, first := seedWord(t, "a")
	_, second := seedWord(t, "b")
	require.NoError(t, repo.SoftDelete(ctx, second.ID, microNow()))

	words, err := repo.ListActive(ctx)
	require.NoError(t, err)
	require.Len(t, words, 1, "加权抽样的候选集只含未删除的生词")
	assert.Equal(t, first.ID, words[0].ID)
}

func TestIntegration_UserWordRepo_LockByID(t *testing.T) {
	db := requireTestDB(t)
	resetTables(t)
	ctx := context.Background()
	repo := NewUserWordRepo(db)

	_, uw := seedWord(t, "lock")

	got, err := repo.LockByID(ctx, uw.ID)
	require.NoError(t, err)
	assert.Equal(t, uw.ID, got.ID)

	_, err = repo.LockByID(ctx, uuid.Must(uuid.NewV7()))
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestIntegration_ReviewRepo_SessionAndItems(t *testing.T) {
	db := requireTestDB(t)
	resetTables(t)
	ctx := context.Background()
	now := microNow()
	repo := NewReviewRepo(db)

	session, err := domain.NewReviewSession(10, 3, now)
	require.NoError(t, err)
	require.NoError(t, repo.InsertSession(ctx, session))

	var uws []*schema.UserWord
	var headwords []string
	for _, suffix := range []string{"a", "b", "c"} {
		entry, uw := seedWord(t, suffix)
		uws = append(uws, uw)
		headwords = append(headwords, entry.Headword)
	}
	var items []*domain.ReviewItem
	for i, uw := range uws {
		item, err := domain.NewReviewItem(session.ID, uw.ID, i, now)
		require.NoError(t, err)
		items = append(items, item)
	}
	require.NoError(t, repo.InsertItems(ctx, items))

	t.Run("session 读回一致", func(t *testing.T) {
		got, err := repo.GetSessionByID(ctx, session.ID)
		require.NoError(t, err)
		assert.Equal(t, session, got)
	})

	t.Run("items 按 position 排序并带出 headword", func(t *testing.T) {
		got, err := repo.ListSessionItems(ctx, session.ID)
		require.NoError(t, err)
		require.Len(t, got, 3)
		for i, row := range got {
			assert.Equal(t, items[i].ID, row.Item.ID)
			assert.Equal(t, domain.ReviewResultPending, row.Item.Result)
			assert.Equal(t, uws[i].ID, row.Item.UserWordID)
			assert.Equal(t, headwords[i], row.Headword)
		}
	})

	t.Run("初始全部为 pending", func(t *testing.T) {
		remembered, forgotten, pending, err := repo.CountResults(ctx, session.ID)
		require.NoError(t, err)
		assert.Zero(t, remembered)
		assert.Zero(t, forgotten)
		assert.Equal(t, 3, pending)
	})
}

func TestIntegration_ReviewRepo_SubmitItem(t *testing.T) {
	db := requireTestDB(t)
	resetTables(t)
	ctx := context.Background()
	now := microNow()
	repo := NewReviewRepo(db)

	session, err := domain.NewReviewSession(5, 2, now)
	require.NoError(t, err)
	require.NoError(t, repo.InsertSession(ctx, session))

	_, uwA := seedWord(t, "a")
	_, uwB := seedWord(t, "b")
	itemA := seedItem(t, session.ID, uwA.ID, 0)
	itemB := seedItem(t, session.ID, uwB.ID, 1)
	otherSession := seedSession(t, 1, 1, domain.ReviewStatusAbandoned)

	t.Run("pending 条件更新命中并返回 user_word_id", func(t *testing.T) {
		got, err := repo.UpdateItemIfPending(ctx, session.ID, itemA.ID, domain.ReviewResultRemembered, now)
		require.NoError(t, err)
		assert.Equal(t, uwA.ID, got)
	})

	t.Run("重复提交幂等短路", func(t *testing.T) {
		_, err := repo.UpdateItemIfPending(ctx, session.ID, itemA.ID, domain.ReviewResultRemembered, now)
		assert.ErrorIs(t, err, ErrNotFound, "已作答的 item 不再命中条件更新")
	})

	t.Run("session 归属不匹配不命中", func(t *testing.T) {
		_, err := repo.UpdateItemIfPending(ctx, otherSession.ID, itemB.ID, domain.ReviewResultRemembered, now)
		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("提交后状态与 reviewed_at 写回", func(t *testing.T) {
		_, err := repo.UpdateItemIfPending(ctx, session.ID, itemB.ID, domain.ReviewResultForgotten, now.Add(time.Second))
		require.NoError(t, err)
		items, err := repo.ListSessionItems(ctx, session.ID)
		require.NoError(t, err)
		byID := map[uuid.UUID]domain.ReviewResult{}
		for _, row := range items {
			byID[row.Item.ID] = row.Item.Result
			if row.Item.ID == itemB.ID {
				require.NotNil(t, row.Item.ReviewedAt)
			}
		}
		assert.Equal(t, domain.ReviewResultRemembered, byID[itemA.ID])
		assert.Equal(t, domain.ReviewResultForgotten, byID[itemB.ID])
	})

	t.Run("计数只按实际作答累计", func(t *testing.T) {
		remembered, forgotten, pending, err := repo.CountResults(ctx, session.ID)
		require.NoError(t, err)
		assert.Equal(t, 1, remembered)
		assert.Equal(t, 1, forgotten)
		assert.Zero(t, pending, "两个 item 都已作答")
	})
}

func TestIntegration_ReviewRepo_CompleteAndAbandon(t *testing.T) {
	db := requireTestDB(t)
	resetTables(t)
	ctx := context.Background()
	now := microNow()
	repo := NewReviewRepo(db)

	session, err := domain.NewReviewSession(5, 2, now)
	require.NoError(t, err)
	require.NoError(t, repo.InsertSession(ctx, session))
	_, uwA := seedWord(t, "a")
	_, uwB := seedWord(t, "b")
	itemA := seedItem(t, session.ID, uwA.ID, 0)
	itemB := seedItem(t, session.ID, uwB.ID, 1)

	_, err = repo.UpdateItemIfPending(ctx, session.ID, itemA.ID, domain.ReviewResultRemembered, now)
	require.NoError(t, err)
	_, err = repo.UpdateItemIfPending(ctx, session.ID, itemB.ID, domain.ReviewResultForgotten, now.Add(time.Second))
	require.NoError(t, err)

	t.Run("全部作答后完成 session 并回写汇总", func(t *testing.T) {
		remembered, forgotten, pending, err := repo.CountResults(ctx, session.ID)
		require.NoError(t, err)
		assert.Zero(t, pending)

		got, err := repo.GetSessionByID(ctx, session.ID)
		require.NoError(t, err)
		require.NoError(t, got.Complete(remembered, forgotten, now))
		require.NoError(t, repo.UpdateSession(ctx, got))

		final, err := repo.GetSessionByID(ctx, session.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.ReviewStatusCompleted, final.Status)
		assert.Equal(t, 1, final.RememberedCount)
		assert.Equal(t, 1, final.ForgottenCount)
		require.NotNil(t, final.CompletedAt)
		assert.True(t, got.CompletedAt.Equal(*final.CompletedAt))
	})

	// 两个 D010 子测试共用这一条 active session：先验证唯一索引拒绝第二个，
	// 再把它放弃、验证可以创建新的 active session。
	active := seedSession(t, 1, 1, domain.ReviewStatusActive)

	t.Run("部分唯一索引拒绝第二个 active session", func(t *testing.T) {
		conflict, err := domain.NewReviewSession(1, 1, now)
		require.NoError(t, err)
		err = repo.InsertSession(ctx, conflict)
		assert.ErrorIs(t, err, ErrAlreadyExists, "D010 最终防线")
	})

	t.Run("放弃旧轮后可创建新的 active session", func(t *testing.T) {
		got, err := repo.GetSessionByID(ctx, active.ID)
		require.NoError(t, err)
		require.NoError(t, got.Abandon(now))
		require.NoError(t, repo.UpdateSession(ctx, got))

		abandoned, err := repo.GetSessionByID(ctx, active.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.ReviewStatusAbandoned, abandoned.Status)
		assert.Nil(t, abandoned.CompletedAt, "abandoned 不是完成，completed_at 保持为空")

		replacement, err := domain.NewReviewSession(1, 1, now)
		require.NoError(t, err)
		require.NoError(t, repo.InsertSession(ctx, replacement), "旧轮放弃后允许创建新的 active session")

		locked, err := repo.LockActiveSession(ctx)
		require.NoError(t, err)
		assert.Equal(t, replacement.ID, locked.ID)
	})
}

func TestIntegration_ReviewRepo_Locks(t *testing.T) {
	db := requireTestDB(t)
	resetTables(t)
	ctx := context.Background()
	repo := NewReviewRepo(db)

	t.Run("无 active session 时返回 ErrNotFound", func(t *testing.T) {
		_, err := repo.LockActiveSession(ctx)
		assert.ErrorIs(t, err, ErrNotFound)
	})

	session := seedSession(t, 5, 2, domain.ReviewStatusActive)
	_, uw := seedWord(t, "lock")
	item := seedItem(t, session.ID, uw.ID, 0)

	t.Run("锁定 active session", func(t *testing.T) {
		got, err := repo.LockActiveSession(ctx)
		require.NoError(t, err)
		assert.Equal(t, session.ID, got.ID)
	})

	t.Run("按 ID 锁定 session 与 item", func(t *testing.T) {
		gotSession, err := repo.LockSessionByID(ctx, session.ID)
		require.NoError(t, err)
		assert.Equal(t, session.ID, gotSession.ID)

		gotItem, err := repo.LockItem(ctx, session.ID, item.ID)
		require.NoError(t, err)
		assert.Equal(t, item.ID, gotItem.ID)

		_, err = repo.LockItem(ctx, uuid.Must(uuid.NewV7()), item.ID)
		assert.ErrorIs(t, err, ErrNotFound, "归属不匹配视为未命中")
		_, err = repo.LockSessionByID(ctx, uuid.Must(uuid.NewV7()))
		assert.ErrorIs(t, err, ErrNotFound)
	})
}

func TestIntegration_ReviewRepo_AdvisoryLockSerializes(t *testing.T) {
	db := requireTestDB(t)
	resetTables(t)
	ctx := context.Background()

	tx1, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer func() { _ = tx1.Rollback() }()
	require.NoError(t, NewReviewRepo(tx1).AcquireActiveSessionLock(ctx))

	// 锁被持有期间，第二个事务的获取会阻塞到上下文超时。
	waitCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	tx2, err := db.BeginTx(waitCtx, nil)
	require.NoError(t, err)
	defer func() { _ = tx2.Rollback() }()
	err = NewReviewRepo(tx2).AcquireActiveSessionLock(waitCtx)
	require.Error(t, err, "持有者未释放前应阻塞直至超时")

	// 持有者释放后（事务回滚自动释放 advisory lock），新事务可获取同一把锁。
	require.NoError(t, tx1.Rollback())
	tx3, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer func() { _ = tx3.Rollback() }()
	assert.NoError(t, NewReviewRepo(tx3).AcquireActiveSessionLock(ctx))
}

func TestIntegration_ReviewRepo_FKViolation(t *testing.T) {
	db := requireTestDB(t)
	resetTables(t)
	ctx := context.Background()
	repo := NewReviewRepo(db)

	_, uw := seedWord(t, "fk")

	t.Run("引用不存在的 session", func(t *testing.T) {
		item, err := domain.NewReviewItem(uuid.Must(uuid.NewV7()), uw.ID, 0, microNow())
		require.NoError(t, err)
		err = repo.InsertItems(ctx, []*domain.ReviewItem{item})
		assert.ErrorIs(t, err, ErrReferenceViolation)
	})

	t.Run("引用不存在的 user_word", func(t *testing.T) {
		session := seedSession(t, 1, 1, domain.ReviewStatusAbandoned)
		item, err := domain.NewReviewItem(session.ID, uuid.Must(uuid.NewV7()), 0, microNow())
		require.NoError(t, err)
		err = repo.InsertItems(ctx, []*domain.ReviewItem{item})
		assert.ErrorIs(t, err, ErrReferenceViolation)
	})
}
