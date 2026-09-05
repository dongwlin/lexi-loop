package schema

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// Meaning 是释义列的 JSON 结构，与 dictionary_entries.raw_meanings /
// review_meanings 及 user_words.custom_review_meaning 的 JSONB 内容同构
// （docs/dictionary/data-model.md §4）。与 domain.Meaning 的转换在 repo 内。
type Meaning struct {
	Pos          string   `json:"pos"`
	Translations []string `json:"translations"`
}

// DictionaryEntry 是 dictionary_entries 表的 bun ORM 映射，字段语义以
// docs/dictionary/data-model.md §2 为准；headword 唯一、lemma 建索引由
// migrations/000001_dictionary 落地。
type DictionaryEntry struct {
	bun.BaseModel `bun:"table:dictionary_entries,alias:de"`

	ID             uuid.UUID                `bun:",pk,type:uuid"`
	Headword       string                   `bun:"headword,notnull"`
	Lemma          string                   `bun:"lemma,notnull"`
	PhoneticUK     string                   `bun:"phonetic_uk,notnull,default:''"`
	PhoneticUS     string                   `bun:"phonetic_us,notnull,default:''"`
	RawMeanings    JSONB[[]Meaning]         `bun:"raw_meanings,type:jsonb,nullzero,notnull,default:'[]'"`
	ReviewMeanings JSONB[[]Meaning]         `bun:"review_meanings,type:jsonb"`
	Exchange       JSONB[map[string]string] `bun:"exchange,type:jsonb,nullzero,notnull,default:'{}'"`
	Frequency      JSONB[map[string]int]    `bun:"frequency,type:jsonb,nullzero,notnull,default:'{}'"`
	Tags           JSONB[[]string]          `bun:"tags,type:jsonb,nullzero,notnull,default:'[]'"`
	Source         string                   `bun:"source,notnull,default:''"`
	SourceVersion  string                   `bun:"source_version,notnull,default:''"`
	CreatedAt      time.Time                `bun:"created_at,nullzero,notnull,default:current_timestamp"`
	UpdatedAt      time.Time                `bun:"updated_at,nullzero,notnull,default:current_timestamp"`
}
