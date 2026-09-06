package httpresp

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dongwlin/lexi-loop/apps/server/internal/apperr"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// rawResponse 只解出顶层契约字段；Data 用 RawMessage 保留原始 JSON 形态，
// 以便断言空对象 {} 与透传的数据。
type rawResponse struct {
	Code    string          `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func newTestContext(t *testing.T) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	return c, w
}

func decodeBody(t *testing.T, w *httptest.ResponseRecorder) rawResponse {
	t.Helper()

	var resp rawResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	return resp
}

func TestSuccess(t *testing.T) {
	t.Parallel()

	t.Run("携带数据时原样透传", func(t *testing.T) {
		t.Parallel()

		c, w := newTestContext(t)
		Success(c, gin.H{"list": []any{}})

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
		resp := decodeBody(t, w)
		assert.Equal(t, "OK", resp.Code)
		assert.Equal(t, "success", resp.Message)
		assert.JSONEq(t, `{"list":[]}`, string(resp.Data))
	})

	t.Run("无数据时渲染为空对象而非 null", func(t *testing.T) {
		t.Parallel()

		c, w := newTestContext(t)
		Success(c, nil)

		resp := decodeBody(t, w)
		assert.Equal(t, "OK", resp.Code)
		assert.Equal(t, "success", resp.Message)
		assert.JSONEq(t, `{}`, string(resp.Data), "空值规范 §5.1：无额外数据时 data 为 {}")
	})
}

func TestError(t *testing.T) {
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
			name:        "未识别 kind 按 500 兜底",
			err:         &apperr.Error{},
			wantStatus:  http.StatusInternalServerError,
			wantCode:    "",
			wantMessage: "",
		},
		{
			name:        "非应用错误按 500 且不暴露 cause",
			err:         errors.New("connection refused"),
			wantStatus:  http.StatusInternalServerError,
			wantCode:    "ERROR",
			wantMessage: "internal server error",
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

			c, w := newTestContext(t)
			Error(c, tt.err)

			assert.Equal(t, tt.wantStatus, w.Code)
			resp := decodeBody(t, w)
			assert.Equal(t, tt.wantCode, resp.Code)
			assert.Equal(t, tt.wantMessage, resp.Message)
			assert.JSONEq(t, `{}`, string(resp.Data), "失败响应未携带细节时 data 为 {}")
			assert.Equal(t, tt.wantRetryAfter, w.Header().Get("Retry-After"))
		})
	}

	t.Run("cause 不出现在响应体", func(t *testing.T) {
		t.Parallel()

		c, w := newTestContext(t)
		Error(c, apperr.Internal(errors.New("secret connection string")))

		assert.NotContains(t, w.Body.String(), "secret")
	})
}

func TestAbortError(t *testing.T) {
	t.Parallel()

	c, w := newTestContext(t)
	AbortError(c, apperr.New(apperr.Unauthenticated, apperr.CodeTokenExpired, "token expired", nil))

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.True(t, c.IsAborted(), "AbortError 必须中断后续中间件与 Handler 链")
	resp := decodeBody(t, w)
	assert.Equal(t, "BASE.AUTH.TOKEN_EXPIRED", resp.Code)
}

func TestErrorDataPassthrough(t *testing.T) {
	t.Parallel()

	c, w := newTestContext(t)
	err := apperr.New(apperr.InvalidArgument, apperr.CodeValidationFailed, "invalid parameter", nil)
	err.Data = gin.H{"fieldErrors": []gin.H{{"field": "count", "reason": "must be positive"}}}
	Error(c, err)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := decodeBody(t, w)
	assert.JSONEq(t, `{"fieldErrors":[{"field":"count","reason":"must be positive"}]}`, string(resp.Data),
		"错误细节经 apperr.Error.Data 进入 data（HTTP API 设计规范 §3.2）")
}
