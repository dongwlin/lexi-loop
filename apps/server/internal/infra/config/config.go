// Package config 基于 viper 加载配置：默认值 + LEXI_* 环境变量覆盖。
package config

import (
	"strings"

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
	HTTP HTTPConfig
}

// HTTPConfig 是 HTTP 服务相关配置。
type HTTPConfig struct {
	// Addr 是 HTTP 监听地址，如 ":8080"。
	Addr string
}

// Load 从默认值与 LEXI_* 环境变量加载配置。
func Load() (*Config, error) {
	v := viper.New()
	v.SetEnvPrefix(EnvPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	v.SetDefault("http.addr", DefaultHTTPAddr)

	// TODO: 需要时补充配置来源与键：--config 配置文件、http 读写超时、
	// database.dsn / 连接池参数等（viper 已支持 flag / 文件 / env 三种来源）。

	return &Config{
		HTTP: HTTPConfig{
			Addr: v.GetString("http.addr"),
		},
	}, nil
}
