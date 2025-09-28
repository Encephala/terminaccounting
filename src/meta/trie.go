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
	var zeroResult T

	if len(path) == 0 {
		return zeroResult, false
	}

	for _, value := range path {
		if child, ok := t.getChild(value); !ok {
			return zeroResult, false
		} else {
			t = child
		}
	}

	if t.isLeaf {
		return t.value, true
	} else {
		return zeroResult, false
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
