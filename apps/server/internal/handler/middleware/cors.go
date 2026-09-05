// Package middleware 承载 HTTP 横切中间件（Handler 子包，不参与版本化）。
// 定义与挂载分离：本包只定义，由 handler/router.go 构造并挂载。
package middleware

// TODO: cors.go —— CORS 中间件。
