package ecdict

// ECDICT 导入的 testcontainers 集成测试（Go 测试规范 §2：离线导入使用
// 小型固定 CSV 与真实 PostgreSQL 验证幂等重跑、批次回滚和字段映射）。

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	"github.com/dongwlin/lexi-loop/apps/server/internal/domain"
	"github.com/dongwlin/lexi-loop/apps/server/internal/repo"
)

// fixtureCSV 是小型固定 ECDICT 标准表头 CSV：覆盖完整字段映射、词形
// 词条（lemma 指向原形）、词频全零行与空词行；释义字段含带引号内换行
// 的多词性翻译。
const fixtureCSV = `word,phonetic,definition,translation,pos,collins,oxford,tag,bnc,frq,exchange,detail,audio
derive,/dɪˈraɪv/,obtain something from,"v. 获得；推导出
v. 起源于",n:0/0,3,1,cet4 cet6 ky toefl,12000,3500,0:derive/d:derived/p:derived/i:deriving/3:derives,,
derived,/dɪˈraɪv/,past simple of derive,v. 获得；推导出,n:0/0,0,0,,0,0,0:derive/p:derived,,
,空词行不导入,,,,,,,,,,
ambiguous,/æmˈbɪɡjuəs/,open to multiple interpretations,"adj. 模棱两可的；含糊不清的
adj. 有歧义的",n:0/0,0,0,zk gk cet4 cet6 ky,8041,2669,0:ambiguous,,http://audio.example/ambiguous.mp3
`

func TestImport_MapsFieldsAndCounts(t *testing.T) {
	db := requireTestDB(t)
	truncateDictionary(t)
	ctx := context.Background()

	result, err := NewImporter(db, 2).Import(ctx, strings.NewReader(fixtureCSV), "test-v1", nil)
	require.NoError(t, err)
	assert.Equal(t, 4, result.RowsProcessed, "数据行总数含被跳过的空词行")
	assert.Equal(t, 1, result.RowsSkipped)
	assert.Equal(t, 3, result.EntriesWritten)
	assert.Equal(t, 2, result.BatchesCommitted, "满批与收尾批次各提交一次")
	assert.Equal(t, 3, countDictionaryEntries(t))

	entryRepo := repo.NewDictionaryRepo(db)

	derive, err := entryRepo.FindByHeadword(ctx, "derive")
	require.NoError(t, err)
	assert.Equal(t, "derive", derive.Lemma)
	assert.Equal(t, "/dɪˈraɪv/", derive.PhoneticUK)
	assert.Equal(t, "", derive.PhoneticUS)
	assert.Equal(t, []domain.Meaning{
		{Pos: "verb", Translations: []string{"获得", "推导出", "起源于"}},
	}, derive.RawMeanings)
	assert.Equal(t, map[string]string{
		"0": "derive", "d": "derived", "p": "derived", "i": "deriving", "3": "derives",
	}, derive.Exchange)
	assert.Equal(t, map[string]int{
		freqKeyBNC: 12000, freqKeyCOCA: 3500, freqKeyCollins: 3, freqKeyOxford: 1,
	}, derive.Frequency)
	assert.Equal(t, []string{"cet4", "cet6", "ky", "toefl"}, derive.Tags)
	assert.Equal(t, SourceName, derive.Source)
	assert.Equal(t, "test-v1", derive.SourceVersion, "数据版本写入 source_version")
	assert.False(t, derive.CreatedAt.IsZero())
	assert.Equal(t, time.UTC, derive.CreatedAt.Location(), "时间戳以 UTC 记录")

	derived, err := entryRepo.FindByHeadword(ctx, "derived")
	require.NoError(t, err)
	assert.Equal(t, "derive", derived.Lemma, "词形词条的 lemma 取 exchange 原形键")
	assert.Empty(t, derived.Frequency, "词频全零视为无数据")

	ambiguous, err := entryRepo.FindByHeadword(ctx, "ambiguous")
	require.NoError(t, err)
	assert.Equal(t, "ambiguous", ambiguous.Lemma)
	assert.Equal(t, []domain.Meaning{
		{Pos: "adjective", Translations: []string{"模棱两可的", "含糊不清的", "有歧义的"}},
	}, ambiguous.RawMeanings, "同词性两行合并，词性归一为英文全称")
	assert.Equal(t, map[string]int{freqKeyBNC: 8041, freqKeyCOCA: 2669}, ambiguous.Frequency)
}

func TestImport_IsIdempotentOnRerun(t *testing.T) {
	db := requireTestDB(t)
	truncateDictionary(t)
	ctx := context.Background()

	csvData := "word,phonetic,translation,exchange\n" +
		"derive,/dɪˈraɪv/,v. 获得；推导出,0:derive\n" +
		"run,,n. 跑,0:run\n"

	// 预置一个带词典层复习释义与人工来源的词条：导入应刷新 ECDICT
	// 字段并接管来源标记，但不得清掉 review_meanings（UpsertBatch 的
	// COALESCE 语义）。
	existing, err := domain.NewDictionaryEntry("derive", "", time.Now())
	require.NoError(t, err)
	existing.ReviewMeanings = []domain.Meaning{{Pos: "verb", Translations: []string{"获得"}}}
	existing.Source = "manual"
	require.NoError(t, repo.NewDictionaryRepo(db).Insert(ctx, existing))

	importer := NewImporter(db, 10)
	result, err := importer.Import(ctx, strings.NewReader(csvData), "", nil)
	require.NoError(t, err)
	assert.Equal(t, 2, result.EntriesWritten, "与既有词条冲突的行按 upsert 计入")

	// 重跑同一文件：不产生重复行，词条数不变。
	rerun, err := importer.Import(ctx, strings.NewReader(csvData), "", nil)
	require.NoError(t, err)
	assert.Equal(t, 2, rerun.EntriesWritten)
	assert.Equal(t, 2, countDictionaryEntries(t))

	derive, err := repo.NewDictionaryRepo(db).FindByHeadword(ctx, "derive")
	require.NoError(t, err)
	assert.Equal(t, "/dɪˈraɪv/", derive.PhoneticUK, "重跑刷新词典字段")
	assert.Equal(t, SourceName, derive.Source, "来源标记切换为本次导入")
	assert.Equal(t, "", derive.SourceVersion, "手动导入未指定版本时空串")
	assert.Equal(t, []domain.Meaning{{Pos: "verb", Translations: []string{"获得"}}},
		derive.ReviewMeanings, "review_meanings 不被导入清空")

	run, err := repo.NewDictionaryRepo(db).FindByHeadword(ctx, "run")
	require.NoError(t, err)
	assert.Equal(t, "run", run.Lemma)
}

func TestImport_RollsBackFailedBatchAndResumes(t *testing.T) {
	db := requireTestDB(t)
	truncateDictionary(t)
	ctx := context.Background()

	csvData := "word,translation,exchange\n" +
		"alpha,,0:alpha\n" +
		"beta,,0:beta\n" +
		"failword,,0:failword\n" +
		"gamma,,0:gamma\n"

	// 注入失败：headword = failword 的写入在测试库触发器中报错，其所在
	// 批次（failword + gamma，batch size 2）整体回滚（失败注入使用测试库
	// 临时触发器，不触碰 migrations，与 service 集成测试同一方式）。
	setupFailwordTrigger(t, db)

	result, err := NewImporter(db, 2).Import(ctx, strings.NewReader(csvData), "", nil)
	require.Error(t, err, "批次失败应中止导入")
	assert.Equal(t, 1, result.BatchesCommitted, "仅失败前的批次保持已提交")
	assert.Equal(t, 2, result.EntriesWritten)
	assert.Equal(t, 4, result.RowsProcessed)

	entryRepo := repo.NewDictionaryRepo(db)
	for _, word := range []string{"alpha", "beta"} {
		_, err := entryRepo.FindByHeadword(ctx, word)
		require.NoError(t, err, "已提交批次的词条应保留: "+word)
	}
	for _, word := range []string{"failword", "gamma"} {
		_, err := entryRepo.FindByHeadword(ctx, word)
		require.ErrorIs(t, err, repo.ErrNotFound, "失败批次应整体回滚: "+word)
	}

	// 移除失败注入后重跑同一文件：UpsertBatch 幂等续传，四行全部就位。
	dropFailwordTrigger(t, db)
	rerun, err := NewImporter(db, 2).Import(ctx, strings.NewReader(csvData), "", nil)
	require.NoError(t, err)
	assert.Equal(t, 4, rerun.EntriesWritten)
	assert.Equal(t, 4, countDictionaryEntries(t))
}

func TestImport_SameBatchDuplicateHeadword(t *testing.T) {
	db := requireTestDB(t)
	truncateDictionary(t)
	ctx := context.Background()

	// 同一批内出现重复 headword：就地覆盖保留后出现的行，语句不因
	// 「ON CONFLICT 同一行作用两次」而失败。
	csvData := "word,phonetic,translation,exchange\n" +
		"derive,/wrong/,v. 错误音标,0:derive\n" +
		"derive,/dɪˈraɪv/,v. 获得,0:derive\n"

	result, err := NewImporter(db, 10).Import(ctx, strings.NewReader(csvData), "", nil)
	require.NoError(t, err)
	assert.Equal(t, 1, result.EntriesWritten, "同批重复词只写入一行")
	assert.Equal(t, 1, countDictionaryEntries(t))

	derive, err := repo.NewDictionaryRepo(db).FindByHeadword(ctx, "derive")
	require.NoError(t, err)
	assert.Equal(t, "/dɪˈraɪv/", derive.PhoneticUK, "重复词保留后出现的行")
}

// setupFailwordTrigger 注入使 headword = 'failword' 写入报错的
// BEFORE INSERT 触发器，测试结束自动清理。
func setupFailwordTrigger(t *testing.T, db *bun.DB) {
	t.Helper()
	ctx := context.Background()
	_, err := db.ExecContext(ctx, `
		CREATE FUNCTION ecdict_test_reject_failword() RETURNS trigger AS $$
		BEGIN
			IF NEW.headword = 'failword' THEN
				RAISE EXCEPTION 'ecdict test: injected failure for %', NEW.headword;
			END IF;
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql`)
	require.NoError(t, err, "创建失败注入函数")
	_, err = db.ExecContext(ctx, `CREATE TRIGGER ecdict_test_reject_failword
		BEFORE INSERT ON dictionary_entries
		FOR EACH ROW EXECUTE FUNCTION ecdict_test_reject_failword()`)
	require.NoError(t, err, "创建失败注入触发器")
	t.Cleanup(func() {
		_, _ = db.ExecContext(ctx, "DROP TRIGGER IF EXISTS ecdict_test_reject_failword ON dictionary_entries")
		_, _ = db.ExecContext(ctx, "DROP FUNCTION IF EXISTS ecdict_test_reject_failword()")
	})
}

// dropFailwordTrigger 提前移除失败注入，用于验证移除后重跑可续传。
func dropFailwordTrigger(t *testing.T, db *bun.DB) {
	t.Helper()
	ctx := context.Background()
	_, err := db.ExecContext(ctx, "DROP TRIGGER IF EXISTS ecdict_test_reject_failword ON dictionary_entries")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, "DROP FUNCTION IF EXISTS ecdict_test_reject_failword()")
	require.NoError(t, err)
}
