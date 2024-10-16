package vim

import (
	"strings"
	"testing"
)

func TestInsertTrieKeysOnly(t *testing.T) {
	var trie Trie

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
		changed := trie.Insert(strings.Split(test.value, ""), CompletedMotionMsg{})
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

func TestHandleEmptyKey(t *testing.T) {
	var trie Trie

	changed := trie.Insert([]string{}, CompletedMotionMsg{})
	if changed {
		t.Errorf("Expected changed to be false for %q, got %t", "", changed)
	}

	_, exists := trie.get([]string{})
	if exists {
		t.Errorf("Found %q in trie, expected not to", "")
	}
}

func TestInsertTrieWithValues(t *testing.T) {
	var trie Trie

	trie.Insert([]string{"f", "1"}, CompletedMotionMsg{Data: "f1"})
	trie.Insert([]string{"f", "2"}, CompletedMotionMsg{Data: "f2"})

	f1, _ := trie.get([]string{"f", "1"})
	if f1.Data.(string) != "f1" {
		t.Errorf("Expected to get back %q, got back %q", "f1", f1.Data.(string))
	}

	f2, _ := trie.get([]string{"f", "2"})
	if f2.Data.(string) != "f2" {
		t.Errorf("Expected to get back %q, got back %q", "f2", f2.Data.(string))
	}
}

func TestTrieInsertValueOnExistingPath(t *testing.T) {
	var trie Trie

	trie.Insert(strings.Split("asdf", ""), CompletedMotionMsg{})

	_, ok := trie.get(strings.Split("as", ""))
	if ok {
		t.Errorf("Expected to not find leaf node at partial path, but did")
	}

	trie.Insert(strings.Split("as", ""), CompletedMotionMsg{Data: "found"})
	result, ok := trie.get(strings.Split("as", ""))
	if !ok {
		t.Errorf("Expected to find leaf at partial path, but didn't")
	}

	if result.Data.(string) != "found" {
		t.Errorf("Expected to get %q, got %q", "found", result.Data.(string))
	}
}

func TestContainsPath(t *testing.T) {
	var trie Trie

	trie.Insert(strings.Split("abc", ""), CompletedMotionMsg{})

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
