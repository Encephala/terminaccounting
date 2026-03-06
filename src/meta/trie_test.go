package meta

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInsertTrieKeysOnly(t *testing.T) {
	var trie Trie[int]

	tests := []struct {
		value           string
		changedExpected bool
	}{
		{"abc", true},
		{"abcd", true},
		{"efg", true},
		{"efg", false},
	}

	for _, test := range tests {
		changed := trie.Insert(strings.Split(test.value, ""), 0)
		assert.Equal(t, changed, test.changedExpected)
	}

	for _, test := range tests {
		_, exists := trie.get(strings.Split(test.value, ""))
		assert.True(t, exists)
	}

	_, exists := trie.get([]string{"a", "b"})
	assert.False(t, exists)
}

// By default, the root element has empty string as key, but it should also have "isLeaf" as bool
func TestDefaultValueSane(t *testing.T) {
	var trie Trie[int]

	_, exists := trie.get([]string{})
	assert.False(t, exists)

	_, exists = trie.get([]string{""})
	assert.False(t, exists)
}

func TestHandleEmptyKey(t *testing.T) {
	var trie Trie[int]

	changed := trie.Insert([]string{}, 0)
	assert.False(t, changed)

	_, exists := trie.get([]string{})
	assert.False(t, exists)
}

func TestInsertTrieWithValues(t *testing.T) {
	var trie Trie[int]

	trie.Insert([]string{"f", "1"}, 0)
	trie.Insert([]string{"f", "2"}, 1)

	f1, _ := trie.get([]string{"f", "1"})
	assert.Equal(t, f1, 0)

	f2, _ := trie.get([]string{"f", "2"})
	assert.Equal(t, f2, 1)
}

func TestTrieInsertValueOnExistingPath(t *testing.T) {
	var trie Trie[int]

	trie.Insert(strings.Split("asdf", ""), 0)

	_, ok := trie.get(strings.Split("as", ""))
	assert.False(t, ok)

	trie.Insert(strings.Split("as", ""), 69)
	result, ok := trie.get(strings.Split("as", ""))

	assert.True(t, ok)
	assert.Equal(t, result, 69)
}

func TestContainsPath(t *testing.T) {
	var trie Trie[int]

	trie.Insert(strings.Split("abc", ""), 0)

	tests := []struct {
		testValue string
		expected  bool
	}{
		{"a", true},
		{"ab", true},
		{"abc", true},
		{"acb", false},
		{"abcd", false},
		{"xd", false},
	}

	for _, test := range tests {
		result := trie.containsPath(strings.Split(test.testValue, ""))

		assert.Equal(t, result, test.expected)
	}
}

func TestAutocomplete(t *testing.T) {
	var trie Trie[int]

	trie.Insert(strings.Split("quit", ""), 0)
	trie.Insert(strings.Split("messages", ""), 1)

	tests := []struct {
		input    string
		expected []string
	}{
		// Empty path returns nil
		{"", nil},
		// Path not in trie returns nil
		{"xyz", nil},
		// Exact match (already a leaf) returns nil — "quit" -> "quit" is not an autocompletion
		{"quit", nil},
		// Prefix completes to the leaf
		{"q", []string{"q", "u", "i", "t"}},
		{"qu", []string{"q", "u", "i", "t"}},
		{"qui", []string{"q", "u", "i", "t"}},
		{"m", []string{"m", "e", "s", "s", "a", "g", "e", "s"}},
		{"mess", []string{"m", "e", "s", "s", "a", "g", "e", "s"}},
	}

	for _, test := range tests {
		var path []string
		if test.input != "" {
			path = strings.Split(test.input, "")
		}
		result := trie.autocomplete(path)
		assert.Equal(t, test.expected, result, "input: %q", test.input)
	}
}
