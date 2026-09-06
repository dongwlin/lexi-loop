// Package httpresp 提供统一响应输出与唯一的 apperr.Kind → HTTP 状态映射
// （docs/backend/structure.md §4.4；docs/specs/backend/HTTP API 设计规范.md §11）。
//
// 业务 Handler 与 Middleware 的成功 / 失败响应一律经由本包输出，
// 不得各自复制响应结构（code/message/data）或状态映射。
// /healthz 等基础设施端点不受此约束。
package httpresp

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/dongwlin/lexi-loop/apps/server/internal/apperr"
	"github.com/gin-gonic/gin"
)

// Response 是全站统一响应结构：code 是客户端可分支处理的稳定机器契约，
// message 仅用于展示，data 在无数据时渲染为空对象 {}（HTTP API 设计规范
// §1、§3、§5）。
type Response struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

// Success 输出 200 成功响应，message 固定为 success；data 为 nil 时按空
// 对象 {} 渲染。列表为空返回 []、分页字段归入 pagination 由 DTO 保证，
// 本包不做结构转换。
func Success(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Response{
		Code:    apperr.CodeOK,
		Message: "success",
		Data:    emptyObjectIfNil(data),
	})
}

// Error 将 err 渲染为统一失败响应：apperr.Error 按其 kind 映射 HTTP 状态，
// 原样返回 code 与 message；非应用错误与未识别 kind 一律按 500 的
// CodeInternal 处理，不向客户端暴露 cause（docs/specs/backend/Go 单体应用
// 架构规范.md §7.2）。
func Error(c *gin.Context, err error) {
	renderError(c, err, false)
}

// AbortError 行为同 Error，并中断后续中间件与 Handler 链，供 Middleware 使用。
func AbortError(c *gin.Context, err error) {
	renderError(c, err, true)
}

func renderError(c *gin.Context, err error, abort bool) {
	var appErr *apperr.Error
	if !errors.As(err, &appErr) {
		appErr = apperr.Internal(err)
	}
	if appErr.RetryAfter > 0 {
		c.Header("Retry-After", retryAfterSeconds(appErr.RetryAfter))
	}
	c.JSON(statusFor(appErr.Kind), Response{
		Code:    appErr.Code,
		Message: appErr.Message,
		Data:    emptyObjectIfNil(appErr.Data),
	})
	if abort {
		c.Abort()
	}
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

// emptyObjectIfNil 保证响应 data 不为 null（HTTP API 设计规范 §5.1）。
func emptyObjectIfNil(data any) any {
	if data == nil {
		return gin.H{}
	}
	return data
}
