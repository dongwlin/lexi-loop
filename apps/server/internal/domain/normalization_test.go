package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeWord(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "纯小写单词保持不变", input: "ambiguous", want: "ambiguous"},
		{name: "大写统一为小写", input: "Derived", want: "derived"},
		{name: "去除首尾空白", input: "  constraint\t", want: "constraint"},
		{name: "空白与大写同时归一", input: " Constraints ", want: "constraints"},
		{name: "空白输入归一为空", input: "   ", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, NormalizeWord(tt.input))
		})
	}
}
