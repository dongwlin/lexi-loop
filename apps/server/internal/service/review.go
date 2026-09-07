package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/dongwlin/lexi-loop/apps/server/internal/apperr"
	"github.com/dongwlin/lexi-loop/apps/server/internal/domain"
	"github.com/dongwlin/lexi-loop/apps/server/internal/repo"
)

// errNoReviewableWords：可复习词数为 0（api/reviews.md §2），由
// StartSession 在候选查询后返回、事务外映射为 apperr，事务不创建 session。
var errNoReviewableWords = errors.New("service: no reviewable words available")

// Review 提供复习用例（docs/api/reviews.md）：开始一轮、提交结果、获取
// 结果。事务与并发边界见 docs/backend/structure.md §5。
type Review struct {
	db      *bun.DB
	sampler *WeightedSampler
}

// NewReview 构造 ReviewService；sampler 用于开始一轮时的加权随机抽样。
func NewReview(db *bun.DB, sampler *WeightedSampler) *Review {
	return &Review{db: db, sampler: sampler}
}

// ---- 用例请求 / 结果类型（无 json 标签，JSON 契约由版本化 DTO 定义）----

// StartSessionRequest 是开始一轮复习用例的请求。
type StartSessionRequest struct {
	Count int
}

// StartSessionItem 是本轮抽中单词的展示读模型（api/reviews.md §2 items）。
type StartSessionItem struct {
	ItemID           uuid.UUID
	Headword         string
	Phonetic         string
	EffectiveMeaning []domain.Meaning
	MeaningSource    domain.MeaningSource
}

// StartSessionResult 是开始一轮复习用例的结果。
type StartSessionResult struct {
	SessionID      uuid.UUID
	RequestedCount int
	TotalCount     int
	Items          []StartSessionItem
}

// SubmitResultRequest 是提交单词复习结果用例的请求。
type SubmitResultRequest struct {
	SessionID uuid.UUID
	ItemID    uuid.UUID
	// Result 必须是 remembered / forgotten 的有效作答。
	Result domain.ReviewResult
}

// GetSessionRequest 是获取一轮复习结果用例的请求。
type GetSessionRequest struct {
	SessionID uuid.UUID
}

// AbandonSessionRequest 是放弃一轮复习用例的请求。
type AbandonSessionRequest struct {
	SessionID uuid.UUID
}

// GetSessionItem 是逐词结果读模型（api/reviews.md §5 items）。
type GetSessionItem struct {
	Headword string
	Result   domain.ReviewResult
}

// GetSessionResult 是获取一轮复习结果用例的结果。
type GetSessionResult struct {
	SessionID   uuid.UUID
	Status      domain.ReviewStatus
	Total       int
	Remembered  int
	Forgotten   int
	CompletedAt *time.Time
	Items       []GetSessionItem
}

// StartSession 开始一轮复习（structure.md §5.1）：整个流程在单个事务内
// 完成——获取 active-session 作用域锁 → 查找并锁定旧 active session 并
// Abandon → 读取可复习候选词并按同一个 now 计算权重 → 加权随机不放回
// 抽样 → 创建 active ReviewSession → 批量创建全部 pending ReviewItem。
// 并发经 Repo 封装的 advisory transaction lock 串行化，部分唯一索引兜底
// （D010）；唯一约束冲突 / 死锁对整个可重放事务做有上限重试。
//
// D009：total = min(count, available)，请求数量超过候选数时截断不报错；
// 候选数为 0 时不创建 session，按契约返回 422。
func (s *Review) StartSession(ctx context.Context, req StartSessionRequest) (*StartSessionResult, error) {
	if req.Count < 1 {
		return nil, apperr.New(apperr.InvalidArgument, apperr.CodeValidationFailed, "count must be positive", nil)
	}

	var result *StartSessionResult
	err := runReplayableTx(ctx, s.db, func(ctx context.Context, tx bun.Tx) error {
		r, err := s.startSessionTx(ctx, tx, req.Count, time.Now())
		if err != nil {
			return err
		}
		result = r
		return nil
	})
	if errors.Is(err, errNoReviewableWords) {
		return nil, apperr.New(apperr.FailedPrecondition, apperr.CodeNoReviewableWords, "no reviewable words available", err)
	}
	if err != nil {
		return nil, internalError(err)
	}
	return result, nil
}

// startSessionTx 是 StartSession 的单事务实现（structure.md §5.1 顺序）。
// 同一用例共享一个 now：旧 session 的放弃时间、权重计算、session 与
// items 的时间戳全部一致。
func (s *Review) startSessionTx(ctx context.Context, tx bun.Tx, count int, now time.Time) (*StartSessionResult, error) {
	reviewRepo := repo.NewReviewRepo(tx)
	userWordRepo := repo.NewUserWordRepo(tx)
	dictionaryRepo := repo.NewDictionaryRepo(tx)

	// 1. 获取 active-session 作用域锁，串行化并发开始（D010）。
	if err := reviewRepo.AcquireActiveSessionLock(ctx); err != nil {
		return nil, err
	}

	// 2. 查找并锁定旧 active session；用户明确开始新一轮即视为放弃旧一轮
	//（review/data-model.md §4）。不存在则直接进入抽样。
	old, err := reviewRepo.LockActiveSession(ctx)
	if err != nil && !errors.Is(err, repo.ErrNotFound) {
		return nil, err
	}
	if old != nil {
		if err := old.Abandon(now); err != nil {
			return nil, err
		}
		if err := reviewRepo.UpdateSession(ctx, old); err != nil {
			return nil, err
		}
	}

	// 3. 读取可复习候选词（deleted_at IS NULL，api/reviews.md §2）。
	candidates, err := userWordRepo.ListActive(ctx)
	if err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		return nil, errNoReviewableWords
	}

	// 4. 按同一个 now 计算权重并加权随机不放回抽样；
	// total = min(count, available)（D009）。
	total := min(count, len(candidates))
	weights := make([]float64, len(candidates))
	for i, w := range candidates {
		weights[i] = w.ReviewWeight(now)
	}
	selected := s.sampler.Sample(weights, total)

	// 5. 创建 active session 与全部 pending items；部分唯一索引冲突
	//（ErrAlreadyExists）由 runReplayableTx 对整个事务重试。
	session, err := domain.NewReviewSession(count, total, now)
	if err != nil {
		return nil, err
	}
	if err := reviewRepo.InsertSession(ctx, session); err != nil {
		return nil, err
	}
	items := make([]*domain.ReviewItem, len(selected))
	picked := make([]*domain.UserWord, len(selected))
	for i, idx := range selected {
		item, err := domain.NewReviewItem(session.ID, candidates[idx].ID, i, now)
		if err != nil {
			return nil, err
		}
		items[i] = item
		picked[i] = candidates[idx]
	}
	if err := reviewRepo.InsertItems(ctx, items); err != nil {
		return nil, err
	}

	// 6. 组装本轮单词预览：词典字段 + 三层取值释义（D007）。
	entryIDs := make([]uuid.UUID, 0, len(picked))
	seen := make(map[uuid.UUID]struct{}, len(picked))
	for _, w := range picked {
		if _, ok := seen[w.DictionaryEntryID]; !ok {
			seen[w.DictionaryEntryID] = struct{}{}
			entryIDs = append(entryIDs, w.DictionaryEntryID)
		}
	}
	entries, err := dictionaryRepo.FindByIDs(ctx, entryIDs)
	if err != nil {
		return nil, err
	}
	byID := make(map[uuid.UUID]*domain.DictionaryEntry, len(entries))
	for _, entry := range entries {
		byID[entry.ID] = entry
	}

	result := &StartSessionResult{
		SessionID:      session.ID,
		RequestedCount: count,
		TotalCount:     total,
		Items:          make([]StartSessionItem, 0, len(items)),
	}
	for i, item := range items {
		entry := byID[picked[i].DictionaryEntryID]
		if entry == nil {
			// user_words.dictionary_entry_id 外键保证词条存在；缺失属于数据损坏。
			return nil, fmt.Errorf("service: dictionary entry %s missing for user word %s", picked[i].DictionaryEntryID, picked[i].ID)
		}
		meaning, source := domain.EffectiveReviewMeaning(picked[i].CustomReviewMeaning, entry.ReviewMeanings, entry.RawMeanings)
		result.Items = append(result.Items, StartSessionItem{
			ItemID:           item.ID,
			Headword:         entry.Headword,
			Phonetic:         displayPhonetic(entry),
			EffectiveMeaning: meaning,
			MeaningSource:    source,
		})
	}
	return result, nil
}

// SubmitResult 提交一个单词的复习结果（structure.md §5.2）：一次提交的
// item 状态、词级累计数据与 session 汇总必须在**一个事务**内完成，固定
// 锁顺序为 review_session → review_item → user_word，所有提交路径都遵守。
//
// 幂等与归属校验按 API 契约「条件 UPDATE 不命中即返回当前状态」执行
// （docs/api/reviews.md §3）：session 不存在、item 不存在、归属不匹配或
// 已作答都不修改统计、不报错。死锁 / 序列化失败不自动重试，映射为
// Conflict / BASE.BIZ.CONCURRENT_UPDATE。
func (s *Review) SubmitResult(ctx context.Context, req SubmitResultRequest) error {
	if req.Result != domain.ReviewResultRemembered && req.Result != domain.ReviewResultForgotten {
		return apperr.New(apperr.InvalidArgument, apperr.CodeValidationFailed, "result must be remembered or forgotten", nil)
	}

	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		return s.submitResultTx(ctx, tx, req, time.Now())
	})
	switch {
	case err == nil:
		return nil
	case errors.Is(err, repo.ErrConflict):
		return apperr.New(apperr.Conflict, apperr.CodeConcurrentUpdate, "resource was modified concurrently, please retry", err)
	default:
		return internalError(err)
	}
}

// submitResultTx 是 SubmitResult 的单事务实现（structure.md §5.2 顺序）。
func (s *Review) submitResultTx(ctx context.Context, tx bun.Tx, req SubmitResultRequest, now time.Time) error {
	reviewRepo := repo.NewReviewRepo(tx)
	userWordRepo := repo.NewUserWordRepo(tx)

	// 1. 先锁定 URL 指定的 ReviewSession，串行化同一 session 的提交。
	// session 不存在：契约上等同条件 UPDATE 不命中，幂等短路。
	session, err := reviewRepo.LockSessionByID(ctx, req.SessionID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil
		}
		return err
	}

	// 1.5. session 已非 active（abandoned / completed）：幂等短路、不再
	// 计数，防止放弃或完成一轮之后迟到的提交计入已结束的轮次
	//（api/reviews.md §3；completed 本就全部非 pending，此处一并防御）。
	if session.Status != domain.ReviewStatusActive {
		return nil
	}

	// 2. 按 session_id + item_id 锁定 ReviewItem；不存在或归属不匹配：
	// 幂等短路，不修改任何统计（structure.md §5.2）。
	item, err := reviewRepo.LockItem(ctx, req.SessionID, req.ItemID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil
		}
		return err
	}

	// 3. 已非 pending：幂等返回当前状态，不重复计数。
	if item.Result != domain.ReviewResultPending {
		return nil
	}

	// 4. Domain 方法执行业务状态迁移，条件 UPDATE 是持久化层并发保护，
	// 两者必须同时保留（structure.md §5.2）。
	if err := item.Submit(req.Result, now); err != nil {
		return err
	}
	userWordID, err := reviewRepo.UpdateItemIfPending(ctx, req.SessionID, req.ItemID, req.Result, now)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			// 理论不可达（item 行已锁定且仍为 pending）；按契约幂等短路防御。
			return nil
		}
		return err
	}

	// 5. 锁定对应 UserWord（锁顺序第三步），经 Domain 方法累计并写回。
	word, err := userWordRepo.LockByID(ctx, userWordID)
	if err != nil {
		return err
	}
	if err := word.ApplyReview(req.Result, now); err != nil {
		return err
	}
	if err := userWordRepo.UpdateReviewCounters(ctx, word); err != nil {
		return err
	}

	// 6. 若已无 pending item，调用 Complete 并回写汇总（同一事务内，
	// session 行锁保证并发提交后仍恰好完成一次）。
	remembered, forgotten, pending, err := reviewRepo.CountResults(ctx, req.SessionID)
	if err != nil {
		return err
	}
	if pending == 0 {
		if err := session.Complete(remembered, forgotten, now); err != nil {
			return err
		}
		if err := reviewRepo.UpdateSession(ctx, session); err != nil {
			return err
		}
	}
	return nil
}

// AbandonSession 放弃一轮进行中的复习（api/reviews.md §6）：active →
// abandoned。持有 session 行锁，与提交共享同一锁顺序（structure.md §5.2），
// 并发时二者串行：先提交者按契约计数，先放弃者使后续提交幂等短路。
// 幂等：对已是 abandoned 的 session 重复放弃同样返回成功；completed 不可
// 放弃，映射为业务前置 422；session 不存在返回 404。
func (s *Review) AbandonSession(ctx context.Context, req AbandonSessionRequest) error {
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		reviewRepo := repo.NewReviewRepo(tx)
		session, err := reviewRepo.LockSessionByID(ctx, req.SessionID)
		if err != nil {
			return err
		}
		if session.Status == domain.ReviewStatusAbandoned {
			return nil
		}
		if err := session.Abandon(time.Now()); err != nil {
			return err
		}
		return reviewRepo.UpdateSession(ctx, session)
	})
	switch {
	case err == nil:
		return nil
	case errors.Is(err, repo.ErrNotFound):
		return sessionNotFound(err)
	case errors.Is(err, domain.ErrSessionNotActive):
		// 复用标准业务前置 code BASE.BIZ.USER_DISABLED（422）：「不满足
		// 业务前置条件」的既有语义，message 区分具体场景。
		return apperr.New(apperr.FailedPrecondition, apperr.CodeNoReviewableWords, "review session is not active", err)
	case errors.Is(err, repo.ErrConflict):
		return apperr.New(apperr.Conflict, apperr.CodeConcurrentUpdate, "resource was modified concurrently, please retry", err)
	default:
		return internalError(err)
	}
}

// GetSession 获取一轮复习的汇总与逐词结果（api/reviews.md §5）。
// remembered / forgotten 从 review_items 实时统计，保证与逐词结果一致：
// completed 后与 session 存储的汇总相等，active / abandoned 时反映已作答
// 进度（review/data-model.md §4：进度由 review_items 保存）。
func (s *Review) GetSession(ctx context.Context, req GetSessionRequest) (*GetSessionResult, error) {
	reviewRepo := repo.NewReviewRepo(s.db)

	session, err := reviewRepo.GetSessionByID(ctx, req.SessionID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, sessionNotFound(err)
		}
		return nil, apperr.Internal(err)
	}
	items, err := reviewRepo.ListSessionItems(ctx, req.SessionID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	remembered, forgotten, _, err := reviewRepo.CountResults(ctx, req.SessionID)
	if err != nil {
		return nil, apperr.Internal(err)
	}

	result := &GetSessionResult{
		SessionID:   session.ID,
		Status:      session.Status,
		Total:       session.TotalCount,
		Remembered:  remembered,
		Forgotten:   forgotten,
		CompletedAt: session.CompletedAt,
		Items:       make([]GetSessionItem, 0, len(items)),
	}
	for _, row := range items {
		result.Items = append(result.Items, GetSessionItem{Headword: row.Headword, Result: row.Item.Result})
	}
	return result, nil
}

// sessionNotFound 统一 session 不存在的应用错误（api/reviews.md §5 的契约 message）。
func sessionNotFound(cause error) error {
	return apperr.New(apperr.NotFound, apperr.CodeNotFound, "review session not found", cause)
}
