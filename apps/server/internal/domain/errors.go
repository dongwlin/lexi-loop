package domain

import "errors"

// 领域不变量错误以 sentinel error 定义在 domain 包内，供 Service 通过
// errors.Is 识别后映射为 apperr.Error（docs/backend/structure.md §4.5）。
var (
	// ErrHeadwordRequired：词条必须有 headword（dictionary_entries 不变式）。
	ErrHeadwordRequired = errors.New("domain: headword is required")
	// ErrInvalidEncounterCount：user_words 初始遇词次数必须为正整数
	//（首次导入为 1，重复导入累计，见 docs/dictionary/data-model.md §3）。
	ErrInvalidEncounterCount = errors.New("domain: encounter count must be positive")
	// ErrInvalidSessionCounts：review_sessions 数量不合法；D009 截断
	//（total = min(count, available)）由 Service 完成，Domain 只校验
	// requested ≥ 1、total ≥ 1 且 total ≤ requested。
	ErrInvalidSessionCounts = errors.New("domain: invalid review session counts")
	// ErrSessionTotalsMismatch：Complete 的 remembered + forgotten
	// 与 total_count 不一致（一轮全部 item 都已作答时两者必须相等）。
	ErrSessionTotalsMismatch = errors.New("domain: remembered and forgotten counts do not match total count")
	// ErrSessionNotActive：session 的完成 / 放弃只允许从 active 发起
	//（状态迁移见 docs/review/data-model.md §4）。
	ErrSessionNotActive = errors.New("domain: review session is not active")
	// ErrInvalidPosition：review_items 的 position 必须为非负整数。
	ErrInvalidPosition = errors.New("domain: review item position must be non-negative")
	// ErrInvalidReviewResult：作答结果只能是 remembered / forgotten，
	// pending 表示未作答、不是有效作答。
	ErrInvalidReviewResult = errors.New("domain: invalid review result")
	// ErrItemAlreadyAnswered：item 已作答；pending → 已作答只能发生一次
	//（幂等语义见 docs/api/reviews.md §3）。
	ErrItemAlreadyAnswered = errors.New("domain: review item already answered")
)
