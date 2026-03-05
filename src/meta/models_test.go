package meta_test

import (
	"terminaccounting/meta"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompileNotes(t *testing.T) {
	tests := []struct {
		input    string
		expected meta.Notes
	}{
		{"", nil},
		{"hello", meta.Notes{"hello"}},
		{"hello\n", meta.Notes{"hello"}},
		{"a\nb", meta.Notes{"a", "b"}},
		{"a\nb\n", meta.Notes{"a", "b"}},
		{"a\nb\n\n", meta.Notes{"a", "b"}},
		// Middle blank lines are preserved; only trailing blanks are trimmed
		{"a\n\nb", meta.Notes{"a", "", "b"}},
		{"a\n\nb\n\n", meta.Notes{"a", "", "b"}},
	}

	for _, testCase := range tests {
		result := meta.CompileNotes(testCase.input)
		assert.Equal(t, testCase.expected, result, "input: %q", testCase.input)
	}
}