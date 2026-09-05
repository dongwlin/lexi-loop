package schema

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// ReviewSession 是 review_sessions 表的 bun ORM 映射，字段语义以
// docs/review/data-model.md §1 为准；status 取值与「全库至多一个 active
// session」的部分唯一索引由 migrations/000002_review 落地（D010）。
// Status 是 domain.ReviewStatus 的持久化形态，转换在 repo 内。
type ReviewSession struct {
	bun.BaseModel `bun:"table:review_sessions,alias:rs"`

	ID              uuid.UUID  `bun:",pk,type:uuid"`
	RequestedCount  int        `bun:"requested_count,notnull"`
	TotalCount      int        `bun:"total_count,notnull"`
	RememberedCount int        `bun:"remembered_count,notnull,default:0"`
	ForgottenCount  int        `bun:"forgotten_count,notnull,default:0"`
	Status          string     `bun:"status,notnull,default:'active'"`
	StartedAt       time.Time  `bun:"started_at,nullzero,notnull,default:current_timestamp"`
	CompletedAt     *time.Time `bun:"completed_at"`
	CreatedAt       time.Time  `bun:"created_at,nullzero,notnull,default:current_timestamp"`
}

// ReviewItem 是 review_items 表的 bun ORM 映射，字段语义以
// docs/review/data-model.md §2 为准；result 取值、(session_id, position)
// 与 (session_id, user_word_id) 唯一约束由 migrations/000002_review 落地。
// Result 是 domain.ReviewResult 的持久化形态，转换在 repo 内。
type ReviewItem struct {
	bun.BaseModel `bun:"table:review_items,alias:ri"`

	ID         uuid.UUID  `bun:",pk,type:uuid"`
	SessionID  uuid.UUID  `bun:"session_id,notnull"`
	UserWordID uuid.UUID  `bun:"user_word_id,notnull"`
	Position   int        `bun:"position,notnull"`
	Result     string     `bun:"result,notnull,default:'pending'"`
	CreatedAt  time.Time  `bun:"created_at,nullzero,notnull,default:current_timestamp"`
	ReviewedAt *time.Time `bun:"reviewed_at"`
}

// ReviewItemWithHeadword 是 review_item 与所属词条 headword 组合扫描的行
// 模型：GetSession 的逐词结果需要单词本身（JOIN 出 headword）。
type ReviewItemWithHeadword struct {
	ReviewItem `bun:",extend"`

	Headword string `bun:"headword"`
}
