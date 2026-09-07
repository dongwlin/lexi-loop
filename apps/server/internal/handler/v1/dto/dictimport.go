package dto

// DictImportStatus 是 GET /api/v1/dictionary-import 的响应数据
// （docs/api/meta.md §3）：词典自动导入的进程内进度快照。时间与错误
// 为可空字段，前端据此收敛轮询（completed / idle 停止轮询）。
type DictImportStatus struct {
	// State 是导入状态机取值：idle | checking | importing | completed | failed。
	State string `json:"state"`
	// SourceVersion 是当前守卫 / 导入的数据集版本（manifest 版本）。
	SourceVersion string `json:"sourceVersion"`
	// RowsProcessed 是已读取的数据行数（含跳过行，不含表头）。
	RowsProcessed int64 `json:"rowsProcessed"`
	// RowsTotal 是 manifest 期望行数；idle 时为 0。
	RowsTotal int64 `json:"rowsTotal"`
	// EntriesWritten 是已写入 / 刷新的词条数。
	EntriesWritten int64 `json:"entriesWritten"`
	// StartedAt 是本次导入任务开始时间（RFC3339 UTC）；null 表示尚未开始。
	StartedAt *string `json:"startedAt"`
	// UpdatedAt 是进度最近更新时间（RFC3339 UTC）。
	UpdatedAt *string `json:"updatedAt"`
	// Error 是失败信息；仅 failed 非空，其余状态为 null。
	Error *string `json:"error"`
}
