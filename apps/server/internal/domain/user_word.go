package domain

import (
	"math"
	"time"

	"github.com/google/uuid"
)

// ReviewWeight 公式中的固定参数（唯一权威为 docs/review/algorithm.md §3–§4）。
const (
	reviewWeightBase              = 1.0 // base
	reviewWeightDifficultyScale   = 4.0 // difficulty_factor = difficulty × 4
	reviewWeightUnknownDifficulty = 0.5 // 新词无正确率可用，difficulty 直接取 0.5
	reviewWeightNewWordBonus      = 2.0 // 新词保底，避免长期抽不到
	reviewWeightStreakRecalledMin = 4   // 连续记得达到该次数后略降权
	reviewWeightStreakRecallDrop  = 1.0 // 连续记得较多时的降权值
	reviewWeightOverduePerDay     = 1.0 / 7.0
	reviewWeightOverdueCap        = 3.0
	reviewWeightFloor             = 0.5 // 最终权重下限，保证所有词保留最低出现概率
)

// UserWord 对应 user_words：用户对某个词条的个人学习状态，与词典数据
// （DictionaryEntry）完全分离（docs/dictionary/data-model.md §3）。
// weight / mastery 等派生指标不落库、动态计算（冻结决策 D011）。
type UserWord struct {
	ID                  uuid.UUID
	DictionaryEntryID   uuid.UUID
	EncounterCount      int
	ReviewCount         int
	RememberCount       int
	ForgetCount         int
	CurrentStreak       int // 正数 = 连续记得，负数 = 连续忘记
	LastReviewedAt      *time.Time
	CustomReviewMeaning []Meaning // NULL 表示未自定义（空切片同理）
	DeletedAt           *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// NewUserWord 创建生词学习行：复习累计字段全部为零，encounterCount 为
// 首次导入计入的遇词次数（首次导入为 1、重复导入累计，
// docs/dictionary/data-model.md §3）。生成 UUID v7 主键与 UTC 时间戳。
func NewUserWord(dictionaryEntryID uuid.UUID, encounterCount int, now time.Time) (*UserWord, error) {
	if encounterCount < 1 {
		return nil, ErrInvalidEncounterCount
	}
	id, err := newUUIDv7()
	if err != nil {
		return nil, err
	}
	created := now.UTC()
	return &UserWord{
		ID:                id,
		DictionaryEntryID: dictionaryEntryID,
		EncounterCount:    encounterCount,
		CreatedAt:         created,
		UpdatedAt:         created,
	}, nil
}

// ApplyReview 依据一次有效作答更新词级累计状态
// （docs/review/data-model.md §5；streak 规则见 docs/api/reviews.md §4）：
// review_count + 1，按结果累计 remember / forget，按 streak 规则更新
// current_streak，并记录 last_reviewed_at。result 为 pending 或未知值时
// 返回 ErrInvalidReviewResult 且不做任何修改。
func (w *UserWord) ApplyReview(result ReviewResult, now time.Time) error {
	if !result.isAnswer() {
		return ErrInvalidReviewResult
	}
	w.ReviewCount++
	if result == ReviewResultRemembered {
		w.RememberCount++
	} else {
		w.ForgetCount++
	}
	switch {
	case result == ReviewResultRemembered && w.CurrentStreak < 0:
		w.CurrentStreak = 1
	case result == ReviewResultRemembered:
		w.CurrentStreak++
	case w.CurrentStreak > 0:
		w.CurrentStreak = -1
	default:
		w.CurrentStreak--
	}
	reviewed := now.UTC()
	w.LastReviewedAt = &reviewed
	w.UpdatedAt = reviewed
	return nil
}

// ReviewWeight 计算复习优先级权重（docs/review/algorithm.md §3–§5，
// 公式的唯一权威）：基础权重 + 遇到频率 + 当前困难度 + 连续表现修正
// + 时间因素 + 新词保底。now 由调用方显式传入；weight 不落库。
func (w *UserWord) ReviewWeight(now time.Time) float64 {
	weight := reviewWeightBase
	weight += math.Log2(float64(w.EncounterCount) + 1)
	if w.ReviewCount == 0 {
		weight += reviewWeightUnknownDifficulty * reviewWeightDifficultyScale
		weight += reviewWeightNewWordBonus
	} else {
		rememberRate := float64(w.RememberCount) / float64(w.ReviewCount)
		weight += (1 - rememberRate) * reviewWeightDifficultyScale
	}
	switch {
	case w.CurrentStreak < 0:
		weight += float64(-w.CurrentStreak)
	case w.CurrentStreak >= reviewWeightStreakRecalledMin:
		weight -= reviewWeightStreakRecallDrop
	}
	if w.LastReviewedAt != nil {
		days := max(now.Sub(*w.LastReviewedAt).Hours()/24, 0)
		weight += min(days*reviewWeightOverduePerDay, reviewWeightOverdueCap)
	}
	return max(weight, reviewWeightFloor)
}

// MasteryScore 计算掌握程度（docs/review/algorithm.md §6）：
// (remember_count + 1) / (review_count + 2) × 100，相当于给每个词
// 初始 50% 的未知状态；与 ReviewWeight 相互独立、不落库。
func (w *UserWord) MasteryScore() float64 {
	return (float64(w.RememberCount) + 1) / (float64(w.ReviewCount) + 2) * 100
}
