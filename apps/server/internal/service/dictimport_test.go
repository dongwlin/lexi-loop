package service

// DictImport 自动导入编排的 testcontainers 集成测试：守卫完整跳过、
// 不完整导入、失败语义、开关与缺文件跳过、优雅关闭语义。异步 goroutine
// 经 Snapshot 轮询收敛断言（Go 测试规范：并发路径开 -race）。

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	"github.com/dongwlin/lexi-loop/apps/server/internal/domain"
	"github.com/dongwlin/lexi-loop/apps/server/internal/importer/ecdict"
	"github.com/dongwlin/lexi-loop/apps/server/internal/repo"
)

// fixtureCSV 是三词条的小型 ECDICT CSV（含词形 lemma 与不同音标）。
const dictImportFixtureCSV = `word,phonetic,translation,exchange
derive,/dɪˈraɪv/,v. 获得；推导出,0:derive
run,,n. 跑,0:run
derived,,v. 获得,0:derive
`

// writeDictFixture 在临时目录写入 CSV 与 manifest，返回 CSV 路径
// （manifest 与 CSV 固定同目录，与 fetch.sh / 镜像布局一致）。
func writeDictFixture(t *testing.T, csvData, version string, rowsTotal int64) string {
	t.Helper()
	dir := t.TempDir()
	csvPath := filepath.Join(dir, "ecdict.csv")
	manifestPath := filepath.Join(dir, "manifest.json")
	require.NoError(t, os.WriteFile(csvPath, []byte(csvData), 0o644))
	manifest := fmt.Sprintf(`{"version":%q,"source":"ecdict","rowsTotal":%d}`, version, rowsTotal)
	require.NoError(t, os.WriteFile(manifestPath, []byte(manifest), 0o644))
	return csvPath
}

// waitForState 轮询快照直到到达期望状态或超时，返回最终快照。
func waitForState(t *testing.T, svc *DictImport, want ...DictImportState) DictImportSnapshot {
	t.Helper()
	allowed := make(map[DictImportState]bool, len(want))
	for _, s := range want {
		allowed[s] = true
	}
	var snap DictImportSnapshot
	require.Eventually(t, func() bool {
		snap = svc.Snapshot()
		return allowed[snap.State]
	}, 10*time.Second, 20*time.Millisecond, "等待状态 %v，最后快照: %+v", want, snap)
	return snap
}

// seedSourceEntries 预置 n 个带指定来源与版本的词条（守卫的完备性
// 由 repo 层 CountBySource 测试覆盖，这里只需凑数量）。
func seedSourceEntries(t *testing.T, db *bun.DB, source, version string, n int) {
	t.Helper()
	entryRepo := repo.NewDictionaryRepo(db)
	for i := 0; i < n; i++ {
		entry, err := domain.NewDictionaryEntry(fmt.Sprintf("seedword%02d", i), "", time.Now())
		require.NoError(t, err)
		entry.Source = source
		entry.SourceVersion = version
		require.NoError(t, entryRepo.Insert(context.Background(), entry))
	}
}

func TestDictImport_CompleteGuardSkipsImport(t *testing.T) {
	db := requireTestDB(t)
	resetTables(t)
	ctx := context.Background()

	seedSourceEntries(t, db, ecdict.SourceName, "guard-v1", 2)
	csvPath := writeDictFixture(t, dictImportFixtureCSV, "guard-v1", 2)

	svc := NewDictImport(db, ecdict.NewImporter(db, 0), csvPath, true, zerolog.Nop())
	svc.StartAutoImport(ctx)
	// 单飞：第二次启动是无副作用的 no-op。
	svc.StartAutoImport(ctx)

	snap := waitForState(t, svc, DictImportStateCompleted)
	assert.Equal(t, "guard-v1", snap.SourceVersion)
	assert.Equal(t, int64(2), snap.RowsTotal)
	assert.Equal(t, int64(2), snap.RowsProcessed, "守卫路径按实际计数收敛")
	assert.Equal(t, int64(2), snap.EntriesWritten)
	assert.False(t, snap.StartedAt.IsZero())
	assert.False(t, snap.UpdatedAt.IsZero())
	assert.Empty(t, snap.ErrorMessage)

	// 守卫不导入：manifest 之外的同版本计数不变化，fixture 词条未写入。
	count, err := repo.NewDictionaryRepo(db).CountBySource(ctx, ecdict.SourceName, "guard-v1")
	require.NoError(t, err)
	assert.Equal(t, 2, count)
	_, err = repo.NewDictionaryRepo(db).FindByHeadword(ctx, "derive")
	assert.ErrorIs(t, err, repo.ErrNotFound, "守卫跳过导入，fixture 词条不出现")
}

func TestDictImport_ImportsIncompleteDictionary(t *testing.T) {
	db := requireTestDB(t)
	resetTables(t)
	ctx := context.Background()

	csvPath := writeDictFixture(t, dictImportFixtureCSV, "import-v1", 3)

	svc := NewDictImport(db, ecdict.NewImporter(db, 0), csvPath, true, zerolog.Nop())
	svc.StartAutoImport(ctx)

	snap := waitForState(t, svc, DictImportStateCompleted)
	assert.Equal(t, "import-v1", snap.SourceVersion)
	assert.Equal(t, int64(3), snap.RowsTotal)
	assert.Equal(t, int64(3), snap.RowsProcessed)
	assert.Equal(t, int64(3), snap.EntriesWritten)
	assert.False(t, snap.StartedAt.IsZero())
	assert.False(t, snap.UpdatedAt.IsZero())

	// 词条带版本落库，守卫计数可见。
	count, err := repo.NewDictionaryRepo(db).CountBySource(ctx, ecdict.SourceName, "import-v1")
	require.NoError(t, err)
	assert.Equal(t, 3, count)
	derive, err := repo.NewDictionaryRepo(db).FindByHeadword(ctx, "derive")
	require.NoError(t, err)
	assert.Equal(t, "import-v1", derive.SourceVersion)
}

func TestDictImport_FailsOnMalformedCSV(t *testing.T) {
	db := requireTestDB(t)
	resetTables(t)
	ctx := context.Background()

	// 注入 CSV 解析错误（非引号字段中的裸引号）：首批提交后命中坏行，
	// 导入中止 → failed，且错误信息经快照暴露。
	csvData := "word,translation\nderive,v. 获得\nbad\"quote,v. 损坏\n"
	csvPath := writeDictFixture(t, csvData, "fail-v1", 2)

	svc := NewDictImport(db, ecdict.NewImporter(db, 1), csvPath, true, zerolog.Nop())
	svc.StartAutoImport(ctx)

	snap := waitForState(t, svc, DictImportStateFailed)
	assert.Equal(t, "fail-v1", snap.SourceVersion)
	assert.Equal(t, int64(2), snap.RowsTotal)
	assert.Equal(t, int64(1), snap.EntriesWritten, "失败前的已提交批次保留")
	assert.False(t, snap.StartedAt.IsZero())
	assert.NotEmpty(t, snap.ErrorMessage, "失败信息经快照暴露")
}

func TestDictImport_DisabledOrMissingDataStaysIdle(t *testing.T) {
	db := requireTestDB(t)
	resetTables(t)
	ctx := context.Background()

	t.Run("开关关闭不产生任何动作", func(t *testing.T) {
		csvPath := writeDictFixture(t, dictImportFixtureCSV, "off-v1", 3)
		svc := NewDictImport(db, ecdict.NewImporter(db, 0), csvPath, false, zerolog.Nop())
		svc.StartAutoImport(ctx)
		// goroutine 可能尚未执行，短暂等待后确认状态从未离开 idle。
		time.Sleep(200 * time.Millisecond)
		snap := svc.Snapshot()
		assert.Equal(t, DictImportStateIdle, snap.State)
		assert.True(t, snap.StartedAt.IsZero())
	})

	t.Run("CSV 不存在视为非镜像运行静默跳过", func(t *testing.T) {
		svc := NewDictImport(db, ecdict.NewImporter(db, 0), filepath.Join(t.TempDir(), "missing.csv"), true, zerolog.Nop())
		svc.StartAutoImport(ctx)
		time.Sleep(200 * time.Millisecond)
		snap := svc.Snapshot()
		assert.Equal(t, DictImportStateIdle, snap.State)
		assert.True(t, snap.StartedAt.IsZero())
	})
}

func TestDictImport_PreCanceledContextStopsAsShutdown(t *testing.T) {
	db := requireTestDB(t)
	resetTables(t)

	// 预先取消的 ctx：守卫查询不再执行，按优雅关闭语义处理——状态停在
	// checking 且不落入 failed（进程即将退出，重启后重新守卫判断）。
	csvPath := writeDictFixture(t, dictImportFixtureCSV, "cancel-v1", 3)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	svc := NewDictImport(db, ecdict.NewImporter(db, 0), csvPath, true, zerolog.Nop())
	svc.StartAutoImport(ctx)

	snap := waitForState(t, svc, DictImportStateChecking)
	assert.Equal(t, "cancel-v1", snap.SourceVersion)
	assert.False(t, snap.StartedAt.IsZero())

	count, err := repo.NewDictionaryRepo(db).CountBySource(context.Background(), ecdict.SourceName, "cancel-v1")
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

// TestDictImport_MalformedManifestSkips：CSV 存在但 manifest 不可用时
// 按数据不可用跳过，不导入也不置失败。
func TestDictImport_MalformedManifestSkips(t *testing.T) {
	db := requireTestDB(t)
	resetTables(t)
	ctx := context.Background()

	dir := t.TempDir()
	csvPath := filepath.Join(dir, "ecdict.csv")
	require.NoError(t, os.WriteFile(csvPath, []byte(dictImportFixtureCSV), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "manifest.json"), []byte("{invalid"), 0o644))

	svc := NewDictImport(db, ecdict.NewImporter(db, 0), csvPath, true, zerolog.Nop())
	svc.StartAutoImport(ctx)
	time.Sleep(200 * time.Millisecond)
	snap := svc.Snapshot()
	assert.Equal(t, DictImportStateIdle, snap.State, "manifest 不可用时静默跳过")
	assert.True(t, snap.StartedAt.IsZero())
}
