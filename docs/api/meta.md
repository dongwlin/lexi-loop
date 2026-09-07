# Meta API（版本信息）

> 应用版本与构建信息的查询接口：供前端「关于」页展示与运维核对部署版本。
> 所有接口遵循 [HTTP API 设计规范](../specs/backend/HTTP%20API%20设计规范.md) 的标准响应结构。

## 1. 接口总览

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| GET | `/api/v1/version` | 查询应用版本号与构建信息 |

无入参、恒成功：契约只文档化 200，不声明 400 / 404 / 422。

## 2. 版本信息

```text
GET /api/v1/version
```

成功响应：

```json
{
  "code": "OK",
  "message": "success",
  "data": {
    "version": "1.2.3",
    "buildTime": "2026-09-07T12:00:00Z",
    "goVersion": "go1.26.7"
  }
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `version` | string | 应用版本号 |
| `buildTime` | string | 构建时间（RFC3339 UTC）；未注入时为空串 |
| `goVersion` | string | 编译服务二进制的 Go 工具链版本（`runtime.Version()`） |

### 契约：开发构建以 dev 占位

版本信息的唯一来源是 server 的 `internal/infra/buildinfo`（单一真相源）：`Version` / `BuildTime` 由 `go build -ldflags "-X"` 在构建期注入，`goVersion` 运行时读取。

- 发布构建：Dockerfile 经 `ARG VERSION` 注入版本号（如 `git describe --tags --always --dirty` 的产物，`docker-compose.yml` 透传宿主环境变量 `VERSION`），`BuildTime` 自动取镜像构建时刻。
- 开发构建（未注入）：`version` 为占位 `"dev"`，`buildTime` 为空串；前端「关于」页对空 `buildTime` 兜底展示「—」，页面不在构建期重复注入版本。
- CLI 对照：`lexi-loop version` 只输出版本号；`lexi-loop version -b` 输出 `version` / `build_time` / `go_version` 三行（`buildTime` 为空时显示 `-`）。
