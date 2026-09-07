package dto

// VersionInfo 是 GET /api/v1/version 的响应数据（docs/api/meta.md §2）。
type VersionInfo struct {
	// Version 是应用版本号；开发构建为占位 "dev"。
	Version string `json:"version"`
	// BuildTime 是构建时间（RFC3339 UTC）；空串表示未注入（本地开发构建）。
	BuildTime string `json:"buildTime"`
	// GoVersion 是编译本服务所用的 Go 工具链版本。
	GoVersion string `json:"goVersion"`
}
