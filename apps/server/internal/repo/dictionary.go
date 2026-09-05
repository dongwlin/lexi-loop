// Package repo 提供无状态数据访问适配器：构造即用即弃、仅依赖 bun.IDB，
// 返回 domain 类型，schema ↔ domain 转换只在 repo 内部完成。
package repo

// TODO: 实现 DictionaryRepo（查 / 写 dictionary_entries、lemma 查询），
// 数据访问错误按 SQLSTATE 归一化为 errors.go 中的可识别错误。
