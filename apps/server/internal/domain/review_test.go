package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReviewStatusConstants(t *testing.T) {
	t.Parallel()

	// 状态值即数据库与 API 中的字符串形态（docs/review/data-model.md §4）。
	assert.Equal(t, "active", string(ReviewStatusActive))
	assert.Equal(t, "completed", string(ReviewStatusCompleted))
	assert.Equal(t, "abandoned", string(ReviewStatusAbandoned))
	assert.Equal(t, "pending", string(ReviewResultPending))
	assert.Equal(t, "remembered", string(ReviewResultRemembered))
	assert.Equal(t, "forgotten", string(ReviewResultForgotten))
}

func TestNewReviewSession(t *testing.T) {
	t.Parallel()

	cst := time.FixedZone("CST", 8*3600)
	now := time.Date(2026, 9, 6, 12, 30, 0, 0, cst)

	t.Run("创建 active session 初始化汇总字段", func(t *testing.T) {
		t.Parallel()

		session, err := NewReviewSession(50, 23, now)
		require.NoError(t, err)

		assert.Equal(t, uuid.Version(7), session.ID.Version())
		assert.Equal(t, 50, session.RequestedCount)
		assert.Equal(t, 23, session.TotalCount)
		assert.Zero(t, session.RememberedCount)
		assert.Zero(t, session.ForgottenCount)
		assert.Equal(t, ReviewStatusActive, session.Status)
		assert.Nil(t, session.CompletedAt)
		assert.Equal(t, now.UTC(), session.StartedAt)
		assert.Equal(t, now.UTC(), session.CreatedAt)
		assert.True(t, session.StartedAt.Location() == time.UTC)
	})

	tests := []struct {
		name           string
		requestedCount int
		totalCount     int
	}{
		{name: "请求数量为零不合法", requestedCount: 0, totalCount: 0},
		{name: "请求数量为负不合法", requestedCount: -1, totalCount: 0},
		{name: "实际抽取数量为零不合法", requestedCount: 10, totalCount: 0},
		{name: "实际抽取数量不能超过请求数量", requestedCount: 10, totalCount: 11},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			session, err := NewReviewSession(tt.requestedCount, tt.totalCount, now)
			require.ErrorIs(t, err, ErrInvalidSessionCounts)
			assert.Nil(t, session)
		})
	}
}

func TestReviewSession_Complete(t *testing.T) {
	t.Parallel()

	cst := time.FixedZone("CST", 8*3600)
	now := time.Date(2026, 9, 6, 12, 30, 0, 0, cst)

	newActiveSession := func(t *testing.T) *ReviewSession {
		t.Helper()
		session, err := NewReviewSession(30, 3, now)
		require.NoError(t, err)
		return session
	}

	t.Run("完成后回写汇总与完成时间", func(t *testing.T) {
		t.Parallel()

		session := newActiveSession(t)
		completedAt := now.Add(30 * time.Minute)

		require.NoError(t, session.Complete(2, 1, completedAt))

		assert.Equal(t, ReviewStatusCompleted, session.Status)
		assert.Equal(t, 2, session.RememberedCount)
		assert.Equal(t, 1, session.ForgottenCount)
		require.NotNil(t, session.CompletedAt)
		assert.Equal(t, completedAt.UTC(), *session.CompletedAt)
	})

	t.Run("汇总计数必须与总题数一致", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name            string
			rememberedCount int
			forgottenCount  int
		}{
			{name: "记得加忘记少于总题数", rememberedCount: 1, forgottenCount: 1},
			{name: "记得加忘记超过总题数", rememberedCount: 3, forgottenCount: 1},
			{name: "记得数量为负", rememberedCount: -1, forgottenCount: 4},
			{name: "忘记数量为负", rememberedCount: 4, forgottenCount: -1},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				session := newActiveSession(t)
				require.ErrorIs(t, session.Complete(tt.rememberedCount, tt.forgottenCount, now), ErrSessionTotalsMismatch)

				// 失败时状态机不迁移。
				assert.Equal(t, ReviewStatusActive, session.Status)
				assert.Nil(t, session.CompletedAt)
				assert.Zero(t, session.RememberedCount)
				assert.Zero(t, session.ForgottenCount)
			})
		}
	})

	t.Run("只有 active 状态可以完成", func(t *testing.T) {
		t.Parallel()

		completed := newActiveSession(t)
		require.NoError(t, completed.Complete(3, 0, now))
		require.ErrorIs(t, completed.Complete(3, 0, now), ErrSessionNotActive)

		abandoned := newActiveSession(t)
		require.NoError(t, abandoned.Abandon(now))
		require.ErrorIs(t, abandoned.Complete(3, 0, now), ErrSessionNotActive)
		assert.Equal(t, ReviewStatusAbandoned, abandoned.Status)
	})
}

func TestReviewSession_Abandon(t *testing.T) {
	t.Parallel()

	cst := time.FixedZone("CST", 8*3600)
	now := time.Date(2026, 9, 6, 12, 30, 0, 0, cst)

	t.Run("active session 放弃后不写完成时间", func(t *testing.T) {
		t.Parallel()

		session, err := NewReviewSession(30, 3, now)
		require.NoError(t, err)

		require.NoError(t, session.Abandon(now))

		assert.Equal(t, ReviewStatusAbandoned, session.Status)
		assert.Nil(t, session.CompletedAt)
	})

	t.Run("只有 active 状态可以放弃", func(t *testing.T) {
		t.Parallel()

		session, err := NewReviewSession(30, 3, now)
		require.NoError(t, err)
		require.NoError(t, session.Abandon(now))
		require.ErrorIs(t, session.Abandon(now), ErrSessionNotActive)

		completed, err := NewReviewSession(30, 3, now)
		require.NoError(t, err)
		require.NoError(t, completed.Complete(3, 0, now))
		require.ErrorIs(t, completed.Abandon(now), ErrSessionNotActive)
		assert.Equal(t, ReviewStatusCompleted, completed.Status)
	})
}

func TestNewReviewItem(t *testing.T) {
	t.Parallel()

	cst := time.FixedZone("CST", 8*3600)
	now := time.Date(2026, 9, 6, 12, 30, 0, 0, cst)

	t.Run("创建 pending item 初始化字段", func(t *testing.T) {
		t.Parallel()

		sessionID := uuid.Must(uuid.NewV7())
		userWordID := uuid.Must(uuid.NewV7())

		item, err := NewReviewItem(sessionID, userWordID, 2, now)
		require.NoError(t, err)

		assert.Equal(t, uuid.Version(7), item.ID.Version())
		assert.Equal(t, sessionID, item.SessionID)
		assert.Equal(t, userWordID, item.UserWordID)
		assert.Equal(t, 2, item.Position)
		assert.Equal(t, ReviewResultPending, item.Result)
		assert.Nil(t, item.ReviewedAt)
		assert.Equal(t, now.UTC(), item.CreatedAt)
	})

	t.Run("position 为负不合法", func(t *testing.T) {
		t.Parallel()

		item, err := NewReviewItem(uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), -1, now)
		require.ErrorIs(t, err, ErrInvalidPosition)
		assert.Nil(t, item)
	})
}

func TestReviewItem_Submit(t *testing.T) {
	t.Parallel()

	cst := time.FixedZone("CST", 8*3600)
	now := time.Date(2026, 9, 6, 12, 30, 0, 0, cst)

	newPendingItem := func(t *testing.T) *ReviewItem {
		t.Helper()
		item, err := NewReviewItem(uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), 0, now)
		require.NoError(t, err)
		return item
	}

	t.Run("pending 记得后记录作答时间", func(t *testing.T) {
		t.Parallel()

		item := newPendingItem(t)
		answeredAt := now.Add(5 * time.Minute)

		require.NoError(t, item.Submit(ReviewResultRemembered, answeredAt))

		assert.Equal(t, ReviewResultRemembered, item.Result)
		require.NotNil(t, item.ReviewedAt)
		assert.Equal(t, answeredAt.UTC(), *item.ReviewedAt)
	})

	t.Run("pending 忘记后记录作答时间", func(t *testing.T) {
		t.Parallel()

		item := newPendingItem(t)

		require.NoError(t, item.Submit(ReviewResultForgotten, now))

		assert.Equal(t, ReviewResultForgotten, item.Result)
		require.NotNil(t, item.ReviewedAt)
	})

	t.Run("已作答的 item 不能再次提交", func(t *testing.T) {
		t.Parallel()

		item := newPendingItem(t)
		require.NoError(t, item.Submit(ReviewResultRemembered, now))
		firstReviewedAt := *item.ReviewedAt

		require.ErrorIs(t, item.Submit(ReviewResultForgotten, now), ErrItemAlreadyAnswered)

		// 幂等短路：状态停留在首次作答结果。
		assert.Equal(t, ReviewResultRemembered, item.Result)
		assert.Equal(t, firstReviewedAt, *item.ReviewedAt)
	})

	t.Run("pending 与未知结果不是有效作答", func(t *testing.T) {
		t.Parallel()

		for _, result := range []ReviewResult{ReviewResultPending, ReviewResult("bogus")} {
			item := newPendingItem(t)

			require.ErrorIs(t, item.Submit(result, now), ErrInvalidReviewResult)

			assert.Equal(t, ReviewResultPending, item.Result)
			assert.Nil(t, item.ReviewedAt)
		}
	})
}
