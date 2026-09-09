package v1

import (
	"context"
	"errors"
	"math/rand/v2"
	"time"

	"github.com/uptrace/bun"

	"github.com/dongwlin/lexi-loop/apps/server/internal/apperr"
	"github.com/dongwlin/lexi-loop/apps/server/internal/repo"
)

// replayableTxMaxAttempts 是可重放事务的最大尝试次数（含首次）。
const replayableTxMaxAttempts = 3

// runReplayableTx 执行一个可整体重放的事务（structure.md §5）：唯一约束
// 冲突（repo.ErrAlreadyExists，如并发创建词条、active session 部分唯一
// 索引兜底）与死锁 / 序列化失败（repo.ErrConflict）时，以带抖动的指数
// 退避重试**整个事务**，不能只重试最后一条语句；超出上限后映射为
// Conflict / BASE.BIZ.CONCURRENT_UPDATE。其余错误（含业务 apperr 与
// 不可重放的领域错误）不重试、原样返回。
func runReplayableTx(ctx context.Context, db *bun.DB, fn func(context.Context, bun.Tx) error) error {
	var lastErr error
	for attempt := 1; attempt <= replayableTxMaxAttempts; attempt++ {
		err := db.RunInTx(ctx, nil, fn)
		if err == nil {
			return nil
		}
		if !errors.Is(err, repo.ErrAlreadyExists) && !errors.Is(err, repo.ErrConflict) {
			return err
		}
		lastErr = err
		if attempt < replayableTxMaxAttempts {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(retryBackoff(attempt)):
			}
		}
	}
	return apperr.New(apperr.Conflict, apperr.CodeConcurrentUpdate, "concurrent update, please retry", lastErr)
}

// retryBackoff 返回第 attempt 次重试前的等待：指数退避 25ms 起步，
// 叠加均匀抖动，避免重试风暴。
func retryBackoff(attempt int) time.Duration {
	base := time.Duration(1<<uint(attempt-1)) * 25 * time.Millisecond
	return base/2 + time.Duration(rand.Int64N(int64(base)))
}

// internalError 把未预期错误统一包装为内部应用错误；已为 apperr.Error
// 的原样返回，nil 原样返回（docs/specs/backend/Go 单体应用架构规范.md §7.2）。
func internalError(err error) error {
	if err == nil {
		return nil
	}
	var appErr *apperr.Error
	if errors.As(err, &appErr) {
		return appErr
	}
	return apperr.Internal(err)
}
