package vim

import "fmt"

type trie[K comparable, V comparable] struct {
	children []*node[K, V]
}

type node[K comparable, V comparable] struct {
	key   K
	value V

	isLeaf bool
	// NOTE: This maybe shouldn't be a slice but an O(1) lookup type from T -> child
	// But then again that's only really going to save time when restricting to the alphabet,
	// as then we can use rune(character) as an array index,
	// something like hashes are way slower than search over ~10 elements probably
	children []*node[K, V]
}

func (t *trie[K, V]) getChild(key K) (index int, found bool) {
	for i, child := range t.children {
		if child.key == key {
			return i, true
		}
	}

	return 0, false
}
func (n *node[K, V]) getChild(key K) (index int, found bool) {
	for i, child := range n.children {
		if child.key == key {
			return i, true
		}
	}

	return 0, false
}

func (t *trie[K, V]) Get(key []K) (value V, found bool) {
	if len(key) == 0 {
		found = true
		return
	}

	if i, ok := t.getChild(key[0]); ok {
		return t.children[i].get(key[1:])
	}

	return
}
func (n *node[K, V]) get(key []K) (value V, found bool) {
	for _, k := range key {
		if i, ok := n.getChild(k); ok {
			n = n.children[i]
		} else {
			found = false
			return
		}
	}

	if n.isLeaf {
		return n.value, true
	} else {
		found = false
		return
	}
}

func (t *trie[K, V]) ContainsPath(key []K) bool {
	result := true

	if len(key) == 0 {
		return true
	}

	if i, ok := t.getChild(key[0]); ok {
		result = t.children[i].containsPath(key[1:])
	} else {
		result = false
	}

	return result
}
func (n *node[K, V]) containsPath(key []K) bool {
	for i, k := range key {
		fmt.Printf("Iterated %d, now %v\n", i, n)
		if i, ok := n.getChild(k); !ok {
			return false
		} else {
			n = n.children[i]
		}
	}

	return true
}

func (t *trie[K, V]) Insert(key []K, value V) (changed bool) {
	changed = false

	if len(key) == 0 {
		return false
	}

	if i, ok := t.getChild(key[0]); ok {
		changed = t.children[i].insert(key[1:], value)
	} else {
		newChild := &node[K, V]{
			key:      key[0],
			isLeaf:   len(key) == 1,
			children: []*node[K, V]{},
		}

		if newChild.isLeaf {
			newChild.value = value
			return true
		}

		t.children = append(t.children, newChild)
		changed = true

		t.children[len(t.children)-1].insert(key[1:], value)
	}

	return changed
}
func (n *node[K, V]) insert(key []K, value V) (changed bool) {
	changed = false

	for i, k := range key {
		isLastIteration := i == len(key)-1

		if j, ok := n.getChild(k); ok {
			n = n.children[j]

			if isLastIteration {
				if !n.isLeaf {
					fmt.Println("true because not leaf")
					n.isLeaf = true
					n.value = value
					changed = true
				} else if n.value != value {
					fmt.Println("true because value doesn't match")
					n.value = value
					changed = true
				}
			}

			continue
		}

		newChild := &node[K, V]{
			key:      k,
			isLeaf:   isLastIteration,
			children: []*node[K, V]{},
		}

		if isLastIteration {
			newChild.value = value
		}

		n.children = append(n.children, newChild)
		changed = true

		n = n.children[len(n.children)-1]
	}

	return changed
}
