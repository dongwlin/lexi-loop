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
			for _, part := range relationParts(translation) {
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

// relationParts 统一处理新旧词典数据的换行与分号，只返回非空片段。
func relationParts(translation string) []string {
	decoded := strings.ReplaceAll(translation, `\n`, "\n")
	parts := strings.FieldsFunc(decoded, func(r rune) bool { return r == ';' || r == '；' || r == '\n' })
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			result = append(result, part)
		}
	}
	return result
}

// LexicalMeanings 返回仅含实际语义的独立副本，保留词性和义项顺序。
// 原形自身的关系说明不能成为下一层词形的补充，也不继续追溯。
func LexicalMeanings(meanings []Meaning) []Meaning {
	var result []Meaning
	for _, meaning := range meanings {
		var translations []string
		for _, translation := range meaning.Translations {
			parts := relationParts(translation)
			var lexical []string
			for _, part := range parts {
				if !inflectionRelation.MatchString(part) {
					lexical = append(lexical, part)
				}
			}
			if len(lexical) == 0 {
				continue
			}
			if len(lexical) == len(parts) {
				// 没有关系片段时保持原有展示格式。
				translations = append(translations, translation)
			} else {
				translations = append(translations, strings.Join(lexical, "；"))
			}
		}
		if len(translations) > 0 {
			result = append(result, Meaning{Pos: meaning.Pos, Translations: translations})
		}
	}
	return result
}
