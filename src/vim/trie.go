package vim

import "fmt"

type Trie[K comparable, V comparable] struct {
	Children []*node[K, V]
}

type node[K comparable, V comparable] struct {
	key   K
	value V

	isLeaf bool
	// TODO: This maybe shouldn't be a slice but an O(1) lookup type from T -> child
	// But then again that's only really going to save time when restricting to the alphabet,
	// as then we can use rune(character) as an array index,
	// something like hashes are way slower than search over ~10 elements probably
	Children []*node[K, V]
}

func (t *Trie[K, V]) getChild(key K) (index int, found bool) {
	for i, child := range t.Children {
		if child.key == key {
			return i, true
		}
	}

	return 0, false
}
func (n *node[K, V]) getChild(key K) (index int, found bool) {
	for i, child := range n.Children {
		if child.key == key {
			return i, true
		}
	}

	return 0, false
}

func (t *Trie[K, V]) Get(key []K) (value V, found bool) {
	if len(key) == 0 {
		found = true
		return
	}

	if i, ok := t.getChild(key[0]); ok {
		return t.Children[i].Get(key[1:])
	}

	return
}
func (n *node[K, V]) Get(key []K) (value V, found bool) {
	for _, v := range key {
		if i, ok := n.getChild(v); ok {
			n = n.Children[i]
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

func (t *Trie[K, V]) Insert(key []K, value V) (changed bool) {
	changed = false

	if len(key) == 0 {
		return false
	}

	if i, ok := t.getChild(key[0]); ok {
		changed = t.Children[i].Insert(key[1:], value)
	} else {
		newChild := &node[K, V]{
			key:      key[0],
			isLeaf:   len(key) == 1,
			Children: []*node[K, V]{},
		}

		if newChild.isLeaf {
			newChild.value = value
			return true
		}

		t.Children = append(t.Children, newChild)
		changed = true

		t.Children[len(t.Children)-1].Insert(key[1:], value)
	}

	return changed
}
func (n *node[K, V]) Insert(key []K, value V) (changed bool) {
	changed = false

	for i, k := range key {
		isLastIteration := i == len(key)-1

		if j, ok := n.getChild(k); ok {
			n = n.Children[j]

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
			Children: []*node[K, V]{},
		}

		if isLastIteration {
			newChild.value = value
		}

		n.Children = append(n.Children, newChild)
		changed = true

		n = n.Children[len(n.Children)-1]
	}

	return changed
}
