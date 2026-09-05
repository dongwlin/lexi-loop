package app

// 本文件是组合根的组件组装入口（docs/backend/structure.md §3、§4.4）。
//
// MVP 骨架阶段业务组件尚未实现，serve 只启动 HTTP 服务与 /healthz。
// 后续在此按以下顺序显式组装长生命周期组件：
//
//   - infra/config：viper 配置（已就位，见 internal/infra/config）
//   - infra/database：*bun.DB 连接池（实现 migrate / 业务用例时接入）
//   - 日志（zerolog）等基础设施
//   - service.Word / service.Dictionary / service.Review（具体类型）
//   - handler/v1 版本化 Handler
//
// 组装完成后把上述依赖传入 handler.RegisterRoutes(...)，server.go 无需再
// 感知具体组件。Repo、Domain、Middleware 不作为长生命周期组件注入。
