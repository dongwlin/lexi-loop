// Package httpresp 提供统一响应输出与唯一的 apperr.Kind → HTTP 状态映射
// （docs/backend/structure.md §4.4；docs/specs/backend/HTTP API 设计规范.md §11）。
//
// 业务 Handler 与 Middleware 的成功 / 失败响应一律经由本包输出，
// 不得各自复制响应结构（code/message/data）或状态映射。
// /healthz 等基础设施端点不受此约束。
package httpresp

// TODO: 实现统一响应结构：
//
//	type Response struct {
//	    Code    string `json:"code"`
//	    Message string `json:"message"`
//	    Data    any    `json:"data"`
//	}
//
// 并提供 Success(c, data) / Error(c, *apperr.Error) 等辅助函数；
// Kind → HTTP 状态映射只存在于本包。
