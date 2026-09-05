package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDictionaryEntry(t *testing.T) {
	t.Parallel()

	cst := time.FixedZone("CST", 8*3600)
	now := time.Date(2026, 9, 6, 12, 30, 0, 0, cst)

	t.Run("创建词条生成 UUID v7 与 UTC 时间戳", func(t *testing.T) {
		t.Parallel()

		entry, err := NewDictionaryEntry("ambiguous", "ambiguous", now)
		require.NoError(t, err)

		assert.Equal(t, uuid.Version(7), entry.ID.Version())
		assert.Equal(t, "ambiguous", entry.Headword)
		assert.Equal(t, "ambiguous", entry.Lemma)
		assert.Equal(t, now.UTC(), entry.CreatedAt)
		assert.Equal(t, now.UTC(), entry.UpdatedAt)
		assert.True(t, entry.CreatedAt.Location() == time.UTC)
	})

	t.Run("lemma 未知时回退为 headword", func(t *testing.T) {
		t.Parallel()

		entry, err := NewDictionaryEntry("Derived", "", now)
		require.NoError(t, err)
		assert.Equal(t, "Derived", entry.Headword)
		assert.Equal(t, "Derived", entry.Lemma)
	})

	t.Run("headword 为空报错", func(t *testing.T) {
		t.Parallel()

		entry, err := NewDictionaryEntry("", "whatever", now)
		require.ErrorIs(t, err, ErrHeadwordRequired)
		assert.Nil(t, entry)
	})
}

func TestEffectiveReviewMeaning(t *testing.T) {
	t.Parallel()

	custom := []Meaning{{Pos: "noun", Translations: []string{"自定义释义"}}}
	review := []Meaning{{Pos: "verb", Translations: []string{"获得", "推导"}}}
	raw := []Meaning{{Pos: "verb", Translations: []string{"获得", "推导", "源于"}}}

	tests := []struct {
		name       string
		custom     []Meaning
		review     []Meaning
		raw        []Meaning
		want       []Meaning
		wantSource MeaningSource
	}{
		{
			name:       "用户自定义优先",
			custom:     custom,
			review:     review,
			raw:        raw,
			want:       custom,
			wantSource: MeaningSourceCustom,
		},
		{
			name:       "未自定义时回退词典层默认复习释义",
			custom:     nil,
			review:     review,
			raw:        raw,
			want:       review,
			wantSource: MeaningSourceReview,
		},
		{
			name:       "自定义为空切片视为未设置",
			custom:     []Meaning{},
			review:     review,
			raw:        raw,
			want:       review,
			wantSource: MeaningSourceReview,
		},
		{
			name:       "两层皆空时回退原始释义",
			custom:     nil,
			review:     nil,
			raw:        raw,
			want:       raw,
			wantSource: MeaningSourceRaw,
		},
		{
			name:       "三层皆空返回空与零值来源",
			custom:     nil,
			review:     nil,
			raw:        nil,
			want:       nil,
			wantSource: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, source := EffectiveReviewMeaning(tt.custom, tt.review, tt.raw)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantSource, source)
		})
	}
}
