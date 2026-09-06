// Package httpresp 提供统一响应结构（code/message/data 外壳）与唯一的
// apperr.Kind → HTTP 状态映射（docs/backend/structure.md §4.4；
// docs/specs/backend/HTTP API 设计规范.md §11）。
//
// 业务 Handler（huma 操作）与 Middleware 的失败响应一律经由本包模型渲染，
// 不得各自复制响应结构或状态映射。/healthz 等基础设施端点不受此约束。
package httpresp

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gin-gonic/gin"

	"github.com/dongwlin/lexi-loop/apps/server/internal/apperr"
)

// Envelope 是成功响应外壳：code 是客户端可分支处理的稳定机器契约，
// message 仅用于展示，data 在无数据时渲染为空对象 {}（HTTP API 设计规范
// §1、§3.1、§5.1）。T 为 dto 定义的具体数据契约。
type Envelope[T any] struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

// OK 构造成功外壳：code 固定 OK、message 固定 success；无数据的端点传
// dto.NoData{}（渲染为 {}，空值规范 §5.1）。
func OK[T any](data T) Envelope[T] {
	return Envelope[T]{Code: apperr.CodeOK, Message: "success", Data: data}
}

// FieldError 是字段级校验错误细节（HTTP API 设计规范 §3.2）。
type FieldError struct {
	Field  string `json:"field"`
	Reason string `json:"reason"`
}

// ErrorData 是失败响应 data 的形态：仅承载字段级错误细节，无细节时渲染
// 为 {}。
type ErrorData struct {
	FieldErrors []FieldError `json:"fieldErrors,omitempty"`
}

// Error 是 huma 渲染的错误响应模型：外壳与 Envelope 一致，业务 code 由
// apperr 携带。status 与 retryAfter 不进响应体；retryAfter 以 Retry-After
// 响应头透出（huma.HeadersError）。不实现 huma 的 ContentTypeFilter，
// Content-Type 保持 application/json（不使用 RFC 9457 problem+json）。
type Error struct {
	status     int
	retryAfter string
	Code       string    `json:"code"`
	Message    string    `json:"message"`
	Data       ErrorData `json:"data"`
}

// Error 实现 error 接口；文本仅供日志，不是稳定契约。
func (e *Error) Error() string {
	return e.Message
}

// GetStatus 返回 HTTP 状态码（huma.StatusError）。
func (e *Error) GetStatus() int {
	return e.status
}

// GetHeaders 返回错误附带的响应头；当前仅 RateLimited 的 Retry-After
// （huma.HeadersError，huma 写响应前经 appendErrorHeaders 透出）。
func (e *Error) GetHeaders() http.Header {
	if e.retryAfter == "" {
		return nil
	}
	return http.Header{"Retry-After": {e.retryAfter}}
}

// FromError 把错误渲染为错误响应模型：apperr.Error 按其 kind 映射 HTTP
// 状态，原样返回 code 与 message；错误细节仅支持 []FieldError 形态
// （HTTP API 设计规范 §3.2）。非应用错误与未识别 kind 一律按 500 的
// CodeInternal 处理，不向客户端暴露 cause（docs/specs/backend/Go 单体
// 应用架构规范.md §7.2）。
func FromError(err error) *Error {
	var appErr *apperr.Error
	if !errors.As(err, &appErr) {
		appErr = apperr.Internal(err)
	}
	m := &Error{
		status:  statusFor(appErr.Kind),
		Code:    appErr.Code,
		Message: appErr.Message,
	}
	if appErr.RetryAfter > 0 {
		m.retryAfter = retryAfterSeconds(appErr.RetryAfter)
	}
	if fieldErrors, ok := appErr.Data.([]FieldError); ok {
		m.Data.FieldErrors = fieldErrors
	}
	return m
}

// NewHumaError 是 huma 错误模型工厂，经 UseHumaError 装配为 huma.NewError，
// 覆盖 huma 默认的 RFC 9457 problem+json 模型：
//
//   - 请求参数 / 请求体校验失败（huma 默认 422）降为 400——项目 422 已被
//     FailedPrecondition 业务前置条件占用（HTTP API 设计规范 §11）；
//     415 / 406（媒体类型与响应协商不可接受）同属参数类错误一并归入；
//   - 错误体保持 {code, message, data} 外壳与字段级 data.fieldErrors；
//   - 其余 huma 内部错误（含未预期 500）按内部错误渲染，不透传细节。
//
// status 为 0 时仅用于 spec 生成的样例模型（取类型，与值无关）。
func NewHumaError(status int, msg string, errs ...error) *Error {
	switch status {
	case http.StatusBadRequest, http.StatusNotAcceptable, http.StatusUnsupportedMediaType, http.StatusUnprocessableEntity:
		return newValidationError(msg, errs)
	default:
		return internalError()
	}
}

// UseHumaError 把本包的 NewHumaError 装配为 huma 的错误模型工厂（重复
// 赋值幂等）。spec 中 components.schemas.Error 由 huma 按本模型类型派生，
// 文档与实际错误响应保持一致；构造 huma API 前必须调用一次。
func UseHumaError() {
	huma.NewError = func(status int, msg string, errs ...error) huma.StatusError {
		return NewHumaError(status, msg, errs...)
	}
}

// newValidationError 构造参数校验失败的错误模型：code 固定为
// BASE.PARAM.VALIDATION_FAILED、message 固定为 invalid parameter，字段级
// 细节取自 huma 的错误明细（location 去掉 body/query/path/header 前缀）。
func newValidationError(msg string, errs []error) *Error {
	m := &Error{
		status:  http.StatusBadRequest,
		Code:    apperr.CodeValidationFailed,
		Message: "invalid parameter",
	}
	for _, err := range errs {
		if err == nil {
			continue
		}
		var detailer huma.ErrorDetailer
		if errors.As(err, &detailer) {
			detail := detailer.ErrorDetail()
			m.Data.FieldErrors = append(m.Data.FieldErrors, FieldError{
				Field:  fieldName(detail.Location),
				Reason: detail.Message,
			})
			continue
		}
		m.Data.FieldErrors = append(m.Data.FieldErrors, FieldError{Field: "body", Reason: err.Error()})
	}
	// 无明细但携带原因时（如 "request body is required"），以 body 字段透出。
	if len(m.Data.FieldErrors) == 0 && msg != "" && msg != "validation failed" {
		m.Data.FieldErrors = append(m.Data.FieldErrors, FieldError{Field: "body", Reason: msg})
	}
	return m
}

// internalError 构造内部错误模型：客户端只见 CodeInternal 与通用 message。
func internalError() *Error {
	return &Error{
		status:  http.StatusInternalServerError,
		Code:    apperr.CodeInternal,
		Message: "internal server error",
	}
}

// fieldName 把 huma 的错误位置（如 body.words.0.count、query.page）转成
// 契约的 field：去掉来源前缀；无剩余部分时保留原位置（如 body）。
func fieldName(location string) string {
	for _, prefix := range []string{"body.", "query.", "path.", "header."} {
		if rest, ok := strings.CutPrefix(location, prefix); ok && rest != "" {
			return rest
		}
	}
	if location == "" {
		return "body"
	}
	return location
}

// AbortError 把 err 渲染为统一失败响应并中断后续中间件与 Handler 链，
// 供 Middleware 使用（业务端点的失败响应由 huma 经本包模型渲染）。
func AbortError(c *gin.Context, err error) {
	m := FromError(err)
	for key, values := range m.GetHeaders() {
		for _, value := range values {
			c.Header(key, value)
		}
	}
	c.JSON(m.GetStatus(), m)
	c.Abort()
}

// statusFor 是唯一的 apperr.Kind → HTTP 状态映射
// （docs/specs/backend/HTTP API 设计规范.md §11）。
func statusFor(kind apperr.Kind) int {
	switch kind {
	case apperr.InvalidArgument:
		return http.StatusBadRequest
	case apperr.Unauthenticated:
		return http.StatusUnauthorized
	case apperr.PermissionDenied:
		return http.StatusForbidden
	case apperr.NotFound:
		return http.StatusNotFound
	case apperr.Conflict:
		return http.StatusConflict
	case apperr.FailedPrecondition:
		return http.StatusUnprocessableEntity
	case apperr.RateLimited:
		return http.StatusTooManyRequests
	default:
		// InternalKind 与未识别 kind 一律按内部错误处理。
		return http.StatusInternalServerError
	}
}

// retryAfterSeconds 把时长换算为 Retry-After 头的整秒数（向上取整）。
func retryAfterSeconds(d time.Duration) string {
	return strconv.FormatInt(int64((d+time.Second-1)/time.Second), 10)
}
