package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInflectionLemma(t *testing.T) {
	for _, tt := range []struct{ name, text, lemma string }{
		{"过去式", "oversee的过去式", "oversee"},
		{"过去分词", "write 的过去分词", "write"},
		{"现在分词", "run的现在分词", "run"},
		{"第三人称", "watch的第三人称单数形式", "watch"},
		{"复数", "child 的复数", "child"},
		{"并列形态", "work的过去式和过去分词", "work"},
		{"旧数据字面量换行", `work的过去式\nwork的过去分词`, "work"},
		{"多个关系", "work的过去式；work的过去分词", "work"},
		{"已有语义", "锯；see的过去式", ""},
		{"不同原形", "good的比较级；well的比较级", ""},
		{"未知关系", "参见 oversee", ""},
		{"空释义", "", ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			entry := &DictionaryEntry{Headword: "inflected", RawMeanings: []Meaning{{Translations: []string{tt.text}}}}
			assert.Equal(t, tt.lemma, InflectionLemma(entry))
		})
	}
	t.Run("结构化信息冲突时回退", func(t *testing.T) {
		entry := &DictionaryEntry{Headword: "oversaw", Lemma: "oversee", Exchange: map[string]string{"0": "oversee"}, RawMeanings: []Meaning{{Translations: []string{"oversee的过去式"}}}}
		assert.Equal(t, "oversee", InflectionLemma(entry))
		entry.Exchange["0"] = "see"
		assert.Empty(t, InflectionLemma(entry))
		entry.Exchange["0"] = "oversee/see"
		assert.Empty(t, InflectionLemma(entry))
	})
}

func TestInflectionMeaningLayers(t *testing.T) {
	relation := []Meaning{{Translations: []string{"oversee的过去式"}}}
	lexical := []Meaning{{Translations: []string{"监管"}}}
	for _, tt := range []struct {
		name  string
		entry DictionaryEntry
		want  string
	}{
		{"lemma字段提供原形", DictionaryEntry{Headword: "oversaw", Lemma: "oversee", RawMeanings: relation}, "oversee"},
		{"exchange字段提供原形", DictionaryEntry{Headword: "oversaw", Exchange: map[string]string{"0": "oversee"}, RawMeanings: relation}, "oversee"},
		{"自身引用", DictionaryEntry{Headword: "oversee", RawMeanings: relation}, ""},
		{"已有有效review释义", DictionaryEntry{Headword: "oversaw", RawMeanings: relation, ReviewMeanings: lexical}, ""},
		{"选中的review仍为关系", DictionaryEntry{Headword: "oversaw", RawMeanings: lexical, ReviewMeanings: relation}, "oversee"},
		{"混合义项", DictionaryEntry{Headword: "oversaw", RawMeanings: append(append([]Meaning{}, relation...), lexical...)}, ""},
	} {
		t.Run(tt.name, func(t *testing.T) { assert.Equal(t, tt.want, InflectionLemma(&tt.entry)) })
	}
}

func TestLexicalMeanings(t *testing.T) {
	for _, tt := range []struct {
		name        string
		input, want []Meaning
	}{
		{"纯关系的旧换行", []Meaning{{Translations: []string{`work的过去式\nwork的过去分词`}}}, nil},
		{"混合义项保留词性", []Meaning{{Pos: "v", Translations: []string{"放置"}}, {Pos: "v", Translations: []string{"lie的过去式"}}}, []Meaning{{Pos: "v", Translations: []string{"放置"}}}},
		{"同一translation混合", []Meaning{{Pos: "v", Translations: []string{`放置；lie的过去式\n铺设`}}}, []Meaning{{Pos: "v", Translations: []string{"放置；铺设"}}}},
		{"实际换行混合", []Meaning{{Translations: []string{"放置\nlie的过去式"}}}, []Meaning{{Translations: []string{"放置"}}}},
		{"空内容", []Meaning{{Translations: []string{" ", `;；\n`}}}, nil},
		{"有效内容保持原样", []Meaning{{Pos: "v", Translations: []string{"放置；铺设", "安排"}}}, []Meaning{{Pos: "v", Translations: []string{"放置；铺设", "安排"}}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			before := make([]Meaning, len(tt.input))
			for i, m := range tt.input {
				before[i] = Meaning{Pos: m.Pos, Translations: append([]string(nil), m.Translations...)}
			}
			got := LexicalMeanings(tt.input)
			assert.Equal(t, tt.want, got)
			if len(got) > 0 {
				got[0].Translations[0] = "changed"
			}
			assert.Equal(t, before, tt.input, "不修改原始释义或共享切片")
		})
	}
}
