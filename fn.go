// Copyright 2016 Evans. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cedar

import (
	"errors"
)

var (
	// ErrNoKey not have key error
	ErrNoKey = errors.New("cedar: not have key")
	// ErrNoVal not have value error
	ErrNoVal = errors.New("cedar: not have val")
	// ErrInvalidKey invalid key error
	ErrInvalidKey = errors.New("cedar: invalid key")
	// ErrInvalidVal invalid value error
	ErrInvalidVal = errors.New("cedar: invalid val")
)

func isReduced(reduced ...bool) bool {
	if len(reduced) > 0 && !reduced[0] {
		return false
	}

	return true
}

func (cd *Cedar) get(key []byte, from, pos int) *int {
	to := cd.getNode(key, from, pos)
	return &cd.array[to].baseV
}

// getNode get the follow node by key, split by update()
func (cd *Cedar) getNode(key []byte, from, pos int) int {
	for ; pos < len(key); pos++ {
		if cd.Reduced {
			value := cd.array[from].baseV
			if value >= 0 && value != ValLimit {
				to := cd.follow(from, 0)
				cd.array[to].baseV = value
			}
		}

		from = cd.follow(from, key[pos])
	}

	to := from
	if cd.array[from].baseV < 0 || !cd.Reduced {
		to = cd.follow(from, 0)
	}

	return to
}

// Jump jump a node `from` to another node by following the `path`, split by find()
func (cd *Cedar) Jump(key []byte, from int) (to int, err error) {
	// pos := 0
	// recursively matching the key.
	for _, k := range key {
		if cd.array[from].baseV >= 0 && cd.Reduced {
			return from, ErrNoKey
		}

		to = cd.array[from].base(cd.Reduced) ^ int(k)
		if cd.array[to].check != from {
			return from, ErrNoKey
		}
		from = to
	}

	return to, nil
}

// Find key from double array trie, with `from` as the cursor to traverse the nodes.
func (cd *Cedar) Find(key []byte, from int) (int, error) {
	to, err := cd.Jump(key, from)
	if cd.Reduced {
		if cd.array[from].baseV >= 0 {
			if err == nil && to != 0 {
				return cd.array[to].baseV, nil
			}
			return 0, ErrNoKey
		}
	}

	// return the value of the node if `check` is correctly marked fpr the ownership,
	// otherwise it means no value is stored.
	n := cd.array[cd.array[to].base(cd.Reduced)]
	if n.check != to {
		return 0, ErrNoKey
	}
	return n.baseV, nil
}

// Value get the path value
func (cd *Cedar) Value(path int) (val int, err error) {
	val = cd.array[path].baseV
	if val >= 0 {
		return val, nil
	}

	to := cd.array[path].base(cd.Reduced)
	if cd.array[to].check == path && cd.array[to].baseV >= 0 {
		return cd.array[to].baseV, nil
	}

	return 0, ErrNoVal
}

// Insert the key for the value on []byte
func (cd *Cedar) Insert(key []byte, val int) error {
	if val < 0 || val >= ValLimit {
		return ErrInvalidVal
	}

	p := cd.get(key, 0, 0)
	*p = val

	return nil
}

// Update the key for the value, it is public interface that works on []byte
func (cd *Cedar) Update(key []byte, value int) error {
	p := cd.get(key, 0, 0)

	if *p == ValLimit && cd.Reduced {
		*p = value
		return nil
	}

	*p += value
	return nil
}

// Delete the key from the trie, the internal interface that works on []byte
func (cd *Cedar) Delete(key []byte) error {
	// move the cursor to the right place and use erase__ to delete it.
	to, err := cd.Jump(key, 0)
	if err != nil {
		return ErrNoKey
	}

	if cd.array[to].baseV < 0 && cd.Reduced {
		base := cd.array[to].base(cd.Reduced)
		if cd.array[base].check == to {
			to = base
		}
	}

	if !cd.Reduced {
		to = cd.array[to].base(cd.Reduced)
	}

	from := to
	for to > 0 {
		if cd.Reduced {
			from = cd.array[to].check
		}
		base := cd.array[from].base(cd.Reduced)
		label := byte(to ^ base)

		hasSibling := cd.nInfos[to].sibling != 0 || cd.nInfos[from].child != label
		// if the node has siblings, then remove `e` from the sibling.
		if hasSibling {
			cd.popSibling(from, base, label)
		}

		// maintain the data structures.
		cd.pushENode(to)
		// traverse to the parent.
		to = from

		// if it has sibling then this layer has more than one nodes, then we are done.
		if hasSibling {
			break
		}
	}

	return nil
}

// Get get the key value on []byte
func (cd *Cedar) Get(key []byte) (value int, err error) {
	to, err := cd.Jump(key, 0)
	if err != nil {
		return 0, err
	}

	return cd.Value(to)
}

// ExactMatch to check if `key` is in the dictionary.
func (cd *Cedar) ExactMatch(key []byte) (int, bool) {
	from := 0
	val, err := cd.Find(key, from)
	if err != nil {
		return 0, false
	}
	return val, true
}

// PrefixMatch return the collection of the common prefix
// in the dictionary with the `key`
func (cd *Cedar) PrefixMatch(key []byte, n ...int) (ids []int) {
	num := 0
	if len(n) > 0 {
		num = n[0]
	}

	for from, i := 0, 0; i < len(key); i++ {
		to, err := cd.Jump(key[i:i+1], from)
		if err != nil {
			break
		}

		_, err = cd.Value(to)
		if err == nil {
			ids = append(ids, to)
			num--
			if num == 0 {
				return
			}
		}

		from = to
	}

	return
}

// PrefixPredict eturn the list of words in the dictionary
// that has `key` as their prefix
func (cd *Cedar) PrefixPredict(key []byte, n ...int) (ids []int) {
	num := 0
	if len(n) > 0 {
		num = n[0]
	}

	root, err := cd.Jump(key, 0)
	if err != nil {
		return
	}

	for from, err := cd.begin(root); err == nil; from, err = cd.next(from, root) {
		ids = append(ids, from)
		num--
		if num == 0 {
			return
		}
	}

	return
}

// To get the cursor of the first leaf node starting by `from`
func (cd *Cedar) begin(from int) (to int, err error) {
	// recursively traversing down to look for the first leaf.
	for c := cd.nInfos[from].child; c != 0; {
		from = cd.array[from].base(cd.Reduced) ^ int(c)
		c = cd.nInfos[from].child
	}

	if cd.array[from].base() > 0 {
		return cd.array[from].base(), nil
	}

	// To return the value of the leaf.
	return from, nil
}

// To move the cursor from one leaf to the next for the common prefix predict.
func (cd *Cedar) next(from int, root int) (to int, err error) {
	c := cd.nInfos[from].sibling
	if !cd.Reduced {
		c = cd.nInfos[cd.array[from].base(cd.Reduced)].sibling
	}

	// traversing up until there is a sibling or it has reached the root.
	for c == 0 && from != root && cd.array[from].check >= 0 {
		from = cd.array[from].check
		c = cd.nInfos[from].sibling
	}

	if from == root || cd.array[from].check < 0 {
		return 0, ErrNoKey
	}

	// it has a sibling so we leverage on `begin` to traverse the subtree down again.
	from = cd.array[cd.array[from].check].base(cd.Reduced) ^ int(c)
	return cd.begin(from)
}
