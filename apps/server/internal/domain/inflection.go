package domain

import (
	"regexp"
	"strings"
)

// 只接受完整的词形关系，避免把含独立词义的条目误判为关系词条。
const inflectionForm = `(?:过去式|过去分词|现在分词|第三人称单数|复数|比较级|最高级)(?:形式)?`

var inflectionRelation = regexp.MustCompile(`(?i)^([a-z]+(?:[-'][a-z]+)*)\s*的\s*` + inflectionForm + `(?:\s*(?:和|及|与|、|或)\s*` + inflectionForm + `)*[。.]?$`)
var lemmaWord = regexp.MustCompile(`^[a-z]+(?:[-'][a-z]+)*$`)

// InflectionLemma 只为仅含词形关系的默认释义返回单一、无冲突的原形。
// 这是展示补全的目标，不参与导入归并（data-model.md §7.1）。
func InflectionLemma(entry *DictionaryEntry) string {
	meanings, _ := EffectiveReviewMeaning(nil, entry.ReviewMeanings, entry.RawMeanings)
	textLemma := ""
	for _, meaning := range meanings {
		for _, translation := range meaning.Translations {
			for _, part := range strings.FieldsFunc(translation, func(r rune) bool { return r == ';' || r == '；' || r == '\n' }) {
				part = strings.TrimSpace(part)
				if part == "" {
					continue
				}
				match := inflectionRelation.FindStringSubmatch(part)
				if match == nil {
					return ""
				}
				candidate := NormalizeWord(match[1])
				if textLemma != "" && candidate != textLemma {
					return ""
				}
				textLemma = candidate
			}
		}
	}
	if textLemma == "" {
		return ""
	}
	// 优先结构化目标；任何非自身映射与文本冲突时宁可不补全。
	target := NormalizeWord(entry.Exchange["0"])
	lemma := NormalizeWord(entry.Lemma)
	headword := NormalizeWord(entry.Headword)
	if target == "" || target == headword {
		target = lemma
	}
	if target == "" || target == headword {
		target = textLemma
	}
	if target != textLemma || (lemma != "" && lemma != headword && lemma != target) || target == headword || !lemmaWord.MatchString(target) {
		return ""
	}
	return target
}

// HasLexicalMeaning 排除空内容与全部由词形关系组成的目标释义。
// 不尝试解析目标的下一跳，从而不会递归或循环查询。
func HasLexicalMeaning(meanings []Meaning) bool {
	for _, meaning := range meanings {
		for _, translation := range meaning.Translations {
			for _, part := range strings.FieldsFunc(translation, func(r rune) bool { return r == ';' || r == '；' || r == '\n' }) {
				part = strings.TrimSpace(part)
				if part != "" && !inflectionRelation.MatchString(part) {
					return true
				}
			}
		}
	}
	return false
}
