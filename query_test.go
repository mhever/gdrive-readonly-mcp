package main

import "testing"

func TestEscapeQuery(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty string", input: "", want: ""},
		{name: "no special chars", input: "hello world", want: "hello world"},
		{name: "single quote", input: "it's a file", want: `it\'s a file`},
		{name: "backslash", input: `path\to\file`, want: `path\\to\\file`},
		{name: "combined special chars", input: `it's a\path`, want: `it\'s a\\path`},
		{name: "multiple single quotes", input: "a'b'c", want: `a\'b\'c`},
		{name: "multiple backslashes", input: `a\\b`, want: `a\\\\b`},
		{name: "unicode characters", input: "文件名'测试", want: `文件名\'测试`},
		{name: "backslash then quote", input: `\'`, want: `\\\'`},
		{name: "only single quote", input: "'", want: `\'`},
		{name: "only backslash", input: `\`, want: `\\`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := escapeQuery(tt.input)
			if got != tt.want {
				t.Errorf("escapeQuery(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
