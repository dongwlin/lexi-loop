// Package apperr 集中维护类型化应用错误：稳定 code、错误 kind、安全 message
// 与原始 cause。本包不依赖 HTTP / Gin。
package apperr

// TODO: 实现 Kind（InvalidArgument / Unauthenticated / PermissionDenied /
// NotFound / Conflict / FailedPrecondition / RateLimited / InternalKind）、
// 稳定业务 code（如 BASE.PARAM.VALIDATION_FAILED、BASE.NOT_FOUND.*）与
// New / Internal 构造函数（docs/specs/backend/Go 单体应用架构规范.md §7）。
// 业务 code 一经发布保持稳定，新增或废弃需同步接口文档。
