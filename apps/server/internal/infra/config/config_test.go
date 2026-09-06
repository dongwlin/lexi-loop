package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_DatabaseDefaults(t *testing.T) {
	cfg, err := Load()
	require.NoError(t, err)

	// URL 无默认值：未配置时为空字符串，由组合根决定如何处理。
	assert.Empty(t, cfg.Database.URL)
	// 连接池参数缺省为 0，语义为采用 pgx 默认（database.Options 只合并非零值）。
	assert.Zero(t, cfg.Database.MaxConns)
	assert.Zero(t, cfg.Database.MinConns)
	assert.Zero(t, cfg.Database.MaxConnLifetime)
	assert.Zero(t, cfg.Database.MaxConnIdleTime)
}

func TestLoad_DatabaseEnvOverrides(t *testing.T) {
	t.Setenv("LEXI_DATABASE_URL", "postgres://lexi:lexi@localhost:5432/lexi_loop?sslmode=disable")
	t.Setenv("LEXI_DATABASE_MAX_CONNS", "10")
	t.Setenv("LEXI_DATABASE_MIN_CONNS", "2")
	t.Setenv("LEXI_DATABASE_MAX_CONN_LIFETIME", "5m")
	t.Setenv("LEXI_DATABASE_MAX_CONN_IDLE_TIME", "90s")

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "postgres://lexi:lexi@localhost:5432/lexi_loop?sslmode=disable", cfg.Database.URL)
	assert.Equal(t, 10, cfg.Database.MaxConns)
	assert.Equal(t, 2, cfg.Database.MinConns)
	assert.Equal(t, 5*time.Minute, cfg.Database.MaxConnLifetime)
	assert.Equal(t, 90*time.Second, cfg.Database.MaxConnIdleTime)
}

func TestLoad_EnvDoesNotAffectHTTP(t *testing.T) {
	t.Setenv("LEXI_DATABASE_URL", "postgres://lexi:lexi@localhost:5432/lexi_loop?sslmode=disable")

	cfg, err := Load()
	require.NoError(t, err)

	// 数据库环境变量不应影响 HTTP 配置（键互相独立）。
	assert.Equal(t, DefaultHTTPAddr, cfg.HTTP.Addr)
}

func TestLoad_CORS默认本地开发源(t *testing.T) {
	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, DefaultCORSAllowedOrigins, cfg.HTTP.CORSAllowedOrigins)
}

func TestLoad_CORSEnvOverrides(t *testing.T) {
	t.Setenv("LEXI_HTTP_CORS_ALLOWED_ORIGINS", "https://lexi.example.com, http://localhost:5173 ,")

	cfg, err := Load()
	require.NoError(t, err)

	// 逗号分隔、逐项去首尾空白、丢弃空项。
	assert.Equal(t, []string{"https://lexi.example.com", "http://localhost:5173"},
		cfg.HTTP.CORSAllowedOrigins)
}

func TestLoad_ParseCSVList(t *testing.T) {
	assert.Empty(t, parseCSVList(""))
	assert.Equal(t, []string{"a", "b"}, parseCSVList(" a ,b"))
	assert.Empty(t, parseCSVList(" , , "))
}
