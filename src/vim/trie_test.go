package vim

import (
	"strings"
	"testing"
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
		if changed != test.changedExpected {
			t.Errorf("Expected changed to be %t for %q, got %t", test.changedExpected, test.value, changed)
		}
	}

	for _, test := range tests {
		_, exists := trie.get(strings.Split(test.value, ""))
		if !exists {
			t.Errorf("Failed to find %q in trie", test.value)
		}
	}

	_, exists := trie.get([]string{"a", "b"})
	if exists {
		t.Errorf("Expected partial path %q to not be found, but it was", "ab")
	}
}

// By default, the root element has empty string as key, but it should also have "isLeaf" as bool
func TestDefaultValueSane(t *testing.T) {
	var trie Trie[int]

	_, exists := trie.get([]string{})
	if exists {
		t.Errorf("Found empty key in trie, expected not to")
	}

	_, exists = trie.get([]string{""})
	if exists {
		t.Errorf("Found %q in trie, expected not to", "")
	}
}

func TestHandleEmptyKey(t *testing.T) {
	var trie Trie[int]

	changed := trie.Insert([]string{}, 0)
	if changed {
		t.Errorf("Expected changed to be false for inserting %q, got %t", "", changed)
	}

	_, exists := trie.get([]string{})
	if exists {
		t.Errorf("Found %q in trie, expected not to", "")
	}
}

func TestInsertTrieWithValues(t *testing.T) {
	var trie Trie[int]

	trie.Insert([]string{"f", "1"}, 0)
	trie.Insert([]string{"f", "2"}, 1)

	f1, _ := trie.get([]string{"f", "1"})
	if f1 != 0 {
		t.Errorf("Expected to get back %q, got back %q", 0, f1)
	}

	f2, _ := trie.get([]string{"f", "2"})
	if f2 != 1 {
		t.Errorf("Expected to get back %q, got back %q", 1, f2)
	}
}

func TestTrieInsertValueOnExistingPath(t *testing.T) {
	var trie Trie[int]

	trie.Insert(strings.Split("asdf", ""), 0)

	_, ok := trie.get(strings.Split("as", ""))
	if ok {
		t.Errorf("Expected to not find leaf node at partial path, but did")
	}

	trie.Insert(strings.Split("as", ""), 69)
	result, ok := trie.get(strings.Split("as", ""))
	if !ok {
		t.Errorf("Expected to find leaf at partial path, but didn't")
	}

	if result != 69 {
		t.Errorf("Expected to get %q, got %q", 69, result)
	}
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

		if result != test.expected {
			t.Fatalf("Got %t for path %q, expected %t", result, test.testValue, test.expected)
		}
	}
}
