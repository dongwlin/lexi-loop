package ecdict

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"time"

	"github.com/uptrace/bun"

	"github.com/dongwlin/lexi-loop/apps/server/internal/domain"
	"github.com/dongwlin/lexi-loop/apps/server/internal/repo"
)

// DefaultBatchSize 是默认的单个事务提交行数：每个批次一个独立事务，
// 平衡单批失败的重放成本与提交开销（structure.md §5.3：按固定批次提交，
// 失败只回滚当前批次）。
const DefaultBatchSize = 1000

// maxCommitAttempts 是批次事务因死锁 / 序列化失败而整体重放的上限
// （含首次），与 service 层可重放事务同一语义。
const maxCommitAttempts = 3

// Importer 编排 ECDICT CSV 的离线导入（docs/backend/structure.md §8）：
// 流式解析 → 按固定批次分事务经 repo 写入 dictionary_entries。
// 写入按 headword 唯一键幂等（UpsertBatch），任一批次失败后重跑同一
// 文件即可续传；进度经 onProgress 逐批汇报。
type Importer struct {
	db        *bun.DB
	batchSize int
}

// NewImporter 构造导入器；batchSize <= 0 时取 DefaultBatchSize。
func NewImporter(db *bun.DB, batchSize int) *Importer {
	if batchSize <= 0 {
		batchSize = DefaultBatchSize
	}
	return &Importer{db: db, batchSize: batchSize}
}

// Progress 是已提交批次的累计进度，由 onProgress 逐批汇报。
type Progress struct {
	BatchesCommitted int // 已提交的批次事务数
	RowsProcessed    int // 已读取的数据行数（含跳过行）
	RowsSkipped      int // 已跳过的无效行数
	EntriesWritten   int // 已写入 / 刷新的词条数
}

// Result 是一次导入的最终统计。
type Result struct {
	RowsProcessed    int // 读取的数据行总数（含跳过行，不含表头）
	RowsSkipped      int // 跳过的无效行数（word 为空）
	EntriesWritten   int // 写入 / 刷新的词条数（新词条与更新词条都计入）
	BatchesCommitted int // 提交的批次事务数
}

// Import 流式导入 r 中的 ECDICT CSV。每个满批（或收尾的剩余行）作为
// 一个独立事务提交：批次内任一行失败则整批回滚并中止导入，已提交的
// 批次保持有效——重跑同一文件即可续传（UpsertBatch 幂等）。
// sourceVersion 是数据版本标记，写入每条词条的 source_version（词典
// 自动导入按它与 manifest 期望行数做完整性守卫），空串表示未版本化。
// onProgress 可为 nil；ctx 取消后在当前批次边界停止，不中断已提交数据。
func (imp *Importer) Import(ctx context.Context, r io.Reader, sourceVersion string, onProgress func(Progress)) (Result, error) {
	reader, err := NewRowReader(r, sourceVersion)
	if err != nil {
		return Result{}, fmt.Errorf("ecdict: prepare csv reader: %w", err)
	}

	var result Result
	batch := make([]*domain.DictionaryEntry, 0, imp.batchSize)
	// batchIndex 记录当前批次内 headword 的下标：同批重复词就地覆盖
	// （保留后出现的行），避免「ON CONFLICT DO UPDATE 同一行作用两次」
	// 的语句级错误；跨批重复由 UpsertBatch 幂等吸收。
	batchIndex := make(map[string]int, imp.batchSize)

	commit := func() error {
		if len(batch) == 0 {
			return nil
		}
		written, err := imp.commitBatch(ctx, batch)
		if err != nil {
			return err
		}
		result.BatchesCommitted++
		result.EntriesWritten += written
		batch = batch[:0]
		clear(batchIndex)
		if onProgress != nil {
			onProgress(Progress{
				BatchesCommitted: result.BatchesCommitted,
				RowsProcessed:    reader.RowsRead(),
				RowsSkipped:      reader.RowsSkipped(),
				EntriesWritten:   result.EntriesWritten,
			})
		}
		return nil
	}

	for {
		if err := ctx.Err(); err != nil {
			result.RowsProcessed = reader.RowsRead()
			result.RowsSkipped = reader.RowsSkipped()
			return result, fmt.Errorf("ecdict: import interrupted: %w", err)
		}
		entry, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			result.RowsProcessed = reader.RowsRead()
			result.RowsSkipped = reader.RowsSkipped()
			return result, err
		}
		if idx, dup := batchIndex[entry.Headword]; dup {
			batch[idx] = entry
		} else {
			batchIndex[entry.Headword] = len(batch)
			batch = append(batch, entry)
		}
		if len(batch) >= imp.batchSize {
			if err := commit(); err != nil {
				result.RowsProcessed = reader.RowsRead()
				result.RowsSkipped = reader.RowsSkipped()
				return result, err
			}
		}
	}
	if err := commit(); err != nil {
		result.RowsProcessed = reader.RowsRead()
		result.RowsSkipped = reader.RowsSkipped()
		return result, err
	}

	result.RowsProcessed = reader.RowsRead()
	result.RowsSkipped = reader.RowsSkipped()
	return result, nil
}

// commitBatch 在一个事务内把一个批次经 repo 写入，返回写入词条数：
// ON CONFLICT DO UPDATE 语句的每一行都会被插入或更新，且同批重复词已在
// Import 内去重，因此写入数即 len(entries)。重跑导入按 headword 冲突
// 刷新词典字段，repo 层的 COALESCE 保证 review_meanings 不被空值覆盖。
func (imp *Importer) commitBatch(ctx context.Context, entries []*domain.DictionaryEntry) (int, error) {
	err := imp.runReplayableTx(ctx, func(ctx context.Context, tx bun.Tx) error {
		return repo.NewDictionaryRepo(tx).UpsertBatch(ctx, entries)
	})
	if err != nil {
		return 0, fmt.Errorf("ecdict: commit batch of %d entries: %w", len(entries), err)
	}
	return len(entries), nil
}

// runReplayableTx 对批次事务做有上限重放。只重放死锁 / 序列化失败
// （repo.ErrConflict）：批次写入幂等，整体重放安全，不能只重放最后一条
// 语句。UpsertBatch 经 ON CONFLICT 吸收唯一约束冲突，不会返回
// repo.ErrAlreadyExists，其余错误不可重放。独立于 service 包的同名实现：
// 依赖方向不允许 importer 反向引用 service（structure.md §1）。
func (imp *Importer) runReplayableTx(ctx context.Context, fn func(context.Context, bun.Tx) error) error {
	var lastErr error
	for attempt := 1; attempt <= maxCommitAttempts; attempt++ {
		err := imp.db.RunInTx(ctx, nil, fn)
		if err == nil {
			return nil
		}
		if !errors.Is(err, repo.ErrConflict) {
			return err
		}
		lastErr = err
		if attempt < maxCommitAttempts {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(commitBackoff(attempt)):
			}
		}
	}
	return lastErr
}

// commitBackoff 返回第 attempt 次重放前的等待：25ms 起步的指数退避叠加
// 均匀抖动（与 service 层同一策略）。
func commitBackoff(attempt int) time.Duration {
	base := time.Duration(1<<uint(attempt-1)) * 25 * time.Millisecond
	return base/2 + time.Duration(rand.Int64N(int64(base)))
}
