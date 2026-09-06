// Package apperr 集中维护类型化应用错误：稳定 code、错误 kind、安全 message
// 与原始 cause。本包不依赖 HTTP / Gin，供 Service 将 Domain / Repo 的可识别
// 错误映射为 Error，Handler / Middleware 经 errors.As 识别
// （docs/specs/backend/Go 单体应用架构规范.md §7）。
package apperr

import "time"

// Kind 表达应用错误的类别，是 apperr.Error → HTTP 状态码映射的唯一依据；
// 映射只存在于 handler/httpresp（docs/specs/backend/HTTP API 设计规范.md §11）。
type Kind uint8

// Kind 取值与 HTTP 状态码的对应关系见 HTTP API 设计规范 §11。
const (
	InvalidArgument    Kind = iota + 1 // 请求参数格式或基础校验错误（400）
	Unauthenticated                    // 未登录、凭证无效或认证失效（401）
	PermissionDenied                   // 已认证，但无当前操作权限（403）
	NotFound                           // 路由或业务资源不存在（404）
	Conflict                           // 资源状态与请求冲突，如并发版本冲突（409）
	FailedPrecondition                 // 请求格式正确，但不满足业务前置条件（422）
	RateLimited                        // 请求频率或业务额度受限（429）
	InternalKind                       // 未预期内部异常（500）；命名避开构造函数 Internal
)

// kindNames 供日志与调试使用，不是稳定契约。
var kindNames = [...]string{
	InvalidArgument:    "InvalidArgument",
	Unauthenticated:    "Unauthenticated",
	PermissionDenied:   "PermissionDenied",
	NotFound:           "NotFound",
	Conflict:           "Conflict",
	FailedPrecondition: "FailedPrecondition",
	RateLimited:        "RateLimited",
	InternalKind:       "Internal",
}

// String 返回 kind 的可读名称，仅用于日志与调试。
func (k Kind) String() string {
	if k >= InvalidArgument && k <= InternalKind {
		return kindNames[k]
	}
	return "UnknownKind"
}

// 稳定业务 code（docs/specs/backend/HTTP API 设计规范.md §4.1）：客户端可
// 分支处理的机器契约，格式为三段式 BASE.MODULE.DETAIL 或固定值 OK / ERROR；
// 不使用数字或带 HTTP 含义的编码。一经发布保持稳定，新增或废弃需同步接口
// 文档；禁止各 Handler 临时发明字面量。
const (
	// CodeOK 成功，仅由 handler/httpresp 的成功响应使用。
	CodeOK = "OK"
	// CodeInternal 未分类系统错误；message 不携带内部细节，cause 只进日志。
	CodeInternal = "ERROR"

	// 以下为 HTTP API 设计规范 §4.1 表格中的标准业务 code。
	// CodeValidationFailed 参数校验失败（400）。
	CodeValidationFailed = "BASE.PARAM.VALIDATION_FAILED"
	// CodeTokenExpired 认证失效（401）。
	CodeTokenExpired = "BASE.AUTH.TOKEN_EXPIRED"
	// CodeForbidden 权限不足（403）。
	CodeForbidden = "BASE.AUTH.FORBIDDEN"
	// CodeNotFound 资源不存在（404）。MVP 契约中生词与复习 session 的未找到
	// 均返回该 code（docs/api/words.md §4、docs/api/reviews.md §5）。
	CodeNotFound = "BASE.NOT_FOUND.USER"
	// CodeConcurrentUpdate 资源版本冲突（409）。
	CodeConcurrentUpdate = "BASE.BIZ.CONCURRENT_UPDATE"
	// CodeRateLimited 请求频率或业务额度受限（429）。
	CodeRateLimited = "BASE.BIZ.RATE_LIMITED"
	// CodeNoReviewableWords 无可复习生词（422）；契约字面量即标准 code
	// BASE.BIZ.USER_DISABLED（docs/api/reviews.md §2）。
	CodeNoReviewableWords = "BASE.BIZ.USER_DISABLED"
)

// Error 是携带稳定 code 与 kind 的类型化应用错误。跨层稳定契约是错误类型、
// kind 与 code，不是 Error() 文本；Message 只用于安全展示，可本地化或调整。
type Error struct {
	Kind       Kind          // 决定 HTTP 状态（映射在 handler/httpresp）
	Code       string        // 稳定业务 code，客户端分支依据
	Message    string        // 面向客户端的安全提示
	RetryAfter time.Duration // 仅 RateLimited 使用；> 0 时响应输出 Retry-After
	Data       any           // 错误细节（如 fieldErrors，HTTP API 设计规范 §3.2）；nil 时响应 data 为 {}
	Err        error         // 原始 cause，仅用于日志，不得返回给客户端
}

// New 构造类型化应用错误。cause 为被映射的 Domain / Repo 错误或底层失败，
// 仅保留在错误链中供日志排查，可为 nil。
func New(kind Kind, code, message string, cause error) *Error {
	return &Error{Kind: kind, Code: code, Message: message, Err: cause}
}

// Internal 将未预期错误统一包装为内部错误：客户端只见 CodeInternal 与通用
// message，cause 保留在错误链中供日志（docs/specs/backend/Go 单体应用架构
// 规范.md §7.2）。
func Internal(cause error) *Error {
	return New(InternalKind, CodeInternal, "internal server error", cause)
}

// Error 实现 error 接口；文本含 cause 以便日志排查，不是稳定契约。
func (e *Error) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

// Unwrap 保留 cause 链，使 errors.Is / errors.As 可穿透 Error 识别底层错误。
func (e *Error) Unwrap() error {
	return e.Err
}
