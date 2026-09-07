# Meta API（版本信息与词典导入进度）

> 应用版本与构建信息的查询接口：供前端「关于」页展示与运维核对部署版本；词典自动导入的进度端点供前端右上角指示组件轮询。
> 所有接口遵循 [HTTP API 设计规范](../specs/backend/HTTP%20API%20设计规范.md) 的标准响应结构。

## 1. 接口总览

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| GET | `/api/v1/version` | 查询应用版本号与构建信息 |
| GET | `/api/v1/dictionary-import` | 查询词典自动导入进度（进程内存态） |

`/api/v1/version` 无入参、恒成功；`/api/v1/dictionary-import` 无入参、恒成功（进度快照即时可读）：契约只文档化 200，不声明 400 / 404 / 422。

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

## 3. 词典导入进度

```text
GET /api/v1/dictionary-import
```

serve 启动后进程内自动导入镜像内置词典（ECDICT），本端点返回导入编排的实时进度快照；无内置数据（`LEXI_DICT_CSV` 文件不存在或 `LEXI_DICT_AUTOCHECK=0`）时 `state` 为 `idle`。数据与版本由 `deploy/dict/fetch.sh` 固定（获取与校验见 [deploy/release.md](../deploy/release.md)）。

成功响应（`importing`）：

```json
{
  "code": "OK",
  "message": "success",
  "data": {
    "state": "importing",
    "sourceVersion": "82c9872",
    "rowsProcessed": 323400,
    "rowsTotal": 770611,
    "entriesWritten": 323400,
    "startedAt": "2026-09-07T12:00:00Z",
    "updatedAt": "2026-09-07T12:00:20Z",
    "error": null
  }
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `state` | string | 导入状态机取值：`idle \| checking \| importing \| completed \| failed` |
| `sourceVersion` | string | 当前守卫 / 导入的数据集版本（manifest 版本，pinned commit 短 SHA）；`idle` 时为空串 |
| `rowsProcessed` | number | 已读取的数据行数（含跳过行，不含表头） |
| `rowsTotal` | number | manifest 期望行数；`idle` 时为 0 |
| `entriesWritten` | number | 已写入 / 刷新的词条数 |
| `startedAt` | string \| null | 本次导入任务开始时间（RFC3339 UTC）；`null` 表示尚未开始 |
| `updatedAt` | string \| null | 进度最近更新时间（RFC3339 UTC） |
| `error` | string \| null | 失败信息；仅 `failed` 非空，其余状态为 `null` |

### 契约：状态机与轮询收敛

- 进度是**进程内存态**，不落库：进程重启后从 `checking` 重新做完整性守卫判断，不存在跨进程的持久进度。
- 自动检查开启时，服务构造即为 `checking`（含后台任务尚未调度、数据文件检查与守卫查询），避免首次请求误读 `idle` 后停止轮询；manifest 尚未读取时版本为空、总行数为 0、时间为 null。确认数据不可用后进入 `idle`；开关关闭时初始即为 `idle`。数据可用时继续 `checking` → `importing` → `completed` / `failed`，守卫完整也可直接从 `checking` 进入 `completed`。`checking` 与 `importing` 期间容器 `aria-busy` 语义由前端承载。
- **完整性守卫**：`dictionary_entries` 中 `source='ecdict'` 且 `source_version=<manifest 版本>` 的词条数达到 manifest 期望行数即视为完整，直接返回 `completed` 而不导入——同一守卫覆盖空库（全新部署）、半截库（导入中断后续传补齐）与旧版本库（数据集升级后全量刷新）。
- **失败语义**：导入失败不退出进程，`state=failed` 且 `error` 给出原因；服务降级可用（生词流程走最小词条），重启容器自动重试续传。
- **前端轮询建议**：`checking` / `importing` 3 秒轮询；`failed` 低频（如 15 秒）以展示重启后的续传恢复；进入 `completed` / `idle` 稳定态后停止轮询。重启容器产生的新一轮导入在页面不刷新的前提下不主动唤醒——属既定取舍，刷新页面即可恢复轮询。
- 进度只在批次事务提交后推进（崩溃安全）；优雅关闭时导入在批次边界停止，已提交批次保持有效。
