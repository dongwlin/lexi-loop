package v1

// ReviewService 集成测试：真实 PostgreSQL 上的复习用例（docs/api/reviews.md）
// 与 structure.md §5.4 必测场景——并发开始两轮后全库只有一个 active session、
// 同一 item 并发提交只累计一次、最后两个 item 并发提交后 session 必为
// completed、item 更新后注入失败整体回滚、创建 items 中途失败不留半成品
// session。失败注入使用测试库专用的临时触发器（t.Cleanup 移除），
// 不触碰 migrations。

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dongwlin/lexi-loop/apps/server/internal/apperr"
	"github.com/dongwlin/lexi-loop/apps/server/internal/domain"
	"github.com/dongwlin/lexi-loop/apps/server/internal/repo"
	"github.com/dongwlin/lexi-loop/apps/server/internal/service"
)

func TestIntegration_ReviewService_StartSession(t *testing.T) {
	ctx := context.Background()

	t.Run("候选为 0 时不创建 session 并返回业务前置错误", func(t *testing.T) {
		resetTables(t)
		svc := newTestReview(t)

		_, err := svc.StartSession(ctx, service.StartSessionRequest{Count: 5})
		require.Error(t, err)
		var appErr *apperr.Error
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, apperr.FailedPrecondition, appErr.Kind)
		assert.Equal(t, apperr.CodeNoReviewableWords, appErr.Code)
		assert.Equal(t, "no reviewable words available", appErr.Message)
		assertSessionCount(t, 0)
	})

	t.Run("请求数量超过候选数时按实际数量截断", func(t *testing.T) {
		resetTables(t)
		seedWords(t, 3)
		svc := newTestReview(t)

		started, err := svc.StartSession(ctx, service.StartSessionRequest{Count: 50})
		require.NoError(t, err)
		assert.Equal(t, 50, started.RequestedCount)
		assert.Equal(t, 3, started.TotalCount, "D009：total = min(count, available)")
		require.Len(t, started.Items, 3)
		for _, item := range started.Items {
			assert.NotEqual(t, uuid.Nil, item.ItemID)
			assert.NotEmpty(t, item.Headword)
		}

		session, err := repo.NewReviewRepo(testDB).GetSessionByID(ctx, started.SessionID)
		require.NoError(t, err)
		assert.Equal(t, domain.ReviewStatusActive, session.Status)
		assert.Equal(t, 50, session.RequestedCount)
		assert.Equal(t, 3, session.TotalCount)
		items, err := repo.NewReviewRepo(testDB).ListSessionItems(ctx, started.SessionID)
		require.NoError(t, err)
		require.Len(t, items, 3)
		for i, row := range items {
			assert.Equal(t, i, row.Item.Position)
			assert.Equal(t, domain.ReviewResultPending, row.Item.Result)
			assert.Equal(t, started.Items[i].ItemID, row.Item.ID)
		}
	})

	t.Run("开始新一轮时放弃旧 active session", func(t *testing.T) {
		resetTables(t)
		seedWords(t, 2)
		svc := newTestReview(t)

		first := mustStart(t, svc, 1)
		second := mustStart(t, svc, 1)
		assert.NotEqual(t, first.SessionID, second.SessionID)

		oldSession, err := repo.NewReviewRepo(testDB).GetSessionByID(ctx, first.SessionID)
		require.NoError(t, err)
		assert.Equal(t, domain.ReviewStatusAbandoned, oldSession.Status)
		assert.Nil(t, oldSession.CompletedAt, "abandoned 不是完成")

		newSession, err := repo.NewReviewRepo(testDB).GetSessionByID(ctx, second.SessionID)
		require.NoError(t, err)
		assert.Equal(t, domain.ReviewStatusActive, newSession.Status)
	})

	t.Run("并发开始两轮后全库只有一个 active session", func(t *testing.T) {
		resetTables(t)
		seedWords(t, 3)
		svc := newTestReview(t)

		type outcome struct {
			started *service.StartSessionResult
			err     error
		}
		outcomes := make(chan outcome, 2)
		for i := 0; i < 2; i++ {
			go func() {
				started, err := svc.StartSession(context.Background(), service.StartSessionRequest{Count: 10})
				outcomes <- outcome{started, err}
			}()
		}
		sessionIDs := make([]uuid.UUID, 0, 2)
		for i := 0; i < 2; i++ {
			o := <-outcomes
			require.NoError(t, o.err, "并发开始不应互相失败：advisory lock 串行化整个流程")
			require.NotNil(t, o.started)
			assert.Equal(t, 3, o.started.TotalCount)
			sessionIDs = append(sessionIDs, o.started.SessionID)
		}

		assertSessionCount(t, 2)
		assertItemCount(t, 6, "两轮各自创建全部 items")
		statuses := map[domain.ReviewStatus]int{}
		for _, id := range sessionIDs {
			session, err := repo.NewReviewRepo(testDB).GetSessionByID(ctx, id)
			require.NoError(t, err)
			statuses[session.Status]++
		}
		assert.Equal(t, 1, statuses[domain.ReviewStatusActive], "全库至多一个 active session（D010）")
		assert.Equal(t, 1, statuses[domain.ReviewStatusAbandoned])
	})

	t.Run("创建 items 中途失败不留下半成品 session", func(t *testing.T) {
		resetTables(t)
		seedWords(t, 3)
		svc := newTestReview(t)
		// 第二个 item（position >= 1）插入时注入真实数据库失败，
		// 验证同一事务内的 session 与其余 items 一起回滚。
		installTrigger(t, "service_test_mid_batch_items",
			`BEGIN IF NEW.position >= 1 THEN RAISE EXCEPTION 'injected failure during items insert'; END IF; RETURN NEW; END`)

		_, err := svc.StartSession(ctx, service.StartSessionRequest{Count: 3})
		require.Error(t, err)
		assertSessionCount(t, 0)
		assertItemCount(t, 0, "不应留下任何 items")
	})

	t.Run("序列化失败重试上限后返回冲突且不留残留", func(t *testing.T) {
		resetTables(t)
		seedWords(t, 3)
		svc := newTestReview(t)
		// 每次插入 items 都以 SQLSTATE 40001 失败：runReplayableTx 重试
		// 整个事务至上限，最终映射为 Conflict / BASE.BIZ.CONCURRENT_UPDATE。
		installTrigger(t, "service_test_serial_fail_items",
			`BEGIN RAISE EXCEPTION 'injected serialization failure' USING ERRCODE = '40001'; END`)

		_, err := svc.StartSession(ctx, service.StartSessionRequest{Count: 3})
		require.Error(t, err)
		var appErr *apperr.Error
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, apperr.Conflict, appErr.Kind)
		assert.Equal(t, apperr.CodeConcurrentUpdate, appErr.Code)
		assertSessionCount(t, 0)
		assertItemCount(t, 0, "不应留下任何 items")
	})
}

func TestIntegration_ReviewService_SubmitResult(t *testing.T) {
	ctx := context.Background()

	t.Run("提交记得后完成单题 session 并累计词级数据", func(t *testing.T) {
		resetTables(t)
		seedWords(t, 1)
		svc := newTestReview(t)
		started := mustStart(t, svc, 1)

		require.NoError(t, svc.SubmitResult(ctx, service.SubmitResultRequest{
			SessionID: started.SessionID,
			ItemID:    started.Items[0].ItemID,
			Result:    domain.ReviewResultRemembered,
		}))

		item, err := repo.NewReviewRepo(testDB).LockItem(ctx, started.SessionID, started.Items[0].ItemID)
		require.NoError(t, err)
		assert.Equal(t, domain.ReviewResultRemembered, item.Result)
		assert.NotNil(t, item.ReviewedAt)

		word, err := repo.NewUserWordRepo(testDB).FindByID(ctx, itemUserWordID(t, item.ID))
		require.NoError(t, err)
		assert.Equal(t, 1, word.ReviewCount)
		assert.Equal(t, 1, word.RememberCount)
		assert.Equal(t, 0, word.ForgetCount)
		assert.Equal(t, 1, word.CurrentStreak)
		assert.NotNil(t, word.LastReviewedAt)

		session, err := repo.NewReviewRepo(testDB).GetSessionByID(ctx, started.SessionID)
		require.NoError(t, err)
		assert.Equal(t, domain.ReviewStatusCompleted, session.Status, "唯一 item 已作答即完成")
		assert.Equal(t, 1, session.RememberedCount)
		assert.Equal(t, 0, session.ForgottenCount)
		assert.NotNil(t, session.CompletedAt)
	})

	t.Run("提交不记得按 forget 累计且 session 保持 active", func(t *testing.T) {
		resetTables(t)
		seedWords(t, 2)
		svc := newTestReview(t)
		started := mustStart(t, svc, 2)

		require.NoError(t, svc.SubmitResult(ctx, service.SubmitResultRequest{
			SessionID: started.SessionID,
			ItemID:    started.Items[0].ItemID,
			Result:    domain.ReviewResultForgotten,
		}))

		word, err := repo.NewUserWordRepo(testDB).FindByID(ctx, itemUserWordID(t, started.Items[0].ItemID))
		require.NoError(t, err)
		assert.Equal(t, 1, word.ReviewCount)
		assert.Equal(t, 0, word.RememberCount)
		assert.Equal(t, 1, word.ForgetCount)
		assert.Equal(t, -1, word.CurrentStreak)

		session, err := repo.NewReviewRepo(testDB).GetSessionByID(ctx, started.SessionID)
		require.NoError(t, err)
		assert.Equal(t, domain.ReviewStatusActive, session.Status, "仍有 pending item，不触发完成")
	})

	t.Run("重复提交幂等返回成功且不重复计数", func(t *testing.T) {
		resetTables(t)
		seedWords(t, 2)
		svc := newTestReview(t)
		started := mustStart(t, svc, 2)
		req := service.SubmitResultRequest{
			SessionID: started.SessionID,
			ItemID:    started.Items[0].ItemID,
			Result:    domain.ReviewResultRemembered,
		}

		require.NoError(t, svc.SubmitResult(ctx, req))
		require.NoError(t, svc.SubmitResult(ctx, req), "重复提交按契约幂等返回")

		word, err := repo.NewUserWordRepo(testDB).FindByID(ctx, itemUserWordID(t, req.ItemID))
		require.NoError(t, err)
		assert.Equal(t, 1, word.ReviewCount, "只累计一次")
	})

	t.Run("并发提交同一 item 只累计一次", func(t *testing.T) {
		resetTables(t)
		seedWords(t, 1)
		svc := newTestReview(t)
		started := mustStart(t, svc, 1)

		const racers = 8
		errs := make(chan error, racers)
		for i := 0; i < racers; i++ {
			result := domain.ReviewResultRemembered
			if i%2 == 1 {
				result = domain.ReviewResultForgotten
			}
			go func(result domain.ReviewResult) {
				errs <- svc.SubmitResult(context.Background(), service.SubmitResultRequest{
					SessionID: started.SessionID,
					ItemID:    started.Items[0].ItemID,
					Result:    result,
				})
			}(result)
		}
		for i := 0; i < racers; i++ {
			assert.NoError(t, <-errs, "未抢到的提交按契约幂等返回成功")
		}

		word, err := repo.NewUserWordRepo(testDB).FindByID(ctx, itemUserWordID(t, started.Items[0].ItemID))
		require.NoError(t, err)
		assert.Equal(t, 1, word.ReviewCount, "并发重复提交只累计一次")
		require.Equal(t, 1, word.RememberCount+word.ForgetCount)
		item, err := repo.NewReviewRepo(testDB).LockItem(ctx, started.SessionID, started.Items[0].ItemID)
		require.NoError(t, err)
		switch item.Result {
		case domain.ReviewResultRemembered:
			assert.Equal(t, 1, word.RememberCount)
			assert.Equal(t, 1, word.CurrentStreak)
		case domain.ReviewResultForgotten:
			assert.Equal(t, 1, word.ForgetCount)
			assert.Equal(t, -1, word.CurrentStreak)
		default:
			t.Fatalf("item 结果异常: %q", item.Result)
		}
	})

	t.Run("归属不匹配与不存在的目标幂等短路", func(t *testing.T) {
		resetTables(t)
		seedWords(t, 2)
		svc := newTestReview(t)
		first := mustStart(t, svc, 1) // 第二轮开始会放弃此轮，但 items 仍 pending
		second := mustStart(t, svc, 1)

		// 用 first 轮的 URL 提交 second 轮的 item：归属不匹配 → 幂等短路。
		require.NoError(t, svc.SubmitResult(ctx, service.SubmitResultRequest{
			SessionID: first.SessionID,
			ItemID:    second.Items[0].ItemID,
			Result:    domain.ReviewResultRemembered,
		}))

		item, err := repo.NewReviewRepo(testDB).LockItem(ctx, second.SessionID, second.Items[0].ItemID)
		require.NoError(t, err)
		assert.Equal(t, domain.ReviewResultPending, item.Result, "错配提交不修改目标 item")
		word, err := repo.NewUserWordRepo(testDB).FindByID(ctx, itemUserWordID(t, item.ID))
		require.NoError(t, err)
		assert.Equal(t, 0, word.ReviewCount, "错配提交不修改统计")

		// 不存在的 session / item：同样幂等短路，不报错。
		require.NoError(t, svc.SubmitResult(ctx, service.SubmitResultRequest{
			SessionID: uuid.Must(uuid.NewV7()),
			ItemID:    uuid.Must(uuid.NewV7()),
			Result:    domain.ReviewResultRemembered,
		}))
	})

	t.Run("非法 result 返回参数错误", func(t *testing.T) {
		resetTables(t)
		seedWords(t, 1)
		svc := newTestReview(t)
		started := mustStart(t, svc, 1)

		err := svc.SubmitResult(ctx, service.SubmitResultRequest{
			SessionID: started.SessionID,
			ItemID:    started.Items[0].ItemID,
			Result:    domain.ReviewResultPending,
		})
		require.Error(t, err)
		var appErr *apperr.Error
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, apperr.InvalidArgument, appErr.Kind)
		assert.Equal(t, apperr.CodeValidationFailed, appErr.Code)
	})

	t.Run("最后两个 item 并发提交后 session 必为 completed", func(t *testing.T) {
		resetTables(t)
		seedWords(t, 2)
		svc := newTestReview(t)
		started := mustStart(t, svc, 2)

		var wg sync.WaitGroup
		errs := make(chan error, 2)
		results := []domain.ReviewResult{domain.ReviewResultRemembered, domain.ReviewResultForgotten}
		for i, item := range started.Items {
			wg.Add(1)
			go func(itemID uuid.UUID, result domain.ReviewResult) {
				defer wg.Done()
				errs <- svc.SubmitResult(context.Background(), service.SubmitResultRequest{
					SessionID: started.SessionID,
					ItemID:    itemID,
					Result:    result,
				})
			}(item.ItemID, results[i])
		}
		wg.Wait()
		close(errs)
		for err := range errs {
			assert.NoError(t, err)
		}

		session, err := repo.NewReviewRepo(testDB).GetSessionByID(ctx, started.SessionID)
		require.NoError(t, err)
		assert.Equal(t, domain.ReviewStatusCompleted, session.Status,
			"session 行锁串行化后不会漏掉最后一题的完成")
		assert.Equal(t, 1, session.RememberedCount)
		assert.Equal(t, 1, session.ForgottenCount)
		assert.NotNil(t, session.CompletedAt)
	})

	t.Run("item 更新后注入失败会回滚 item 与全部累计字段", func(t *testing.T) {
		resetTables(t)
		seedWords(t, 2)
		svc := newTestReview(t)
		// requested=5 > total=2：便于把 total_count 篡改为 3（仍满足
		// requested >= total 的 CHECK），使最后一题提交时的 Complete
		// 汇总校验必然失败，从而在 item 与累计写回之后注入真实失败。
		started, err := svc.StartSession(ctx, service.StartSessionRequest{Count: 5})
		require.NoError(t, err)
		require.Len(t, started.Items, 2)

		// 先正常提交第一题，作为回滚基线。
		require.NoError(t, svc.SubmitResult(ctx, service.SubmitResultRequest{
			SessionID: started.SessionID,
			ItemID:    started.Items[0].ItemID,
			Result:    domain.ReviewResultRemembered,
		}))
		_, err = testDB.ExecContext(ctx,
			"UPDATE review_sessions SET total_count = 3 WHERE id = ?", started.SessionID)
		require.NoError(t, err)

		err = svc.SubmitResult(ctx, service.SubmitResultRequest{
			SessionID: started.SessionID,
			ItemID:    started.Items[1].ItemID,
			Result:    domain.ReviewResultForgotten,
		})
		require.Error(t, err, "汇总不一致属于不变量违规，整个事务必须失败")
		var appErr *apperr.Error
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, apperr.InternalKind, appErr.Kind)

		// 回滚断言：第二题仍 pending、词级累计未写入、session 仍 active；
		// 第一题已提交的数据不受影响。
		item2, err := repo.NewReviewRepo(testDB).LockItem(ctx, started.SessionID, started.Items[1].ItemID)
		require.NoError(t, err)
		assert.Equal(t, domain.ReviewResultPending, item2.Result)
		word2, err := repo.NewUserWordRepo(testDB).FindByID(ctx, itemUserWordID(t, item2.ID))
		require.NoError(t, err)
		assert.Equal(t, 0, word2.ReviewCount)

		session, err := repo.NewReviewRepo(testDB).GetSessionByID(ctx, started.SessionID)
		require.NoError(t, err)
		assert.Equal(t, domain.ReviewStatusActive, session.Status)

		word1, err := repo.NewUserWordRepo(testDB).FindByID(ctx, itemUserWordID(t, started.Items[0].ItemID))
		require.NoError(t, err)
		assert.Equal(t, 1, word1.ReviewCount)
	})
}

func TestIntegration_ReviewService_GetSession(t *testing.T) {
	ctx := context.Background()

	t.Run("不存在时返回未找到", func(t *testing.T) {
		resetTables(t)
		svc := newTestReview(t)

		_, err := svc.GetSession(ctx, service.GetSessionRequest{SessionID: uuid.Must(uuid.NewV7())})
		require.Error(t, err)
		var appErr *apperr.Error
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, apperr.NotFound, appErr.Kind)
		assert.Equal(t, apperr.CodeNotFound, appErr.Code)
		assert.Equal(t, "review session not found", appErr.Message)
	})

	t.Run("active 状态返回与逐词结果一致的实时汇总", func(t *testing.T) {
		resetTables(t)
		seedWords(t, 3)
		svc := newTestReview(t)
		started := mustStart(t, svc, 3)

		require.NoError(t, svc.SubmitResult(ctx, service.SubmitResultRequest{
			SessionID: started.SessionID, ItemID: started.Items[0].ItemID, Result: domain.ReviewResultRemembered,
		}))
		require.NoError(t, svc.SubmitResult(ctx, service.SubmitResultRequest{
			SessionID: started.SessionID, ItemID: started.Items[1].ItemID, Result: domain.ReviewResultForgotten,
		}))

		got, err := svc.GetSession(ctx, service.GetSessionRequest{SessionID: started.SessionID})
		require.NoError(t, err)
		assert.Equal(t, started.SessionID, got.SessionID)
		assert.Equal(t, domain.ReviewStatusActive, got.Status)
		assert.Equal(t, 3, got.Total)
		assert.Equal(t, 1, got.Remembered)
		assert.Equal(t, 1, got.Forgotten)
		assert.Nil(t, got.CompletedAt)
		require.Len(t, got.Items, 3)
		assert.Equal(t, domain.ReviewResultRemembered, got.Items[0].Result)
		assert.Equal(t, domain.ReviewResultForgotten, got.Items[1].Result)
		assert.Equal(t, domain.ReviewResultPending, got.Items[2].Result)
		assert.NotEmpty(t, got.Items[0].Headword)
	})

	t.Run("完成后返回 completed 汇总", func(t *testing.T) {
		resetTables(t)
		seedWords(t, 2)
		svc := newTestReview(t)
		started := mustStart(t, svc, 2)

		require.NoError(t, svc.SubmitResult(ctx, service.SubmitResultRequest{
			SessionID: started.SessionID, ItemID: started.Items[0].ItemID, Result: domain.ReviewResultForgotten,
		}))
		require.NoError(t, svc.SubmitResult(ctx, service.SubmitResultRequest{
			SessionID: started.SessionID, ItemID: started.Items[1].ItemID, Result: domain.ReviewResultRemembered,
		}))

		got, err := svc.GetSession(ctx, service.GetSessionRequest{SessionID: started.SessionID})
		require.NoError(t, err)
		assert.Equal(t, domain.ReviewStatusCompleted, got.Status)
		assert.NotNil(t, got.CompletedAt)
		assert.Equal(t, 1, got.Remembered)
		assert.Equal(t, 1, got.Forgotten)
	})
}

func TestIntegration_ReviewService_AbandonSession(t *testing.T) {
	ctx := context.Background()

	t.Run("active → abandoned，重复放弃幂等成功", func(t *testing.T) {
		resetTables(t)
		seedWords(t, 3)
		svc := newTestReview(t)
		started := mustStart(t, svc, 3)

		require.NoError(t, svc.AbandonSession(ctx, service.AbandonSessionRequest{SessionID: started.SessionID}))
		// 重复放弃：幂等成功（api/reviews.md §6）。
		require.NoError(t, svc.AbandonSession(ctx, service.AbandonSessionRequest{SessionID: started.SessionID}))

		session, err := repo.NewReviewRepo(testDB).GetSessionByID(ctx, started.SessionID)
		require.NoError(t, err)
		assert.Equal(t, domain.ReviewStatusAbandoned, session.Status)
		assert.Nil(t, session.CompletedAt, "abandoned 不是完成")
	})

	t.Run("放弃后提交幂等短路且不累计词级统计", func(t *testing.T) {
		resetTables(t)
		seedWords(t, 2)
		svc := newTestReview(t)
		started := mustStart(t, svc, 2)

		require.NoError(t, svc.AbandonSession(ctx, service.AbandonSessionRequest{SessionID: started.SessionID}))
		// session 已非 active：提交契约上幂等短路（api/reviews.md §3），
		// 不报错也不计入已废弃的轮次。
		require.NoError(t, svc.SubmitResult(ctx, service.SubmitResultRequest{
			SessionID: started.SessionID, ItemID: started.Items[0].ItemID, Result: domain.ReviewResultRemembered,
		}))

		got, err := svc.GetSession(ctx, service.GetSessionRequest{SessionID: started.SessionID})
		require.NoError(t, err)
		assert.Equal(t, domain.ReviewStatusAbandoned, got.Status)
		assert.Equal(t, 0, got.Remembered)
		assert.Equal(t, 0, got.Forgotten)
		assert.Equal(t, int64(0), countRows(t, "SELECT count(*) FROM user_words WHERE review_count > 0"))
	})

	t.Run("completed 不可放弃，返回业务前置错误", func(t *testing.T) {
		resetTables(t)
		seedWords(t, 1)
		svc := newTestReview(t)
		started := mustStart(t, svc, 1)
		require.NoError(t, svc.SubmitResult(ctx, service.SubmitResultRequest{
			SessionID: started.SessionID, ItemID: started.Items[0].ItemID, Result: domain.ReviewResultRemembered,
		}))

		err := svc.AbandonSession(ctx, service.AbandonSessionRequest{SessionID: started.SessionID})
		require.Error(t, err)
		var appErr *apperr.Error
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, apperr.FailedPrecondition, appErr.Kind)
		assert.Equal(t, apperr.CodeSessionNotActive, appErr.Code)
		assert.Equal(t, "review session is not active", appErr.Message)

		// completed 状态不被放弃破坏。
		session, err := repo.NewReviewRepo(testDB).GetSessionByID(ctx, started.SessionID)
		require.NoError(t, err)
		assert.Equal(t, domain.ReviewStatusCompleted, session.Status)
	})

	t.Run("不存在时返回未找到", func(t *testing.T) {
		resetTables(t)
		svc := newTestReview(t)

		err := svc.AbandonSession(ctx, service.AbandonSessionRequest{SessionID: uuid.Must(uuid.NewV7())})
		require.Error(t, err)
		var appErr *apperr.Error
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, apperr.NotFound, appErr.Kind)
		assert.Equal(t, apperr.CodeNotFound, appErr.Code)
		assert.Equal(t, "review session not found", appErr.Message)
	})
}

// ---- 用例专属辅助 ----

// newTestReview 构造共享数据库上的 ReviewService（固定随机源保证抽样可复现）。
func newTestReview(t *testing.T) *Review {
	t.Helper()
	return NewReview(requireTestDB(t), NewWeightedSampler(rand.NewPCG(7, 8)))
}

// seedWords 通过 WordService 导入 n 个唯一生词（候选集预置）。
func seedWords(t *testing.T, n int) []string {
	t.Helper()
	wordSvc := newTestWord(t)
	words := make([]string, 0, n)
	for i := 0; i < n; i++ {
		word := uniqWord(t, fmt.Sprintf("w%d", i))
		_, err := wordSvc.ImportWords(context.Background(), importReq(word, 1))
		require.NoError(t, err)
		words = append(words, word)
	}
	return words
}

// mustStart 开始一轮并要求成功。
func mustStart(t *testing.T, svc *Review, count int) *service.StartSessionResult {
	t.Helper()
	started, err := svc.StartSession(context.Background(), service.StartSessionRequest{Count: count})
	require.NoError(t, err)
	require.NotEmpty(t, started.Items)
	return started
}

// countRows 统计查询行数。
func countRows(t *testing.T, query string, args ...any) int64 {
	t.Helper()
	var n int64
	require.NoError(t, testDB.NewRaw(query, args...).Scan(context.Background(), &n))
	return n
}

// assertSessionCount 断言全库 session 总数。
func assertSessionCount(t *testing.T, want int64) {
	t.Helper()
	assert.Equal(t, want, countRows(t, "SELECT count(*) FROM review_sessions"))
}

// assertItemCount 断言全库 review_items 总数，reason 说明上下文。
func assertItemCount(t *testing.T, want int64, reason string) {
	t.Helper()
	assert.Equal(t, want, countRows(t, "SELECT count(*) FROM review_items"), reason)
}

// itemUserWordID 查询 item 对应的学习行 ID。
func itemUserWordID(t *testing.T, itemID uuid.UUID) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	require.NoError(t, testDB.NewRaw(
		"SELECT user_word_id FROM review_items WHERE id = ?", itemID,
	).Scan(context.Background(), &id))
	return id
}

// mustParseUUID 解析 UUID 字符串，失败立即终止。
func mustParseUUID(t *testing.T, raw string) uuid.UUID {
	t.Helper()
	id, err := uuid.Parse(raw)
	require.NoError(t, err)
	return id
}

// installTrigger 创建测试专用的 review_items 插入触发器注入真实数据库
// 失败，t.Cleanup 时移除；仅作用于测试库，不触碰 migrations。
func installTrigger(t *testing.T, name, functionBody string) {
	t.Helper()
	ctx := context.Background()
	_, err := testDB.ExecContext(ctx,
		"CREATE OR REPLACE FUNCTION "+name+"_fn() RETURNS trigger AS $fn$ "+functionBody+" $fn$ LANGUAGE plpgsql")
	require.NoError(t, err)
	_, err = testDB.ExecContext(ctx,
		"CREATE TRIGGER "+name+" AFTER INSERT ON review_items FOR EACH ROW EXECUTE FUNCTION "+name+"_fn()")
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = testDB.ExecContext(ctx, "DROP TRIGGER IF EXISTS "+name+" ON review_items")
		_, _ = testDB.ExecContext(ctx, "DROP FUNCTION IF EXISTS "+name+"_fn()")
	})
}
