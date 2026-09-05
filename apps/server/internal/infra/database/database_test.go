package database

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 单元测试只验证解析与合并逻辑，不依赖外部 PostgreSQL。
// 连接池/连接验证的集成测试见 database_integration_test.go（testcontainers）。

func TestMergePoolOptions_ZeroOptionsKeepPgxDefaults(t *testing.T) {
	cfg, err := pgxpool.ParseConfig(testConnURL)
	require.NoError(t, err)
	want := *cfg

	mergePoolOptions(cfg, Options{})

	assert.Equal(t, want.MaxConns, cfg.MaxConns, "0 值不得改动 MaxConns")
	assert.Equal(t, want.MinConns, cfg.MinConns, "0 值不得改动 MinConns")
	assert.Equal(t, want.MaxConnLifetime, cfg.MaxConnLifetime, "0 值不得改动 MaxConnLifetime")
	assert.Equal(t, want.MaxConnIdleTime, cfg.MaxConnIdleTime, "0 值不得改动 MaxConnIdleTime")
}

func TestMergePoolOptions_AppliesNonZeroValues(t *testing.T) {
	cfg, err := pgxpool.ParseConfig(testConnURL)
	require.NoError(t, err)

	mergePoolOptions(cfg, Options{
		MaxConns:        10,
		MinConns:        2,
		MaxConnLifetime: 5 * time.Minute,
		MaxConnIdleTime: 90 * time.Second,
	})

	assert.Equal(t, int32(10), cfg.MaxConns)
	assert.Equal(t, int32(2), cfg.MinConns)
	assert.Equal(t, 5*time.Minute, cfg.MaxConnLifetime)
	assert.Equal(t, 90*time.Second, cfg.MaxConnIdleTime)
}

func TestMergePoolOptions_PartialOptionsKeepOtherDefaults(t *testing.T) {
	cfg, err := pgxpool.ParseConfig(testConnURL)
	require.NoError(t, err)
	want := *cfg

	mergePoolOptions(cfg, Options{MinConns: 3})

	assert.Equal(t, int32(3), cfg.MinConns)
	assert.Equal(t, want.MaxConns, cfg.MaxConns, "未设置字段保持 pgx 默认")
	assert.Equal(t, want.MaxConnLifetime, cfg.MaxConnLifetime, "未设置字段保持 pgx 默认")
	assert.Equal(t, want.MaxConnIdleTime, cfg.MaxConnIdleTime, "未设置字段保持 pgx 默认")
}

func TestNew_InvalidURL(t *testing.T) {
	db, err := New(context.Background(), Options{URL: "://not-a-valid-url"})
	assert.Nil(t, db)
	assert.Error(t, err, "非法连接串应返回解析错误")
}

func TestNew_UnreachableServer(t *testing.T) {
	// 127.0.0.1:1 端口必然拒绝连接，验证连接失败会返回错误而非悬挂。
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	db, err := New(ctx, Options{
		URL: "postgres://lexi:lexi@127.0.0.1:1/lexi_loop?sslmode=disable&connect_timeout=1",
	})
	assert.Nil(t, db)
	assert.Error(t, err, "连接不可达应返回错误")
}

const testConnURL = "postgres://lexi:lexi@localhost:5432/lexi_loop?sslmode=disable"
