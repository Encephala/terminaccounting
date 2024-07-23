package vim

import (
	"testing"
)

func TestInsertTrieKeysOnly(t *testing.T) {
	var trie trie[rune, string]

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
		changed := trie.Insert([]rune(test.value), "")
		if changed != test.changedExpected {
			t.Errorf("Expected changed to be %t for %q, got %t", test.changedExpected, test.value, changed)
		}
	}

	for _, test := range tests {
		_, exists := trie.Get([]rune(test.value))
		if !exists {
			t.Errorf("Failed to find %q in trie", test.value)
		}
	}
}

func TestHandleEmptyValue(t *testing.T) {
	var trie trie[rune, string]

	changed := trie.Insert([]rune{}, "")
	if changed {
		t.Errorf("Expected changed to be false for %q, got %t", "", changed)
	}

	_, exists := trie.Get([]rune{})
	if !exists {
		t.Errorf("Failed to find %q in trie", "")
	}
}

func TestInsertTrieWithValues(t *testing.T) {
	var trie trie[rune, string]

	tests := []struct {
		key             string
		value           string
		changedExpected bool
	}{
		{
			"asdf",
			"first",
			true,
		},
		{
			"asdf",
			"second",
			true,
		},
		{
			"asdf",
			"second",
			false,
		},

		{
			"efg",
			"third",
			true,
		},
		{
			"efgh",
			"fourth",
			true,
		},
	}

	for _, test := range tests {
		changed := trie.Insert([]rune(test.key), test.value)
		if changed != test.changedExpected {
			t.Errorf("Expected changed to be %t when inserting %v: %+v, got %t",
				test.changedExpected, test.key, test.value, changed)
		}
	}

	valueTests := []struct {
		key           string
		expectedValue string
	}{
		{
			"asdf",
			"second",
		},
		{
			"efg",
			"third",
		},
		{
			"efgh",
			"fourth",
		},
	}
	for _, test := range valueTests {
		result, ok := trie.Get([]rune(test.key))
		if !ok {
			t.Errorf("Failed to find key %q in tree", test.key)
		}

		if result != test.expectedValue {
			t.Errorf("Expected value for %q to be %v, got %v", test.key, test.expectedValue, result)
		}
	}
}

func TestContainsPath(t *testing.T) {
	var trie trie[rune, string]

	trie.Insert([]rune("abc"), "")

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
		result := trie.ContainsPath([]rune(test.testValue))

		if result != test.expected {
			t.Fatalf("Got %t for path %q, expected %t", result, test.testValue, test.expected)
		}
	}
}
