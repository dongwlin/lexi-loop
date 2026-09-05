package domain

import "strings"

// NormalizeWord 对单个生词输入做纯字符串归一：去除首尾空白并统一小写
// （docs/dictionary/normalization.md §2 的 trim / lowercase 规则）。
// 它是导入流程的前置步骤；lemma 解析与「不确定不归并」流程由
// DictionaryService 编排（docs/backend/structure.md §4.1），不在本包。
func NormalizeWord(input string) string {
	return strings.ToLower(strings.TrimSpace(input))
}
