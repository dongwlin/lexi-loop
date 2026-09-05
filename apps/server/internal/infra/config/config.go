// Package config 基于 viper 加载配置：默认值 + LEXI_* 环境变量覆盖。
package config

import (
	"strings"
	"time"

	"github.com/spf13/viper"
)

const (
	// DefaultHTTPAddr 是 HTTP 服务默认监听地址。
	DefaultHTTPAddr = ":8080"
	// EnvPrefix 是环境变量前缀，如 LEXI_HTTP_ADDR。
	EnvPrefix = "LEXI"
)

// Config 是服务端配置。
type Config struct {
	HTTP     HTTPConfig
	Database DatabaseConfig
}

// HTTPConfig 是 HTTP 服务相关配置。
type HTTPConfig struct {
	// Addr 是 HTTP 监听地址，如 ":8080"。
	Addr string
}

// DatabaseConfig 是 PostgreSQL 连接与连接池配置。连接池参数 0 值表示
// 采用 pgx 默认（internal/infra/database.Options），此处不做二次默认。
type DatabaseConfig struct {
	// URL 是 PostgreSQL 连接串，如
	// postgres://user:pass@localhost:5432/lexi_loop?sslmode=disable。
	// 必填；未提供时为空字符串，由调用方（组合根）决定如何处理。
	URL string

	// MaxConns / MinConns 是连接池上下限；0 值表示 pgx 默认。
	MaxConns int
	MinConns int

	// MaxConnLifetime / MaxConnIdleTime 是连接的复用与空闲回收时长；
	// 0 值表示 pgx 默认。
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
}

// Load 从默认值与 LEXI_* 环境变量加载配置。
func Load() (*Config, error) {
	v := viper.New()
	v.SetEnvPrefix(EnvPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	v.SetDefault("http.addr", DefaultHTTPAddr)

	// 数据库配置键：database.url 对应环境变量 LEXI_DATABASE_URL；
	// 连接池参数缺省为 0，语义为采用 pgx 默认值（见 DatabaseConfig 注释）。
	return &Config{
		HTTP: HTTPConfig{
			Addr: v.GetString("http.addr"),
		},
		Database: DatabaseConfig{
			URL:             v.GetString("database.url"),
			MaxConns:        v.GetInt("database.max_conns"),
			MinConns:        v.GetInt("database.min_conns"),
			MaxConnLifetime: v.GetDuration("database.max_conn_lifetime"),
			MaxConnIdleTime: v.GetDuration("database.max_conn_idle_time"),
		},
	}, nil
}
