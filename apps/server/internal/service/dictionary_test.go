package service

// DictionaryService 集成测试：真实 PostgreSQL 上的本地 Lookup 用例
// （structure.md §4.3：归一 → 命中 → 词形归并 → 未命中创建最小词条）。

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dongwlin/lexi-loop/apps/server/internal/apperr"
	"github.com/dongwlin/lexi-loop/apps/server/internal/domain"
	"github.com/dongwlin/lexi-loop/apps/server/internal/repo"
)

func TestIntegration_DictionaryService_Lookup(t *testing.T) {
	ctx := context.Background()

	t.Run("命中本地词典时直接返回词条", func(t *testing.T) {
		resetTables(t)
		svc := NewDictionary(requireTestDB(t))
		word := uniqWord(t, "ambiguous")
		entry := testEntry(word, word, map[string]string{"0": word}, nil, nil)
		mustInsertEntry(t, entry)

		got, err := svc.Lookup(ctx, "  "+word+"  ")
		require.NoError(t, err)
		assert.Equal(t, entry.ID, got.ID)
		assert.Equal(t, word, got.Headword)
	})

	t.Run("借助词形变化表归并到原形词条", func(t *testing.T) {
		resetTables(t)
		svc := NewDictionary(requireTestDB(t))
		base := uniqWord(t, "derive")
		inflected := uniqWord(t, "derived")
		baseEntry := testEntry(base, base, map[string]string{"0": base}, nil, nil)
		inflectedEntry := testEntry(inflected, inflected, map[string]string{"0": base}, nil, nil)
		mustInsertEntry(t, baseEntry)
		mustInsertEntry(t, inflectedEntry)

		got, err := svc.Lookup(ctx, inflected)
		require.NoError(t, err)
		assert.Equal(t, baseEntry.ID, got.ID, "应归并到原形词条")
	})

	t.Run("原形词条缺失时保留原词不归并", func(t *testing.T) {
		resetTables(t)
		svc := NewDictionary(requireTestDB(t))
		inflected := uniqWord(t, "constraints")
		// 变化表指向的原形没有入库：归并目标缺失，保留原词。
		inflectedEntry := testEntry(inflected, inflected, map[string]string{"0": uniqWord(t, "constraint")}, nil, nil)
		mustInsertEntry(t, inflectedEntry)

		got, err := svc.Lookup(ctx, inflected)
		require.NoError(t, err)
		assert.Equal(t, inflectedEntry.ID, got.ID)
	})

	t.Run("变化表指向自身或缺失时保留原词", func(t *testing.T) {
		resetTables(t)
		svc := NewDictionary(requireTestDB(t))
		word := uniqWord(t, "saw")
		selfEntry := testEntry(word, word, map[string]string{"0": word}, nil, nil)
		mustInsertEntry(t, selfEntry)
		noExchange := testEntry(uniqWord(t, "plain"), uniqWord(t, "plain"), nil, nil, nil)
		mustInsertEntry(t, noExchange)

		gotSelf, err := svc.Lookup(ctx, word)
		require.NoError(t, err)
		assert.Equal(t, selfEntry.ID, gotSelf.ID)

		plain := uniqWord(t, "plain")
		gotPlain, err := svc.Lookup(ctx, plain)
		require.NoError(t, err)
		assert.Equal(t, noExchange.ID, gotPlain.ID)
	})

	t.Run("未命中时创建仅含 headword 的最小词条", func(t *testing.T) {
		resetTables(t)
		svc := NewDictionary(requireTestDB(t))
		word := uniqWord(t, "mvpword")

		got, err := svc.Lookup(ctx, word)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, word, got.Headword)
		assert.Equal(t, word, got.Lemma, "无法确定 lemma 的输入保留原词")
		assert.Empty(t, got.RawMeanings)
		assert.Empty(t, got.ReviewMeanings)

		again, err := svc.Lookup(ctx, word)
		require.NoError(t, err)
		assert.Equal(t, got.ID, again.ID, "再次 Lookup 应命中已创建的词条")
	})

	t.Run("归一后为空时返回参数错误", func(t *testing.T) {
		resetTables(t)
		svc := NewDictionary(requireTestDB(t))

		_, err := svc.Lookup(ctx, "   ")
		require.Error(t, err)
		var appErr *apperr.Error
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, apperr.InvalidArgument, appErr.Kind)
		assert.Equal(t, apperr.CodeValidationFailed, appErr.Code)
	})
}

// ---- 集成测试公共辅助 ----

// uniqWord 生成用例内唯一的小写词（集成用例共享数据库，按用例名隔离）。
func uniqWord(t *testing.T, suffix string) string {
	t.Helper()
	var b []rune
	for _, r := range []rune(t.Name()) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b = append(b, r)
		default:
			b = append(b, '-')
		}
	}
	return string(b) + "-" + suffix
}

// testEntry 构造词条（不经工厂，便于自定义 lemma / exchange / 释义）。
func testEntry(headword, lemma string, exchange map[string]string, raw, review []domain.Meaning) *domain.DictionaryEntry {
	now := time.Now().UTC().Truncate(time.Microsecond)
	return &domain.DictionaryEntry{
		ID:             uuid.Must(uuid.NewV7()),
		Headword:       headword,
		Lemma:          lemma,
		PhoneticUK:     "/uk/",
		PhoneticUS:     "/us/",
		RawMeanings:    raw,
		ReviewMeanings: review,
		Exchange:       exchange,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// mustInsertEntry 直插词条（预置带 exchange / 释义的词典数据，
// 绕过 Lookup 的创建路径）。
func mustInsertEntry(t *testing.T, entry *domain.DictionaryEntry) {
	t.Helper()
	require.NoError(t, repo.NewDictionaryRepo(testDB).Insert(context.Background(), entry))
}
