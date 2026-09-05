// Package domain 承载不依赖外部资源的领域模型与纯业务规则
// （docs/specs/backend/Go 单体应用架构规范.md §3）。
// 结构体不携带 bun struct tag / bun.BaseModel，方法只操作自身字段。
package domain

// TODO: domain 全部按 docs/dictionary/data-model.md 与
// docs/review/data-model.md 落模型后实现；不变量错误见 errors.go。
