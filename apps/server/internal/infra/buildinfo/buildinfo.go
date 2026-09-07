// Package buildinfo 汇集二进制的构建期信息：版本号与构建时间由 go build
// -ldflags "-X" 注入（Dockerfile server-builder 阶段），未注入即本地开发
// 构建，版本以 "dev" 占位、构建时间为空。CLI（cmd/version）与 HTTP API
// （handler/v1）都从本包读取，版本信息不在其它任何位置复制或硬编码。
package buildinfo

import "runtime"

// Version 是应用版本号；未注入时为开发构建占位 "dev"。
var Version = "dev"

// BuildTime 是构建时间（RFC3339 UTC）；未注入时为空串，表示未知。
var BuildTime = ""

// GoVersion 返回编译本二进制的 Go 工具链版本（形如 go1.26.7）。
func GoVersion() string {
	return runtime.Version()
}
