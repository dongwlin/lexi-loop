package apperr

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKindString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		kind Kind
		want string
	}{
		{InvalidArgument, "InvalidArgument"},
		{Unauthenticated, "Unauthenticated"},
		{PermissionDenied, "PermissionDenied"},
		{NotFound, "NotFound"},
		{Conflict, "Conflict"},
		{FailedPrecondition, "FailedPrecondition"},
		{RateLimited, "RateLimited"},
		{InternalKind, "Internal"},
		{Kind(0), "UnknownKind"},
		{Kind(99), "UnknownKind"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, tt.kind.String())
	}
}

func TestCodes(t *testing.T) {
	t.Parallel()

	// 业务 code 是客户端分支的稳定契约（HTTP API 设计规范 §4.1），字面量
	// 必须与规范表格和接口文档一致：docs/api/words.md §4、docs/api/reviews.md §2/§5。
	assert.Equal(t, "OK", CodeOK)
	assert.Equal(t, "ERROR", CodeInternal)
	assert.Equal(t, "BASE.PARAM.VALIDATION_FAILED", CodeValidationFailed)
	assert.Equal(t, "BASE.AUTH.TOKEN_EXPIRED", CodeTokenExpired)
	assert.Equal(t, "BASE.AUTH.FORBIDDEN", CodeForbidden)
	assert.Equal(t, "BASE.NOT_FOUND.USER", CodeNotFound)
	assert.Equal(t, "BASE.BIZ.CONCURRENT_UPDATE", CodeConcurrentUpdate)
	assert.Equal(t, "BASE.BIZ.RATE_LIMITED", CodeRateLimited)
	assert.Equal(t, "BASE.BIZ.USER_DISABLED", CodeNoReviewableWords)
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("字段原样保留并保留 cause", func(t *testing.T) {
		t.Parallel()

		cause := errors.New("repo: not found")
		err := New(NotFound, CodeNotFound, "word not found", cause)

		assert.Equal(t, NotFound, err.Kind)
		assert.Equal(t, CodeNotFound, err.Code)
		assert.Equal(t, "word not found", err.Message)
		assert.Nil(t, err.Data)
		assert.Zero(t, err.RetryAfter)
		assert.ErrorIs(t, err, cause, "Unwrap 应保留 cause 供 errors.Is 穿透")
	})

	t.Run("无 cause 时 Error 文本只有 message", func(t *testing.T) {
		t.Parallel()

		err := New(InvalidArgument, CodeValidationFailed, "count must be positive", nil)

		assert.Nil(t, err.Unwrap())
		assert.Equal(t, "count must be positive", err.Error())
	})

	t.Run("有 cause 时 Error 文本含 cause 便于日志", func(t *testing.T) {
		t.Parallel()

		err := New(NotFound, CodeNotFound, "word not found", errors.New("repo: not found"))

		assert.Equal(t, "word not found: repo: not found", err.Error(),
			"Error 文本仅用于日志与调试；对外响应只使用 Message 字段")
	})
}

func TestInternal(t *testing.T) {
	t.Parallel()

	cause := errors.New("unexpected EOF")
	err := Internal(cause)

	assert.Equal(t, InternalKind, err.Kind)
	assert.Equal(t, CodeInternal, err.Code)
	assert.Equal(t, "internal server error", err.Message)
	assert.ErrorIs(t, err, cause)
}

func TestErrorsAsThroughWrap(t *testing.T) {
	t.Parallel()

	// 模拟跨层链路：Service 构造 apperr 后被 fmt.Errorf 包装，
	// Handler 侧仍可经 errors.As 还原 code / kind 并穿透到原始 cause。
	cause := errors.New("repo: not found")
	appErr := New(NotFound, CodeNotFound, "word not found", cause)
	wrapped := fmt.Errorf("GetWord: %w", appErr)

	var got *Error
	require.True(t, errors.As(wrapped, &got))
	assert.Equal(t, NotFound, got.Kind)
	assert.Equal(t, CodeNotFound, got.Code)
	assert.ErrorIs(t, wrapped, cause)
}
