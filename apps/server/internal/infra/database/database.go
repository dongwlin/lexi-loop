// Package database 提供 PostgreSQL 数据库基础设施（纯技术组件，无业务逻辑），
// 由组合根（internal/app）组装。
//
// 实现：pgxpool + pgx/v5/stdlib + bun（docs/specs/backend/Go 技术栈.md）。
// 使用 testcontainers + 真实 PostgreSQL 验证（docs/specs/backend/Go 测试规范.md）。
package database

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

// Options 保存数据库连接配置。
type Options struct {
	// URL 是 PostgreSQL 连接串，如
	// postgres://user:pass@localhost:5432/lexi_loop?sslmode=disable。
	URL string

	// 连接池参数。0 值保持 pgx 默认（mergePoolOptions 只合并非零取值）。
	MaxConns        int
	MinConns        int
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
}

// New 使用 pgx 驱动创建 *bun.DB，并验证连接可用性。
//
// 注意：*bun.DB.Close 只释放 database/sql 包装层；由 OpenDBFromPool 包装的
// pgxpool 不随其关闭，进程退出时由操作系统回收。
func New(ctx context.Context, opts Options) (*bun.DB, error) {
	poolConfig, err := pgxpool.ParseConfig(opts.URL)
	if err != nil {
		return nil, err
	}

	mergePoolOptions(poolConfig, opts)

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, err
	}

	sqldb := stdlib.OpenDBFromPool(pool)
	db := bun.NewDB(sqldb, pgdialect.New())

	// 验证连接：失败时关闭 wrapper 并释放连接池，避免连接泄漏。
	if err := db.PingContext(ctx); err != nil {
		_ = sqldb.Close()
		pool.Close()
		return nil, err
	}

	return db, nil
}

// mergePoolOptions 将 opts 中非零的取值合并进 cfg；0 值保持 pgx 默认。
func mergePoolOptions(cfg *pgxpool.Config, opts Options) {
	if opts.MaxConns > 0 {
		cfg.MaxConns = int32(opts.MaxConns)
	}
	if opts.MinConns > 0 {
		cfg.MinConns = int32(opts.MinConns)
	}
	if opts.MaxConnLifetime > 0 {
		cfg.MaxConnLifetime = opts.MaxConnLifetime
	}
	if opts.MaxConnIdleTime > 0 {
		cfg.MaxConnIdleTime = opts.MaxConnIdleTime
	}
}
