package database

// TODO: migrate.go —— 执行 apps/server/migrations/ 版本化 SQL 的适配器。
// 应用启动不隐式执行迁移，由 migrate 命令显式执行；
// 不在 Go 代码中另存一份 schema（docs/backend/structure.md §2）。
