package schema

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// UserWord 是 user_words 表的 bun ORM 映射，字段语义以
// docs/dictionary/data-model.md §3 为准（软删除 D004；weight / mastery
// 不落库 D011）；外键与唯一约束由 migrations/000002_review 落地。
type UserWord struct {
	bun.BaseModel `bun:"table:user_words,alias:uw"`

	ID                  uuid.UUID        `bun:",pk,type:uuid"`
	DictionaryEntryID   uuid.UUID        `bun:"dictionary_entry_id,notnull"`
	EncounterCount      int              `bun:"encounter_count,notnull,default:0"`
	ReviewCount         int              `bun:"review_count,notnull,default:0"`
	RememberCount       int              `bun:"remember_count,notnull,default:0"`
	ForgetCount         int              `bun:"forget_count,notnull,default:0"`
	CurrentStreak       int              `bun:"current_streak,notnull,default:0"`
	LastReviewedAt      *time.Time       `bun:"last_reviewed_at"`
	CustomReviewMeaning JSONB[[]Meaning] `bun:"custom_review_meaning,type:jsonb"`
	DeletedAt           *time.Time       `bun:"deleted_at"`
	CreatedAt           time.Time        `bun:"created_at,nullzero,notnull,default:current_timestamp"`
	UpdatedAt           time.Time        `bun:"updated_at,nullzero,notnull,default:current_timestamp"`
}
