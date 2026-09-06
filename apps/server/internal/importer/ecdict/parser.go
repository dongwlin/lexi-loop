// Package ecdict 是 ECDICT CSV 的离线导入适配器与导入用例编排
// （docs/backend/structure.md §8）。解析器属于离线输入适配器，不放进
// Domain 或 Service；运行时读取的只是导入后的本地词典库
// （docs/dictionary/enrichment.md §1）。
package ecdict

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/dongwlin/lexi-loop/apps/server/internal/domain"
)

// sourceName 写入 dictionary_entries.source，标记词条数据来自 ECDICT
// 离线导入；在线 Provider 的来源标记（V2）取不同的值。
const sourceName = "ecdict"

// frequency 列到 JSONB 键的映射：bnc / frq 分别是 BNC 与 COCA 语料库
// 词频排名（docs/dictionary/data-model.md §8），collins 是柯林斯词频星级，
// oxford 是牛津三千核心词标记，同为客观语料元数据，随词频一并保留。
const (
	freqKeyBNC     = "bnc_frequency"
	freqKeyCOCA    = "coca_frequency"
	freqKeyCollins = "collins"
	freqKeyOxford  = "oxford"
)

// canonicalColumns 把 ECDICT CSV 表头（大小写不敏感，含常见别名）映射为
// 规范列名；未列出的表头不导入——definition（英文释义）与 audio 属 V2
// Enrich 链路（enrichment.md §4），pos 是词性占比统计而非义项词性，
// detail 为展示用杂项数据。
var canonicalColumns = map[string]string{
	"word":        "word",
	"phonetic":    "phonetic_uk", // ECDICT 的单音标列为英式读音
	"phonetic_uk": "phonetic_uk",
	"ukphone":     "phonetic_uk",
	"phonetic_us": "phonetic_us",
	"usphone":     "phonetic_us",
	"translation": "translation",
	"exchange":    "exchange",
	"tag":         "tag",
	"bnc":         "bnc",
	"frq":         "frq",
	"collins":     "collins",
	"oxford":      "oxford",
}

// RowReader 流式读取 ECDICT CSV：按表头把源字段逐行映射为
// domain.DictionaryEntry，数十万行的大型文件不整体载入内存。
type RowReader struct {
	csvReader   *csv.Reader
	columns     map[string]int
	rowsRead    int // 已读取的数据行数（含被跳过的行，不含表头）
	rowsSkipped int // word 为空等不可导入而跳过的行数
}

// NewRowReader 创建读取器并解析表头；缺少 word 列时报错。
func NewRowReader(r io.Reader) (*RowReader, error) {
	csvReader := csv.NewReader(r)
	// 列数不强制一致：按表头映射取用，缺失列 / 越界一律视为空串，
	// 容忍 ECDICT 衍生 CSV 的尾部列差异。
	csvReader.FieldsPerRecord = -1
	csvReader.TrimLeadingSpace = true

	header, err := csvReader.Read()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, errors.New("ecdict: csv is empty: missing header")
		}
		return nil, fmt.Errorf("ecdict: read csv header: %w", err)
	}
	columns, err := parseHeader(header)
	if err != nil {
		return nil, err
	}
	return &RowReader{csvReader: csvReader, columns: columns}, nil
}

// parseHeader 解析表头为「规范列名 → 列下标」映射；同义列名以先出现者
// 为准。
func parseHeader(header []string) (map[string]int, error) {
	columns := make(map[string]int, len(header))
	for i, name := range header {
		name = strings.TrimPrefix(name, "\ufeff") // 容忍带 BOM 的文件头
		name = strings.ToLower(strings.TrimSpace(name))
		canonical, ok := canonicalColumns[name]
		if !ok {
			continue
		}
		if _, exists := columns[canonical]; !exists {
			columns[canonical] = i
		}
	}
	if _, ok := columns["word"]; !ok {
		return nil, fmt.Errorf("ecdict: csv header has no %q column", "word")
	}
	return columns, nil
}

// Next 返回下一行映射出的词条；文件读完返回 io.EOF。word 为空的行不计
// 入词条、计入 RowsSkipped 并继续读取。
func (rr *RowReader) Next() (*domain.DictionaryEntry, error) {
	for {
		fields, err := rr.csvReader.Read()
		if errors.Is(err, io.EOF) {
			return nil, io.EOF
		}
		if err != nil {
			return nil, fmt.Errorf("ecdict: read csv row %d: %w", rr.rowsRead+1, err)
		}
		rr.rowsRead++
		entry, err := mapRow(fields, rr.columns, time.Now())
		if err != nil {
			return nil, fmt.Errorf("ecdict: map csv row %d: %w", rr.rowsRead, err)
		}
		if entry == nil {
			rr.rowsSkipped++
			continue
		}
		return entry, nil
	}
}

// RowsRead 返回已读取的数据行数（含跳过行，不含表头）。
func (rr *RowReader) RowsRead() int { return rr.rowsRead }

// RowsSkipped 返回已跳过的无效行数。
func (rr *RowReader) RowsSkipped() int { return rr.rowsSkipped }

// mapRow 把一行源字段映射为 domain.DictionaryEntry；word 为空时返回
// (nil, nil) 表示跳过。lemma 取词形变化表的原形键（exchange "0"），
// 缺失时由 domain.NewDictionaryEntry 回退为 headword。
func mapRow(fields []string, columns map[string]int, now time.Time) (*domain.DictionaryEntry, error) {
	headword := field(fields, columns, "word")
	if headword == "" {
		return nil, nil
	}
	exchange := parseExchange(field(fields, columns, "exchange"))
	entry, err := domain.NewDictionaryEntry(headword, exchange["0"], now)
	if err != nil {
		return nil, err
	}
	entry.PhoneticUK = field(fields, columns, "phonetic_uk")
	entry.PhoneticUS = field(fields, columns, "phonetic_us")
	entry.RawMeanings = parseTranslation(field(fields, columns, "translation"))
	entry.Exchange = exchange
	entry.Frequency = parseFrequency(map[string]string{
		freqKeyBNC:     field(fields, columns, "bnc"),
		freqKeyCOCA:    field(fields, columns, "frq"),
		freqKeyCollins: field(fields, columns, "collins"),
		freqKeyOxford:  field(fields, columns, "oxford"),
	})
	entry.Tags = parseTags(field(fields, columns, "tag"))
	entry.Source = sourceName
	return entry, nil
}

// field 取规范列名对应的源字段；列缺失或下标越界视为空串。
func field(fields []string, columns map[string]int, canonical string) string {
	i, ok := columns[canonical]
	if !ok || i >= len(fields) {
		return ""
	}
	return strings.TrimSpace(fields[i])
}

// translationPosPattern 匹配行首「词性.」前缀（如 n. / adj. / vt.）。
var translationPosPattern = regexp.MustCompile(`^([a-zA-Z]+)\.\s*(.*)$`)

// posNames 把 ECDICT 的词性缩写归一为英文全称（与 data-model.md §4 示例
// 的 "adjective" 一致，便于将来与在线来源的词性对齐合并）；未收录的
// 缩写保留原文小写。
var posNames = map[string]string{
	"n":      "noun",
	"adj":    "adjective",
	"adv":    "adverb",
	"v":      "verb",
	"vt":     "transitive verb",
	"vi":     "intransitive verb",
	"prep":   "preposition",
	"conj":   "conjunction",
	"pron":   "pronoun",
	"num":    "numeral",
	"art":    "article",
	"aux":    "auxiliary verb",
	"int":    "interjection",
	"interj": "interjection",
	"abbr":   "abbreviation",
	"pl":     "plural",
}

// parseTranslation 把 translation 列解析为结构化义项
// （docs/dictionary/data-model.md §4）：按行拆分，行首「词性.」前缀归一
// 后作为 pos，其余按半角 / 全角分号拆为 translations；同一词性的连续行
// 合并为一个义项组，无词性前缀的行整体作为一个义项（pos 为空）。
func parseTranslation(s string) []domain.Meaning {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	meanings := make([]domain.Meaning, 0, 4)
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		pos, rest := "", line
		if m := translationPosPattern.FindStringSubmatch(line); m != nil {
			pos = strings.ToLower(m[1])
			rest = m[2]
		}
		translations := splitSenses(rest)
		if len(translations) == 0 {
			continue
		}
		pos = normalizePos(pos)
		if n := len(meanings); n > 0 && meanings[n-1].Pos == pos {
			meanings[n-1].Translations = append(meanings[n-1].Translations, translations...)
			continue
		}
		meanings = append(meanings, domain.Meaning{Pos: pos, Translations: translations})
	}
	if len(meanings) == 0 {
		return nil
	}
	return meanings
}

// splitSenses 按半角 / 全角分号把一段释义拆为义项列表，去掉空项；义项
// 内的逗号等标点原样保留。
func splitSenses(s string) []string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == ';' || r == '；'
	})
	senses := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			senses = append(senses, part)
		}
	}
	return senses
}

func normalizePos(abbr string) string {
	if full, ok := posNames[abbr]; ok {
		return full
	}
	return abbr
}

// parseExchange 解析 exchange 词形变化列：形如
// "0:derive/d:derived/p:derived/i:deriving/3:derives"，条目以 "/" 分隔、
// 「键:值」成对；值本身可含多个逗号分隔的词形，原样保留。键 "0" 为
// 原形，是运行时 lemma 解析的依据（normalization.md §2）。畸形片段
// （缺少冒号、键或值为空）忽略。
func parseExchange(s string) map[string]string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	out := make(map[string]string, 8)
	for _, item := range strings.Split(s, "/") {
		key, value, ok := strings.Cut(item, ":")
		if !ok {
			continue
		}
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		out[key] = value
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// parseFrequency 解析词频类列：ECDICT 以 0 与空串表示无数据，两者与
// 非数字一样不写入结果，保持键「有值即有数据」的语义。
func parseFrequency(values map[string]string) map[string]int {
	out := make(map[string]int, len(values))
	for key, raw := range values {
		n, err := strconv.Atoi(strings.TrimSpace(raw))
		if err != nil || n == 0 {
			continue
		}
		out[key] = n
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// parseTags 解析空格分隔的考试 / 语料标签列。
func parseTags(s string) []string {
	tags := strings.Fields(s)
	if len(tags) == 0 {
		return nil
	}
	return tags
}
