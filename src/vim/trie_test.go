package vim

import (
	"strings"
	"testing"
)

func TestInsertTrieKeysOnly(t *testing.T) {
	var trie trie

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
		changed := trie.Insert(strings.Split(test.value, ""))
		if changed != test.changedExpected {
			t.Errorf("Expected changed to be %t for %q, got %t", test.changedExpected, test.value, changed)
		}
	}

	for _, test := range tests {
		exists := trie.Get(strings.Split(test.value, ""))
		if !exists {
			t.Errorf("Failed to find %q in trie", test.value)
		}
	}

	exists := trie.Get([]string{"a", "b"})
	if exists {
		t.Errorf("Expected partial path %q to not be found, but it was", "ab")
	}
}

func TestHandleEmptyValue(t *testing.T) {
	var trie trie

	changed := trie.Insert([]string{})
	if changed {
		t.Errorf("Expected changed to be false for %q, got %t", "", changed)
	}

	exists := trie.Get([]string{})
	if !exists {
		t.Errorf("Failed to find %q in trie", "")
	}
}

func TestInsertTrieWithValues(t *testing.T) {
	var trie trie

	tests := []struct {
		key             string
		changedExpected bool
	}{
		{
			"asdf",
			true,
		},
		{
			"asdf",
			false,
		},

		{
			"efg",
			true,
		},
		{
			"efgh",
			true,
		},
	}

	for _, test := range tests {
		changed := trie.Insert(strings.Split(test.key, ""))
		if changed != test.changedExpected {
			t.Errorf("Expected changed to be %t when inserting %v, got %t",
				test.changedExpected, test.key, changed)
		}
	}

	valueTests := []struct {
		key []string
	}{
		{
			strings.Split("asdf", ""),
		},
		{
			strings.Split("efg", ""),
		},
		{
			strings.Split("efgh", ""),
		},
	}
	for _, test := range valueTests {
		found := trie.Get(test.key)
		if !found {
			t.Errorf("Failed to find key %q in tree", test.key)
		}
	}
}

func TestContainsPath(t *testing.T) {
	var trie trie

	trie.Insert(strings.Split("abc", ""))

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
		result := trie.ContainsPath(strings.Split(test.testValue, ""))

		if result != test.expected {
			t.Fatalf("Got %t for path %q, expected %t", result, test.testValue, test.expected)
		}
	}
}
