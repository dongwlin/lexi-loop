package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/uptrace/bun"

	"github.com/dongwlin/lexi-loop/apps/server/internal/importer/ecdict"
	"github.com/dongwlin/lexi-loop/apps/server/internal/repo"
)

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

// manifestFile 是词典数据集 manifest 的文件名，固定与 CSV 同目录
// （deploy/dict/fetch.sh 的产物，见 docs/deploy/release.md）。
const manifestFile = "manifest.json"

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

// dictManifest 是 deploy/dict/manifest.json（fetch.sh 产物）的读取形态：
// 记录 pinned 数据集的版本与期望行数，是版本完整性守卫的判据。
// sha256 由 fetch.sh 在获取时校验，运行期不重复哈希。
type dictManifest struct {
	Version   string `json:"version"`
	Source    string `json:"source"`
	SHA256    string `json:"sha256"`
	RowsTotal int64  `json:"rowsTotal"`
}

// DictImport 编排 serve 启动后的词典自动导入（issue #3）：读取镜像内置
// CSV → 版本完整性守卫 → 不完整时经 ecdict.Importer 异步导入。进度是
// 进程内存态，不落库、不引入队列（structure.md §9）；由组合根组装并在
// serve 启动后调用 StartAutoImport，goroutine 生命周期随组合根的 ctx。
// 导入编排依赖 ecdict.Importer（structure.md §1 的 service → importer
// 组装边）：importer 是导入期数据源适配器（structure.md §8），本服务把
// 它编排为 serve 期后台用例。
type DictImport struct {
	db           *bun.DB
	importer     *ecdict.Importer
	csvPath      string
	manifestPath string
	autoCheck    bool
	log          zerolog.Logger

	mu      sync.Mutex
	started bool // 单飞：同进程至多启动一个导入任务
	snap    DictImportSnapshot
}

// NewDictImport 构造词典自动导入服务；manifest 固定与 CSV 同目录读取。
// 自动检查开启时初始即 checking，避免 HTTP 已可用但后台尚未调度时
// 返回 idle 导致客户端停止轮询；开关关闭时才直接为 idle。
func NewDictImport(db *bun.DB, importer *ecdict.Importer, csvPath string, autoCheck bool, log zerolog.Logger) *DictImport {
	state := DictImportStateIdle
	if autoCheck {
		state = DictImportStateChecking
	}
	return &DictImport{
		db:           db,
		importer:     importer,
		csvPath:      csvPath,
		manifestPath: filepath.Join(filepath.Dir(csvPath), manifestFile),
		autoCheck:    autoCheck,
		log:          log,
		snap:         DictImportSnapshot{State: state},
	}
}

// StartAutoImport 在 serve 启动后拉起导入编排 goroutine（单飞：同进程
// 至多启动一次）。ctx 取消（优雅关闭）后导入在批次边界停止，已提交
// 批次保持有效，下次启动重新守卫判断后续传。
func (s *DictImport) StartAutoImport(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		return
	}
	s.started = true
	go s.run(ctx)
}

// Snapshot 返回当前进度的副本，供 Meta 进度端点读取。
func (s *DictImport) Snapshot() DictImportSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.snap
}

// run 执行自动导入编排：开关与文件检查 → manifest → 守卫 → 导入。
// 任何失败不退出进程：置 failed 后返回，服务照常响应其它请求
// （issue #3：失败语义），下次重启自动重试续传。
func (s *DictImport) run(ctx context.Context) {
	if !s.autoCheck {
		s.log.Info().Msg("dict auto import disabled (LEXI_DICT_AUTOCHECK=0), skip")
		return
	}
	if _, err := os.Stat(s.csvPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// 非镜像运行（如 go run . serve）：确认缺文件后收敛为 idle。
			s.log.Info().Str("path", s.csvPath).Msg("built-in dict csv not found, skip auto import")
			s.skip()
			return
		}
		s.fail(fmt.Errorf("stat dict csv: %w", err))
		return
	}
	manifest, err := s.loadManifest()
	if err != nil {
		s.log.Warn().Err(err).Str("path", s.manifestPath).Msg("dict manifest unusable, skip auto import")
		s.skip()
		return
	}

	s.beginChecking(manifest)

	// ctx 已取消（启动即优雅关闭）：不进守卫查询，保持 checking 结束，
	// 重启后重新守卫判断。
	if err := ctx.Err(); err != nil {
		s.log.Info().Err(err).Msg("dict import interrupted by shutdown before guard")
		return
	}

	// 版本完整性守卫：source_version 达到 manifest 期望行数即视为完整，
	// 置 completed 不导入。同一守卫覆盖空库（全新部署）、半截库（导入
	// 中断续传补齐）与旧版本库（升级数据集后全量刷新）三种情况。
	count, err := repo.NewDictionaryRepo(s.db).CountBySource(ctx, ecdict.SourceName, manifest.Version)
	if err != nil {
		if ctx.Err() != nil {
			// 守卫查询期间关闭：与导入中断同一语义，不置 failed。
			s.log.Info().Err(err).Msg("dict import interrupted by shutdown")
			return
		}
		s.fail(fmt.Errorf("check dict completeness: %w", err))
		return
	}
	if int64(count) >= manifest.RowsTotal {
		s.log.Info().Str("version", manifest.Version).Int("entries", count).Msg("dict already complete, skip import")
		s.complete(manifest.RowsTotal, int64(count))
		return
	}

	s.beginImport()
	file, err := os.Open(s.csvPath)
	if err != nil {
		s.fail(fmt.Errorf("open dict csv: %w", err))
		return
	}
	defer func() {
		_ = file.Close()
	}()

	result, err := s.importer.Import(ctx, file, manifest.Version, func(p ecdict.Progress) {
		s.updateProgress(int64(p.RowsProcessed), int64(p.EntriesWritten))
	})
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			// 优雅关闭：批次边界停止，已提交数据保持有效，下次启动续传。
			// 进程即将退出，状态不再变更。
			s.log.Info().Err(err).Msg("dict import interrupted by shutdown")
			return
		}
		s.fail(err)
		return
	}
	s.log.Info().
		Int("rows", result.RowsProcessed).
		Int("skipped", result.RowsSkipped).
		Int("entries", result.EntriesWritten).
		Msg("dict import completed")
	s.complete(int64(result.RowsProcessed), int64(result.EntriesWritten))
}

// loadManifest 读取并校验 manifest：version 与 rowsTotal 是守卫判据，
// 缺失或非法时返回错误（调用方按「数据不可用」跳过导入）。
func (s *DictImport) loadManifest() (*dictManifest, error) {
	raw, err := os.ReadFile(s.manifestPath)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}
	var m dictManifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	if m.Version == "" || m.RowsTotal <= 0 {
		return nil, fmt.Errorf("manifest has no version or rowsTotal")
	}
	return &m, nil
}

// skip 确认无可用数据后进入稳定态，客户端可以停止轮询。
func (s *DictImport) skip() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.snap = DictImportSnapshot{State: DictImportStateIdle}
}

// beginChecking 在数据文件检查通过后设置版本与期望行数并盖 StartedAt。
func (s *DictImport) beginChecking(m *dictManifest) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	s.snap = DictImportSnapshot{
		State:         DictImportStateChecking,
		SourceVersion: m.Version,
		RowsTotal:     m.RowsTotal,
		StartedAt:     now,
		UpdatedAt:     now,
	}
}

// beginImport 进入导入态。
func (s *DictImport) beginImport() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.snap.State = DictImportStateImporting
	s.snap.UpdatedAt = time.Now().UTC()
}

// updateProgress 由 importer 的 onProgress 回调逐批推进（批次提交后
// 调用，崩溃安全，structure.md §5.3）。
func (s *DictImport) updateProgress(rowsProcessed, entriesWritten int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.snap.State != DictImportStateImporting {
		return
	}
	s.snap.RowsProcessed = rowsProcessed
	s.snap.EntriesWritten = entriesWritten
	s.snap.UpdatedAt = time.Now().UTC()
}

// complete 落入完成态：守卫路径下 processed 与 written 都是实际计数。
func (s *DictImport) complete(rowsProcessed, entriesWritten int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.snap.State = DictImportStateCompleted
	s.snap.RowsProcessed = rowsProcessed
	s.snap.EntriesWritten = entriesWritten
	s.snap.UpdatedAt = time.Now().UTC()
}

// fail 落入失败态：保留最后进度，记录错误信息（经 DTO 暴露给前端提示，
// 服务降级可用，重启自动续传）。
func (s *DictImport) fail(err error) {
	s.log.Error().Err(err).Msg("dict auto import failed")
	s.mu.Lock()
	defer s.mu.Unlock()
	s.snap.State = DictImportStateFailed
	s.snap.ErrorMessage = err.Error()
	s.snap.UpdatedAt = time.Now().UTC()
}
