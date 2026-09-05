package domain

import (
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUserWord(t *testing.T) {
	t.Parallel()

	cst := time.FixedZone("CST", 8*3600)
	now := time.Date(2026, 9, 6, 12, 30, 0, 0, cst)

	t.Run("创建生词行初始化学习字段", func(t *testing.T) {
		t.Parallel()

		entryID := uuid.Must(uuid.NewV7())
		word, err := NewUserWord(entryID, 3, now)
		require.NoError(t, err)

		assert.Equal(t, uuid.Version(7), word.ID.Version())
		assert.Equal(t, entryID, word.DictionaryEntryID)
		assert.Equal(t, 3, word.EncounterCount)
		assert.Zero(t, word.ReviewCount)
		assert.Zero(t, word.RememberCount)
		assert.Zero(t, word.ForgetCount)
		assert.Zero(t, word.CurrentStreak)
		assert.Nil(t, word.LastReviewedAt)
		assert.Nil(t, word.CustomReviewMeaning)
		assert.Nil(t, word.DeletedAt)
		assert.Equal(t, now.UTC(), word.CreatedAt)
		assert.Equal(t, now.UTC(), word.UpdatedAt)
		assert.True(t, word.CreatedAt.Location() == time.UTC)
	})

	tests := []struct {
		name           string
		encounterCount int
	}{
		{name: "遇词次数为零不合法", encounterCount: 0},
		{name: "遇词次数为负不合法", encounterCount: -2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			word, err := NewUserWord(uuid.Must(uuid.NewV7()), tt.encounterCount, now)
			require.ErrorIs(t, err, ErrInvalidEncounterCount)
			assert.Nil(t, word)
		})
	}
}

func TestUserWord_ApplyReview(t *testing.T) {
	t.Parallel()

	cst := time.FixedZone("CST", 8*3600)
	now := time.Date(2026, 9, 6, 12, 30, 0, 0, cst)

	tests := []struct {
		name          string
		currentStreak int
		result        ReviewResult
		wantStreak    int
		wantRemember  int
		wantForget    int
	}{
		{name: "首次记得，streak 从 0 变 1", currentStreak: 0, result: ReviewResultRemembered, wantStreak: 1, wantRemember: 1},
		{name: "连续记得继续累加", currentStreak: 3, result: ReviewResultRemembered, wantStreak: 4, wantRemember: 1},
		{name: "连续忘记后记得，streak 重置为 1", currentStreak: -2, result: ReviewResultRemembered, wantStreak: 1, wantRemember: 1},
		{name: "记得之后忘记，streak 翻转为 -1", currentStreak: 3, result: ReviewResultForgotten, wantStreak: -1, wantForget: 1},
		{name: "连续忘记继续累加", currentStreak: -1, result: ReviewResultForgotten, wantStreak: -2, wantForget: 1},
		{name: "streak 为 0 时忘记从 -1 开始", currentStreak: 0, result: ReviewResultForgotten, wantStreak: -1, wantForget: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			word, err := NewUserWord(uuid.Must(uuid.NewV7()), 1, now)
			require.NoError(t, err)
			word.CurrentStreak = tt.currentStreak

			require.NoError(t, word.ApplyReview(tt.result, now.Add(time.Hour)))

			assert.Equal(t, 1, word.ReviewCount)
			assert.Equal(t, tt.wantRemember, word.RememberCount)
			assert.Equal(t, tt.wantForget, word.ForgetCount)
			assert.Equal(t, tt.wantStreak, word.CurrentStreak)

			reviewedAt := now.Add(time.Hour).UTC()
			require.NotNil(t, word.LastReviewedAt)
			assert.Equal(t, reviewedAt, *word.LastReviewedAt)
			assert.Equal(t, reviewedAt, word.UpdatedAt)
		})
	}

	t.Run("累计计数在已有基础上递增", func(t *testing.T) {
		t.Parallel()

		word, err := NewUserWord(uuid.Must(uuid.NewV7()), 1, now)
		require.NoError(t, err)
		word.ReviewCount = 5
		word.RememberCount = 3
		word.ForgetCount = 2

		require.NoError(t, word.ApplyReview(ReviewResultForgotten, now))

		assert.Equal(t, 6, word.ReviewCount)
		assert.Equal(t, 3, word.RememberCount)
		assert.Equal(t, 3, word.ForgetCount)
	})

	t.Run("pending 与未知结果不是有效作答", func(t *testing.T) {
		t.Parallel()

		for _, result := range []ReviewResult{ReviewResultPending, ReviewResult("bogus")} {
			word, err := NewUserWord(uuid.Must(uuid.NewV7()), 1, now)
			require.NoError(t, err)
			word.CurrentStreak = 2

			require.ErrorIs(t, word.ApplyReview(result, now), ErrInvalidReviewResult)

			// 失败时不得改动任何累计字段。
			assert.Zero(t, word.ReviewCount)
			assert.Zero(t, word.RememberCount)
			assert.Zero(t, word.ForgetCount)
			assert.Equal(t, 2, word.CurrentStreak)
			assert.Nil(t, word.LastReviewedAt)
		}
	})
}

func TestUserWord_ReviewWeight(t *testing.T) {
	t.Parallel()

	cst := time.FixedZone("CST", 8*3600)
	now := time.Date(2026, 9, 6, 12, 30, 0, 0, cst)

	newReviewedWord := func(t *testing.T, mutate func(*UserWord)) *UserWord {
		t.Helper()
		word, err := NewUserWord(uuid.Must(uuid.NewV7()), 1, now)
		require.NoError(t, err)
		if mutate != nil {
			mutate(word)
		}
		return word
	}

	t.Run("新词有保底权重", func(t *testing.T) {
		t.Parallel()

		// 1(base) + log2(2)(encounter) + 0.5×4(difficulty 未知) + 0(overdue) + 2(new word bonus) = 6。
		word := newReviewedWord(t, nil)
		assert.InDelta(t, 6.0, word.ReviewWeight(now), 1e-9)
	})

	t.Run("文档最终示例约为 8.83", func(t *testing.T) {
		t.Parallel()

		// docs/review/algorithm.md §5：encounter 5 / review 6 / remember 2 /
		// streak -2 / 距上次复习 4 天 → ≈ 8.83。
		word := newReviewedWord(t, func(w *UserWord) {
			w.EncounterCount = 5
			w.ReviewCount = 6
			w.RememberCount = 2
			w.ForgetCount = 4
			w.CurrentStreak = -2
			last := now.Add(-96 * time.Hour).UTC()
			w.LastReviewedAt = &last
		})
		assert.InDelta(t, 8.83, word.ReviewWeight(now), 0.01)
	})

	t.Run("连续记得四次降权 1", func(t *testing.T) {
		t.Parallel()

		// 1(base) + log2(4)(encounter) + 0(difficulty) + -1(streak ≥ 4) + 0(overdue) = 2。
		word := newReviewedWord(t, func(w *UserWord) {
			w.EncounterCount = 3
			w.ReviewCount = 4
			w.RememberCount = 4
			w.CurrentStreak = 4
		})
		assert.InDelta(t, 2.0, word.ReviewWeight(now), 1e-9)
	})

	t.Run("时间因素按天增长并封顶 3", func(t *testing.T) {
		t.Parallel()

		// 1(base) + 1(encounter) + 2(difficulty 0.5) + 0(streak) + 3(overdue 封顶) = 7。
		word := newReviewedWord(t, func(w *UserWord) {
			w.ReviewCount = 2
			w.RememberCount = 1
			w.ForgetCount = 1
			last := now.AddDate(0, 0, -30).UTC()
			w.LastReviewedAt = &last
		})
		assert.InDelta(t, 7.0, word.ReviewWeight(now), 1e-9)
	})

	t.Run("一周未复习约加 1", func(t *testing.T) {
		t.Parallel()

		// 1(base) + 1(encounter) + 2(difficulty 0.5) + 0(streak) + 1(7 天 overdue) = 5。
		word := newReviewedWord(t, func(w *UserWord) {
			w.ReviewCount = 2
			w.RememberCount = 1
			w.ForgetCount = 1
			last := now.Add(-7 * 24 * time.Hour).UTC()
			w.LastReviewedAt = &last
		})
		assert.InDelta(t, 5.0, word.ReviewWeight(now), 1e-9)
	})

	t.Run("LastReviewedAt 在未来时时间因素不产生负权重", func(t *testing.T) {
		t.Parallel()

		word := newReviewedWord(t, func(w *UserWord) {
			w.ReviewCount = 2
			w.RememberCount = 1
			w.ForgetCount = 1
			last := now.Add(24 * time.Hour).UTC()
			w.LastReviewedAt = &last
		})
		assert.False(t, math.IsNaN(word.ReviewWeight(now)))
		assert.GreaterOrEqual(t, word.ReviewWeight(now), 0.0)
	})
}

func TestUserWord_MasteryScore(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		rememberCount int
		reviewCount   int
		want          float64
	}{
		{name: "未复习为初始 50%", rememberCount: 0, reviewCount: 0, want: 50},
		{name: "首次记得约 66.7%", rememberCount: 1, reviewCount: 1, want: 66.67},
		{name: "首次忘记约 33.3%", rememberCount: 0, reviewCount: 1, want: 33.33},
		{name: "多数记得接近但不到 100%", rememberCount: 9, reviewCount: 10, want: 83.33},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			word, err := NewUserWord(uuid.Must(uuid.NewV7()), 1, time.Unix(0, 0))
			require.NoError(t, err)
			word.ReviewCount = tt.reviewCount
			word.RememberCount = tt.rememberCount

			assert.InDelta(t, tt.want, word.MasteryScore(), 0.01)
		})
	}
}
