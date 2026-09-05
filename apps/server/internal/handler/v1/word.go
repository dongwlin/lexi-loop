// Package v1 承载 /api/v1 版本化业务 Handler：只做参数绑定、基础校验、
// DTO 转换与 Service 调用，不写业务逻辑；响应一律经 httpresp 输出。
package v1

// TODO: WordHandler —— 生词导入 / 列表 / 详情 / 更新释义 / 软删除，
// 契约见 docs/api/words.md。
