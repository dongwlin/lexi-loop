package service

import (
	"time"
)

// DictImport 提供词典导入进度的只读能力；启动与后台生命周期由组合根管理。
// Snapshot 只读取进程内快照，不执行 I/O，不需要请求上下文。
type DictImport interface {
	Snapshot() DictImportSnapshot
}

// DictImportState 是词典自动导入的进程内状态机取值（docs/api/meta.md §3）：
// checking → importing → completed / failed；确认跳过时为 idle，重启后回到
// checking 重新守卫判断。
type DictImportState string

const (
	DictImportStateIdle      DictImportState = "idle"
	DictImportStateChecking  DictImportState = "checking"
	DictImportStateImporting DictImportState = "importing"
	DictImportStateCompleted DictImportState = "completed"
	DictImportStateFailed    DictImportState = "failed"
)

// DictImportSnapshot 是导入进度的进程内快照；JSON 契约由版本化 DTO 转换
// （structure.md §4.3：Service 结果类型不带 json 标签）。
type DictImportSnapshot struct {
	State          DictImportState
	SourceVersion  string
	RowsProcessed  int64 // 已读取的数据行数（含跳过行，不含表头）
	RowsTotal      int64 // manifest 期望行数；idle 时为 0
	EntriesWritten int64
	StartedAt      time.Time // 零值表示尚未开始（idle）
	UpdatedAt      time.Time
	ErrorMessage   string // 仅 failed 非空
}
