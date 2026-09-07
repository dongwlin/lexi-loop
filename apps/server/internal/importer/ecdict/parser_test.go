package ecdict

import (
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dongwlin/lexi-loop/apps/server/internal/domain"
)

// testNow 取非 UTC 时区的固定时间，验证映射结果按 UTC 记录时间戳。
var testNow = time.Date(2026, 9, 6, 12, 0, 0, 0, time.FixedZone("CST", 8*3600))

// standardColumns 是标准 ECDICT 表头解析出的列映射，供 mapRow 用例复用。
var standardColumns = func() map[string]int {
	cols, err := parseHeader(strings.Split(
		"word,phonetic,definition,translation,pos,collins,oxford,tag,bnc,frq,exchange,detail,audio", ","))
	if err != nil {
		panic(err)
	}
	return cols
}()

func TestParseHeader(t *testing.T) {
	t.Parallel()

	t.Run("标准表头按列名映射", func(t *testing.T) {
		t.Parallel()
		columns, err := parseHeader(strings.Split(
			"word,phonetic,definition,translation,pos,collins,oxford,tag,bnc,frq,exchange,detail,audio", ","))
		require.NoError(t, err)
		assert.Equal(t, map[string]int{
			"word": 0, "phonetic_uk": 1, "translation": 3, "collins": 5, "oxford": 6,
			"tag": 7, "bnc": 8, "frq": 9, "exchange": 10,
		}, columns)
	})

	t.Run("大小写空格与 BOM 容忍", func(t *testing.T) {
		t.Parallel()
		columns, err := parseHeader([]string{"\ufeff Word ", " PHONETIC ", "Translation"})
		require.NoError(t, err)
		assert.Equal(t, map[string]int{"word": 0, "phonetic_uk": 1, "translation": 2}, columns)
	})

	t.Run("常见别名列", func(t *testing.T) {
		t.Parallel()
		columns, err := parseHeader([]string{"word", "ukphone", "usphone"})
		require.NoError(t, err)
		assert.Equal(t, map[string]int{"word": 0, "phonetic_uk": 1, "phonetic_us": 2}, columns)
	})

	t.Run("同义列先出现者优先", func(t *testing.T) {
		t.Parallel()
		columns, err := parseHeader([]string{"word", "phonetic", "ukphone"})
		require.NoError(t, err)
		assert.Equal(t, 1, columns["phonetic_uk"])
	})

	t.Run("未知列忽略", func(t *testing.T) {
		t.Parallel()
		columns, err := parseHeader([]string{"word", "definition", "pos", "detail", "audio", "extra"})
		require.NoError(t, err)
		assert.Equal(t, map[string]int{"word": 0}, columns)
	})

	t.Run("缺少 word 列报错", func(t *testing.T) {
		t.Parallel()
		_, err := parseHeader([]string{"phonetic", "translation"})
		require.Error(t, err)
		assert.ErrorContains(t, err, `"word"`)
	})
}

func TestNewRowReader(t *testing.T) {
	t.Parallel()

	t.Run("空文件报错", func(t *testing.T) {
		t.Parallel()
		_, err := NewRowReader(strings.NewReader(""), "")
		require.Error(t, err)
	})

	t.Run("纯表头可创建且立即 EOF", func(t *testing.T) {
		t.Parallel()
		reader, err := NewRowReader(strings.NewReader("word,translation\n"), "")
		require.NoError(t, err)
		_, err = reader.Next()
		assert.ErrorIs(t, err, io.EOF)
		assert.Equal(t, 0, reader.RowsRead())
	})

	t.Run("缺少 word 列报错", func(t *testing.T) {
		t.Parallel()
		_, err := NewRowReader(strings.NewReader("phonetic,translation\n/aɪ/,,\n"), "")
		require.Error(t, err)
	})
}

func TestRowReader_Next(t *testing.T) {
	t.Parallel()

	csvData := "word,phonetic,translation,exchange\n" +
		"derive,/dɪˈraɪv/,v. 获得,0:derive\n" +
		"   ,,,,,\n" + // 空词行跳过
		"derived,,v. 获得,0:derive\n"

	t.Run("逐行产出并统计跳过行", func(t *testing.T) {
		t.Parallel()
		reader, err := NewRowReader(strings.NewReader(csvData), "test-v1")
		require.NoError(t, err)

		first, err := reader.Next()
		require.NoError(t, err)
		assert.Equal(t, "derive", first.Headword)
		assert.Equal(t, "test-v1", first.SourceVersion, "版本标记写入词条")

		second, err := reader.Next()
		require.NoError(t, err)
		assert.Equal(t, "derived", second.Headword)

		_, err = reader.Next()
		assert.ErrorIs(t, err, io.EOF)
		assert.Equal(t, 3, reader.RowsRead(), "空词行计入已读行数")
		assert.Equal(t, 1, reader.RowsSkipped(), "空词行计入跳过行数")
	})

	t.Run("行数不足按空串处理", func(t *testing.T) {
		t.Parallel()
		reader, err := NewRowReader(strings.NewReader("word,phonetic,translation\nshort\n"), "")
		require.NoError(t, err)
		entry, err := reader.Next()
		require.NoError(t, err)
		assert.Equal(t, "short", entry.Headword)
		assert.Empty(t, entry.RawMeanings)
	})
}

func TestMapRow(t *testing.T) {
	t.Parallel()

	t.Run("完整行映射到 Domain", func(t *testing.T) {
		t.Parallel()
		fields := []string{
			"derive", "/dɪˈraɪv/", "obtain from",
			"v. 获得；推导出\nv. 起源于", "n:0/0", "3", "1", "cet4 cet6 ky", "12000", "3500",
			"0:derive/d:derived/p:derived/i:deriving/3:derives", "{}", "",
		}
		entry, err := mapRow(fields, standardColumns, "", testNow)
		require.NoError(t, err)

		assert.Equal(t, "derive", entry.Headword)
		assert.Equal(t, "derive", entry.Lemma, "lemma 取 exchange 原形键")
		assert.Equal(t, "/dɪˈraɪv/", entry.PhoneticUK)
		assert.Equal(t, "", entry.PhoneticUS)
		assert.Equal(t, []domain.Meaning{
			{Pos: "verb", Translations: []string{"获得", "推导出", "起源于"}},
		}, entry.RawMeanings, "同词性两行合并为一个义项组")
		assert.Equal(t, map[string]string{
			"0": "derive", "d": "derived", "p": "derived", "i": "deriving", "3": "derives",
		}, entry.Exchange)
		assert.Equal(t, map[string]int{
			freqKeyBNC: 12000, freqKeyCOCA: 3500, freqKeyCollins: 3, freqKeyOxford: 1,
		}, entry.Frequency)
		assert.Equal(t, []string{"cet4", "cet6", "ky"}, entry.Tags)
		assert.Equal(t, SourceName, entry.Source)
		assert.Equal(t, "", entry.SourceVersion, "未指定版本时保持空串")
		assert.Equal(t, testNow.UTC(), entry.CreatedAt)
		assert.Equal(t, testNow.UTC(), entry.UpdatedAt)
	})

	t.Run("仅 word 的最小行", func(t *testing.T) {
		t.Parallel()
		entry, err := mapRow([]string{"SomeWord"}, standardColumns, "test-v1", testNow)
		require.NoError(t, err)
		assert.Equal(t, "SomeWord", entry.Headword)
		assert.Equal(t, "SomeWord", entry.Lemma, "无 exchange 原形键时回退为 headword")
		assert.Empty(t, entry.RawMeanings)
		assert.Nil(t, entry.Exchange)
		assert.Nil(t, entry.Frequency)
		assert.Nil(t, entry.Tags)
		assert.Equal(t, SourceName, entry.Source)
		assert.Equal(t, "test-v1", entry.SourceVersion, "版本标记写入词条")
	})

	t.Run("word 为空的行返回跳过", func(t *testing.T) {
		t.Parallel()
		for _, fields := range [][]string{
			{""},
			{"   ", "/aɪ/", "", "", "", "", "", "", "", "", "", "", ""},
		} {
			entry, err := mapRow(fields, standardColumns, "", testNow)
			require.NoError(t, err)
			assert.Nil(t, entry)
		}
	})

	t.Run("词形指向原形的词条保留原样 lemma", func(t *testing.T) {
		t.Parallel()
		fields := []string{
			"derived", "", "", "v. 获得", "", "", "", "", "0", "0",
			"0:derive/p:derived", "", "",
		}
		entry, err := mapRow(fields, standardColumns, "", testNow)
		require.NoError(t, err)
		assert.Equal(t, "derive", entry.Lemma)
		assert.Empty(t, entry.Frequency, "词频全为 0 时不写入任何键")
	})
}

func TestParseTranslation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  []domain.Meaning
	}{
		{
			name:  "多词性按行拆分",
			input: "n. 模糊；含糊\nadj. 模糊的;含糊的",
			want: []domain.Meaning{
				{Pos: "noun", Translations: []string{"模糊", "含糊"}},
				{Pos: "adjective", Translations: []string{"模糊的", "含糊的"}},
			},
		},
		{
			name:  "同词性连续行合并",
			input: "v. 获得；推导出\nv. 起源于",
			want: []domain.Meaning{
				{Pos: "verb", Translations: []string{"获得", "推导出", "起源于"}},
			},
		},
		{
			name:  "细分词性归一为全称",
			input: "vt. 及物用法\nvi. 不及物用法",
			want: []domain.Meaning{
				{Pos: "transitive verb", Translations: []string{"及物用法"}},
				{Pos: "intransitive verb", Translations: []string{"不及物用法"}},
			},
		},
		{
			name:  "未知词性缩写保留原文",
			input: "comb. 组合词",
			want:  []domain.Meaning{{Pos: "comb", Translations: []string{"组合词"}}},
		},
		{
			name:  "无词性前缀整行为一个义项",
			input: "模棱两可的",
			want:  []domain.Meaning{{Pos: "", Translations: []string{"模棱两可的"}}},
		},
		{
			name:  "义项内逗号原样保留",
			input: "n. 跑，奔跑",
			want:  []domain.Meaning{{Pos: "noun", Translations: []string{"跑，奔跑"}}},
		},
		{
			name:  "无释义的词性行忽略",
			input: "adj.\nn. 东西",
			want:  []domain.Meaning{{Pos: "noun", Translations: []string{"东西"}}},
		},
		{
			name:  "空白输入返回 nil",
			input: "  \n \n",
			want:  nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, parseTranslation(tt.input))
		})
	}
}

func TestParseExchange(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  map[string]string
	}{
		{
			name:  "标准词形变化表",
			input: "0:derive/d:derived/p:derived/i:deriving/3:derives",
			want: map[string]string{
				"0": "derive", "d": "derived", "p": "derived", "i": "deriving", "3": "derives",
			},
		},
		{
			name:  "值含逗号的多个词形原样保留",
			input: "s:phenomena,phenomenons/0:phenomenon",
			want:  map[string]string{"s": "phenomena,phenomenons", "0": "phenomenon"},
		},
		{
			name:  "缺少冒号或值为空的片段忽略",
			input: "0:derive/junk/:/x:",
			want:  map[string]string{"0": "derive"},
		},
		{
			name:  "空输入返回 nil",
			input: "  ",
			want:  nil,
		},
		{
			name:  "全是畸形片段返回 nil",
			input: "junk/:/x:",
			want:  nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, parseExchange(tt.input))
		})
	}
}

func TestParseFrequency(t *testing.T) {
	t.Parallel()

	t.Run("零值与非法值不写入", func(t *testing.T) {
		t.Parallel()
		got := parseFrequency(map[string]string{
			freqKeyBNC: "0", freqKeyCOCA: "", freqKeyCollins: "abc", freqKeyOxford: "1",
		})
		assert.Equal(t, map[string]int{freqKeyOxford: 1}, got)
	})

	t.Run("全部有效值", func(t *testing.T) {
		t.Parallel()
		got := parseFrequency(map[string]string{
			freqKeyBNC: "12000", freqKeyCOCA: "3500", freqKeyCollins: "3", freqKeyOxford: "1",
		})
		assert.Equal(t, map[string]int{
			freqKeyBNC: 12000, freqKeyCOCA: 3500, freqKeyCollins: 3, freqKeyOxford: 1,
		}, got)
	})

	t.Run("全部无效返回 nil", func(t *testing.T) {
		t.Parallel()
		assert.Nil(t, parseFrequency(map[string]string{freqKeyBNC: "0", freqKeyCOCA: " "}))
	})
}

func TestParseTags(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []string{"zk", "gk", "cet4", "cet6", "ky"}, parseTags(" zk   gk\tcet4\ncet6 ky "))
	assert.Nil(t, parseTags("   "))
}
