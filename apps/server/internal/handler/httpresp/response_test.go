package httpresp

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/dongwlin/lexi-loop/apps/server/internal/apperr"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// rawEnvelope 只解出顶层契约字段；Data 用 RawMessage 保留原始 JSON 形态，
// 以便断言空对象 {} 与字段级错误细节。
type rawEnvelope struct {
	Code    string          `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func decodeModel(t *testing.T, m any) rawEnvelope {
	t.Helper()

	raw, err := json.Marshal(m)
	require.NoError(t, err)
	var env rawEnvelope
	require.NoError(t, json.Unmarshal(raw, &env))
	return env
}

func newTestContext(t *testing.T) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	return c, w
}

func TestOK(t *testing.T) {
	t.Parallel()

	t.Run("携带数据时原样透传", func(t *testing.T) {
		t.Parallel()

		env := decodeModel(t, OK(gin.H{"list": []any{}}))
		assert.Equal(t, "OK", env.Code)
		assert.Equal(t, "success", env.Message)
		assert.JSONEq(t, `{"list":[]}`, string(env.Data))
	})

	t.Run("无数据渲染为空对象而非 null", func(t *testing.T) {
		t.Parallel()

		env := decodeModel(t, OK(struct{}{}))
		assert.Equal(t, "OK", env.Code)
		assert.JSONEq(t, `{}`, string(env.Data), "空值规范 §5.1：无额外数据时 data 为 {}")
	})
}

func TestFromError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		err            error
		wantStatus     int
		wantCode       string
		wantMessage    string
		wantRetryAfter string
	}{
		{
			name:        "参数校验失败映射 400",
			err:         apperr.New(apperr.InvalidArgument, apperr.CodeValidationFailed, "invalid parameter", nil),
			wantStatus:  http.StatusBadRequest,
			wantCode:    "BASE.PARAM.VALIDATION_FAILED",
			wantMessage: "invalid parameter",
		},
		{
			name:        "认证失效映射 401",
			err:         apperr.New(apperr.Unauthenticated, apperr.CodeTokenExpired, "token expired", nil),
			wantStatus:  http.StatusUnauthorized,
			wantCode:    "BASE.AUTH.TOKEN_EXPIRED",
			wantMessage: "token expired",
		},
		{
			name:        "权限不足映射 403",
			err:         apperr.New(apperr.PermissionDenied, apperr.CodeForbidden, "forbidden", nil),
			wantStatus:  http.StatusForbidden,
			wantCode:    "BASE.AUTH.FORBIDDEN",
			wantMessage: "forbidden",
		},
		{
			name:        "资源不存在映射 404",
			err:         apperr.New(apperr.NotFound, apperr.CodeNotFound, "word not found", nil),
			wantStatus:  http.StatusNotFound,
			wantCode:    "BASE.NOT_FOUND.USER",
			wantMessage: "word not found",
		},
		{
			name:        "资源冲突映射 409",
			err:         apperr.New(apperr.Conflict, apperr.CodeConcurrentUpdate, "resource was modified", nil),
			wantStatus:  http.StatusConflict,
			wantCode:    "BASE.BIZ.CONCURRENT_UPDATE",
			wantMessage: "resource was modified",
		},
		{
			name:        "业务前置条件不满足映射 422",
			err:         apperr.New(apperr.FailedPrecondition, apperr.CodeNoReviewableWords, "no reviewable words available", nil),
			wantStatus:  http.StatusUnprocessableEntity,
			wantCode:    "BASE.BIZ.USER_DISABLED",
			wantMessage: "no reviewable words available",
		},
		{
			name: "限流映射 429 并输出整秒 Retry-After",
			err: &apperr.Error{
				Kind:       apperr.RateLimited,
				Code:       apperr.CodeRateLimited,
				Message:    "rate limited",
				RetryAfter: 90 * time.Second,
			},
			wantStatus:     http.StatusTooManyRequests,
			wantCode:       "BASE.BIZ.RATE_LIMITED",
			wantMessage:    "rate limited",
			wantRetryAfter: "90",
		},
		{
			name: "Retry-After 不足一秒时向上取整",
			err: &apperr.Error{
				Kind:       apperr.RateLimited,
				Code:       apperr.CodeRateLimited,
				Message:    "rate limited",
				RetryAfter: 1500 * time.Millisecond,
			},
			wantStatus:     http.StatusTooManyRequests,
			wantCode:       "BASE.BIZ.RATE_LIMITED",
			wantMessage:    "rate limited",
			wantRetryAfter: "2",
		},
		{
			name:           "非应用错误按 500 且不暴露 cause",
			err:            errors.New("connection refused"),
			wantStatus:     http.StatusInternalServerError,
			wantCode:       "ERROR",
			wantMessage:    "internal server error",
			wantRetryAfter: "",
		},
		{
			name:        "apperr.Internal 同样不暴露 cause",
			err:         apperr.Internal(errors.New("unexpected EOF")),
			wantStatus:  http.StatusInternalServerError,
			wantCode:    "ERROR",
			wantMessage: "internal server error",
		},
		{
			name:        "nil 错误防御性按 500 处理",
			err:         nil,
			wantStatus:  http.StatusInternalServerError,
			wantCode:    "ERROR",
			wantMessage: "internal server error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := FromError(tt.err)

			require.Equal(t, tt.wantStatus, m.GetStatus())
			env := decodeModel(t, m)
			assert.Equal(t, tt.wantCode, env.Code)
			assert.Equal(t, tt.wantMessage, env.Message)
			assert.JSONEq(t, `{}`, string(env.Data), "失败响应未携带细节时 data 为 {}")
			headers := m.GetHeaders()
			assert.Equal(t, tt.wantRetryAfter, headers.Get("Retry-After"))
		})
	}

	t.Run("cause 不出现在响应体", func(t *testing.T) {
		t.Parallel()

		m := FromError(apperr.Internal(errors.New("secret connection string")))
		raw, err := json.Marshal(m)
		require.NoError(t, err)
		assert.NotContains(t, string(raw), "secret")
	})

	t.Run("未识别 kind 按 500 兜底", func(t *testing.T) {
		t.Parallel()

		m := FromError(&apperr.Error{})
		assert.Equal(t, http.StatusInternalServerError, m.GetStatus())
	})
}

func TestFromErrorDataPassthrough(t *testing.T) {
	t.Parallel()

	err := apperr.New(apperr.InvalidArgument, apperr.CodeValidationFailed, "invalid parameter", nil)
	err.Data = []FieldError{{Field: "count", Reason: "must be positive"}}
	m := FromError(err)

	assert.Equal(t, http.StatusBadRequest, m.GetStatus())
	env := decodeModel(t, m)
	assert.JSONEq(t, `{"fieldErrors":[{"field":"count","reason":"must be positive"}]}`, string(env.Data),
		"错误细节经 apperr.Error.Data 进入 data（HTTP API 设计规范 §3.2）")
}

func TestNewHumaError(t *testing.T) {
	t.Parallel()

	t.Run("校验失败 422 降为 400 且映射字段级错误", func(t *testing.T) {
		t.Parallel()

		m := NewHumaError(http.StatusUnprocessableEntity, "validation failed",
			&huma.ErrorDetail{Message: "expected number", Location: "body.words.0.count", Value: "abc"},
			&huma.ErrorDetail{Message: "must be at least 1", Location: "query.page"},
		)

		require.Equal(t, http.StatusBadRequest, m.GetStatus(), "项目 422 已被 FailedPrecondition 业务前置条件占用")
		env := decodeModel(t, m)
		assert.Equal(t, "BASE.PARAM.VALIDATION_FAILED", env.Code)
		assert.Equal(t, "invalid parameter", env.Message)
		var data struct {
			FieldErrors []struct {
				Field  string `json:"field"`
				Reason string `json:"reason"`
			} `json:"fieldErrors"`
		}
		require.NoError(t, json.Unmarshal(env.Data, &data))
		require.Len(t, data.FieldErrors, 2)
		assert.Equal(t, "words.0.count", data.FieldErrors[0].Field, "location 去掉 body 前缀")
		assert.Equal(t, "expected number", data.FieldErrors[0].Reason)
		assert.Equal(t, "page", data.FieldErrors[1].Field, "location 去掉 query 前缀")
		assert.Empty(t, m.GetHeaders().Get("Retry-After"), "校验错误不携带 Retry-After")
	})

	t.Run("畸形请求体与必填请求体映射 400", func(t *testing.T) {
		t.Parallel()

		m := NewHumaError(http.StatusBadRequest, "request body is required")
		require.Equal(t, http.StatusBadRequest, m.GetStatus())
		env := decodeModel(t, m)
		assert.Equal(t, "BASE.PARAM.VALIDATION_FAILED", env.Code)
		assert.JSONEq(t, `{"fieldErrors":[{"field":"body","reason":"request body is required"}]}`, string(env.Data))
	})

	t.Run("415 与 406 同属参数类错误归入 400", func(t *testing.T) {
		t.Parallel()

		for _, status := range []int{http.StatusUnsupportedMediaType, http.StatusNotAcceptable} {
			m := NewHumaError(status, "unsupported media type",
				&huma.ErrorDetail{Message: "unsupported content type", Location: "body"})
			assert.Equal(t, http.StatusBadRequest, m.GetStatus())
			assert.Equal(t, "BASE.PARAM.VALIDATION_FAILED", decodeModel(t, m).Code)
		}
	})

	t.Run("未预期错误按内部错误渲染且不透传细节", func(t *testing.T) {
		t.Parallel()

		m := NewHumaError(http.StatusInternalServerError, "unexpected error occurred", errors.New("secret cause"))
		require.Equal(t, http.StatusInternalServerError, m.GetStatus())
		env := decodeModel(t, m)
		assert.Equal(t, "ERROR", env.Code)
		assert.Equal(t, "internal server error", env.Message)
		assert.JSONEq(t, `{}`, string(env.Data))
		raw, err := json.Marshal(m)
		require.NoError(t, err)
		assert.NotContains(t, string(raw), "secret")
	})

	t.Run("status 0 的样例模型仅取类型", func(t *testing.T) {
		t.Parallel()

		m := NewHumaError(0, "")
		require.NotNil(t, m)
		env := decodeModel(t, m)
		assert.NotEmpty(t, env.Code)
	})
}

func TestAbortError(t *testing.T) {
	t.Parallel()

	c, w := newTestContext(t)
	AbortError(c, apperr.New(apperr.Unauthenticated, apperr.CodeTokenExpired, "token expired", nil))

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.True(t, c.IsAborted(), "AbortError 必须中断后续中间件与 Handler 链")
	var env rawEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.Equal(t, "BASE.AUTH.TOKEN_EXPIRED", env.Code)
	assert.Equal(t, "", w.Header().Get("Retry-After"))

	t.Run("限流错误透传 Retry-After 响应头", func(t *testing.T) {
		c, w := newTestContext(t)
		AbortError(c, &apperr.Error{
			Kind:       apperr.RateLimited,
			Code:       apperr.CodeRateLimited,
			Message:    "rate limited",
			RetryAfter: 30 * time.Second,
		})

		assert.Equal(t, http.StatusTooManyRequests, w.Code)
		assert.Equal(t, "30", w.Header().Get("Retry-After"))
	})
}
