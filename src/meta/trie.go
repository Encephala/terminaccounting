package meta

type Trie[T any] struct {
	key string

	isLeaf bool
	value  T

	children []*Trie[T]
}

func (t *Trie[T]) getChild(key string) (child *Trie[T], found bool) {
	for _, child := range t.children {
		if child.key == key {
			return child, true
		}
	}

	return nil, false
}

func (t *Trie[T]) get(path []string) (T, bool) {
	var null T

	if len(path) == 0 {
		return null, false
	}

	for _, value := range path {
		if child, ok := t.getChild(value); !ok {
			return null, false
		} else {
			t = child
		}
	}

	if t.isLeaf {
		return t.value, true
	} else {
		return null, false
	}
}

func (t *Trie[T]) containsPath(path []string) bool {
	for _, value := range path {
		if child, ok := t.getChild(value); !ok {
			return false
		} else {
			t = child
		}
	}

	return true
}

// Short-circuiting method to get a leaf node from the current path
// Might upgrade it to not be short-circuiting anymore in the future (i.e. return [][]string, all possible autocompletions)
// but KISS for now, I don't have that many commands anwyays
func (t *Trie[T]) autocompletion(path []string) []string {
	// Walk Trie until along the given path
	// If child not found, there is no autocomplete to be had
	for _, value := range path {
		if child, ok := t.getChild(value); ok {
			t = child
		} else {
			return nil
		}
	}

	if t.isLeaf {
		return path
	}

	// Now keep walking the Trie along the first child (NB: first child -> short-circuiting),
	// until a leaf is found
	result := path

	for len(t.children) > 0 {
		t = t.children[0]
		result = append(result, t.key)
	}

	return result
}

func (t *Trie[T]) Insert(path []string, value T) (changed bool) {
	changed = false

	for i, key := range path {
		isFinalValue := i == len(path)-1
		if child, ok := t.getChild(key); ok {
			t = child

			// Actually, if this happens, we can drop all t's children,
			// because as soon as a motion resolves, it executes
			if isFinalValue {
				// NOTE: When t.isLeaf, this does not check if the t.value is different from the provided value,
				// i.e. it is not possible to update a value in the trie
				if !t.isLeaf {
					t.isLeaf = true
					t.value = value
				}
			}

			continue
		}

		newChild := &Trie[T]{
			key:      key,
			isLeaf:   isFinalValue,
			children: []*Trie[T]{},
		}
		if isFinalValue {
			newChild.value = value
		}

		t.children = append(t.children, newChild)
		changed = true
		t = newChild
	}

	return changed
}
