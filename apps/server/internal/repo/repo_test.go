package repo

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeError(t *testing.T) {
	t.Run("nil 错误原样返回", func(t *testing.T) {
		assert.NoError(t, normalizeError("op", nil))
	})

	t.Run("查无记录归一化为 ErrNotFound", func(t *testing.T) {
		err := normalizeError("find entry", sql.ErrNoRows)
		assert.ErrorIs(t, err, ErrNotFound)
		assert.Contains(t, err.Error(), "find entry")
	})

	t.Run("唯一约束按 SQLSTATE 归一化为 ErrAlreadyExists", func(t *testing.T) {
		pgErr := &pgconn.PgError{
			Code:           "23505",
			Message:        "duplicate key value violates unique constraint",
			ConstraintName: "dictionary_entries_headword_key",
		}
		err := normalizeError("insert entry", pgErr)
		assert.ErrorIs(t, err, ErrAlreadyExists)
		assert.Contains(t, err.Error(), "dictionary_entries_headword_key")
	})

	t.Run("外键约束归一化为 ErrReferenceViolation", func(t *testing.T) {
		pgErr := &pgconn.PgError{Code: "23503", ConstraintName: "review_items_session_id_fkey"}
		err := normalizeError("insert item", pgErr)
		assert.ErrorIs(t, err, ErrReferenceViolation)
	})

	t.Run("序列化失败归一化为 ErrConflict", func(t *testing.T) {
		err := normalizeError("run tx", &pgconn.PgError{Code: "40001"})
		assert.ErrorIs(t, err, ErrConflict)
	})

	t.Run("死锁归一化为 ErrConflict", func(t *testing.T) {
		err := normalizeError("run tx", &pgconn.PgError{Code: "40P01"})
		assert.ErrorIs(t, err, ErrConflict)
	})

	t.Run("未预期错误保留原始 cause", func(t *testing.T) {
		cause := errors.New("connection reset")
		err := normalizeError("query", cause)
		assert.NotErrorIs(t, err, ErrNotFound)
		assert.ErrorIs(t, err, cause)
	})
}

func TestEscapeLike(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"普通单词不变", "ambiguous", "ambiguous"},
		{"空串不变", "", ""},
		{"百分号被转义", "50%", `50\%`},
		{"下划线被转义", "a_b", `a\_b`},
		{"反斜杠被转义", `a\b`, `a\\b`},
		{"组合字符全部转义", `%_\`, `\%\_\\`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, escapeLike(tt.input))
		})
	}
}

func TestRequireAffected(t *testing.T) {
	t.Run("零行更新归一化为 ErrNotFound", func(t *testing.T) {
		res := newFakeSQLResult(0)
		err := requireAffected(res, "soft delete")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("命中行不报错", func(t *testing.T) {
		assert.NoError(t, requireAffected(newFakeSQLResult(1), "soft delete"))
	})
}

// fakeSQLResult 是 requireAffected 的最小 sql.Result 替身（标准库接口的
// 数据结构替身，不涉及 Repo / 数据库 mock）。
type fakeSQLResult struct{ rows int64 }

func newFakeSQLResult(rows int64) fakeSQLResult { return fakeSQLResult{rows: rows} }

func (r fakeSQLResult) LastInsertId() (int64, error) { return 0, nil }
func (r fakeSQLResult) RowsAffected() (int64, error) { return r.rows, nil }
