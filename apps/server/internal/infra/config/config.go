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

// DefaultCORSAllowedOrigins 是 CORS Origin 白名单缺省值：本地前端开发源
// （Vite 默认端口 5173）。生产部署必须通过 LEXI_HTTP_CORS_ALLOWED_ORIGINS
// 显式配置真实前端 Origin。
var DefaultCORSAllowedOrigins = []string{
	"http://localhost:5173",
	"http://127.0.0.1:5173",
}

// Config 是服务端配置。
type Config struct {
	HTTP     HTTPConfig
	Database DatabaseConfig
}

// HTTPConfig 是 HTTP 服务相关配置。
type HTTPConfig struct {
	// Addr 是 HTTP 监听地址，如 ":8080"。
	Addr string

	// CORSAllowedOrigins 是允许跨源访问 API 的前端 Origin 白名单
	//（中间件层只做显式匹配，不支持通配符）。未配置时为本地开发源；
	// 同源请求不受 CORS 影响。
	CORSAllowedOrigins []string
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

// parseCSVList 把逗号分隔的环境变量值拆为列表：逐项去首尾空白并丢弃空项。
func parseCSVList(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// Load 从默认值与 LEXI_* 环境变量加载配置。
func Load() (*Config, error) {
	v := viper.New()
	v.SetEnvPrefix(EnvPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	v.SetDefault("http.addr", DefaultHTTPAddr)
	// 默认值以逗号串形式给出：环境变量覆盖后经 parseCSVList 拆分，保证
	// 「未配置」与「显式配置多个源」走同一条解析路径。
	v.SetDefault("http.cors.allowed_origins", strings.Join(DefaultCORSAllowedOrigins, ","))

	// 数据库配置键：database.url 对应环境变量 LEXI_DATABASE_URL；
	// 连接池参数缺省为 0，语义为采用 pgx 默认值（见 DatabaseConfig 注释）。
	return &Config{
		HTTP: HTTPConfig{
			Addr:               v.GetString("http.addr"),
			CORSAllowedOrigins: parseCSVList(v.GetString("http.cors.allowed_origins")),
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
