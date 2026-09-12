// Copyright 2016 Evans. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cedar

import "math"

// NInfo stores the information about the trie
type NInfo struct {
	sibling, child byte // uint8
}

// Node contains the array of `base` and `check` as specified in the paper:
// "An efficient implementation of trie structures"
// https://dl.acm.org/citation.cfm?id=146691
type Node struct {
	baseV, check int32 // int
}

// base returns the base offset of the node; `reduced` selects the encoding used
// by the reduced trie. It is non-variadic on purpose: it sits on the hot path
// of every lookup and insert.
func (n *Node) base(reduced bool) int {
	if !reduced {
		return int(n.baseV)
	}

	return -(int(n.baseV) + 1)
}

// Block stores the linked-list pointers and the stats info for blocks.
//
// All fields fit in int32 (block indexes are < 2^24, counters are <= 257,
// and eHead is a node index which is bounded by ValLimit), which halves the
// per-block footprint compared to int.
type Block struct {
	prev   int32 // previous block's index, 3 bytes width
	next   int32 // next block's index, 3 bytes width
	num    int32 // the number of slots that is free, the range is 0-256
	reject int32 // a heuristic number to make the search for free space faster...
	trial  int32 // the number of times this block has been probed by `find_places` for the free block.
	eHead  int32 // the index of the first empty elemenet in this block
}

func (b *Block) init() {
	b.num = 256    // each of block has 256 free slots at the beginning
	b.reject = 257 // initially every block need to be fully iterated through so that we can reject it to be unusable.
}

// Cedar holds all of the information about double array trie.
type Cedar struct {
	// Reduced option the reduced trie
	Reduced bool

	array  []Node // storing the `base` and `check` info from the original paper.
	nInfos []NInfo
	blocks []Block
	reject [257]int32

	blocksHeadFull   int32 // the index of the first 'Full' block, 0 means no 'Full' block
	blocksHeadClosed int32 // the index of the first 'Closed' block, 0 means no ' Closed' block
	blocksHeadOpen   int32 // the index of the first 'Open' block, 0 means no 'Open' block

	capacity int
	size     int
	ordered  bool
	maxTrial int32 // the parameter for cedar, it could be tuned for more, but the default is 1.

	// scratch buffer reused by setChild/resolve so relocations do not allocate.
	childBuf [257]byte
}

const (
	// ValLimit cedar value limit, the values are stored as int32
	ValLimit = math.MaxInt32
	// NoVal not have value
	NoVal = -1
)

// type PrefixIter struct {
// }

// New initialize the Cedar for further use
func New(reduced ...bool) *Cedar {
	cd := Cedar{
		Reduced: isReduced(reduced...),

		array:  make([]Node, 256),
		nInfos: make([]NInfo, 256),
		blocks: make([]Block, 1),

		capacity: 256,
		size:     256,
		ordered:  true,
		maxTrial: 1,
	}

	if !cd.Reduced {
		cd.array[0] = Node{baseV: 0, check: -1}
	} else {
		cd.array[0] = Node{baseV: -1, check: -1}
	}
	// make `baseV` point to the previous element, and make `check` point to the next element
	for i := int32(1); i < 256; i++ {
		cd.array[i] = Node{baseV: -(i - 1), check: -(i + 1)}
	}
	// make them link as a cyclic doubly-linked list
	cd.array[1].baseV = -255
	cd.array[255].check = -1

	cd.blocks[0].eHead = 1
	cd.blocks[0].init()

	for i := range cd.reject {
		cd.reject[i] = int32(i + 1)
	}

	return &cd
}

// follow To move in the trie by following the `label`, and insert the node if the node is not there,
// it is used by the `update` to populate the trie.
func (cd *Cedar) follow(from int, label byte) (to int) {
	base := cd.array[from].base(cd.Reduced)

	// the node is not there
	to = base ^ int(label)
	if base < 0 || cd.array[to].check < 0 {
		// allocate a e node
		to = cd.popENode(base, from, label)
		branch := to ^ int(label)

		// maintain the info in ninfo
		cd.pushSibling(from, branch, label, base >= 0)
		return
	}

	// the node is already there and the ownership is not `from`,
	// therefore a conflict.
	if int(cd.array[to].check) != from {
		// call `resolve` to relocate.
		to = cd.resolve(from, base, label)
	}

	return
}

// Mark an edge `e` as used in a trie node.
// pop empty node from block; never transfer the special block (idx = 0)
func (cd *Cedar) popENode(base, from int, label byte) int {
	e := base ^ int(label)
	if base < 0 {
		e = cd.findPlace()
	}

	idx := int32(e >> 8)
	arr := &cd.array[e]

	b := &cd.blocks[idx]
	b.num--
	// move the block at idx to the correct linked-list depending the free slots it still have.
	if b.num == 0 {
		if idx != 0 {
			// Closed to Full
			cd.transferBlock(idx, &cd.blocksHeadClosed, &cd.blocksHeadFull)
		}
	} else {
		// release empty node from empty ring
		cd.array[-arr.baseV].check = arr.check
		cd.array[-arr.check].baseV = arr.baseV

		if int32(e) == b.eHead {
			b.eHead = -arr.check
		}

		if idx != 0 && b.num == 1 && b.trial != cd.maxTrial {
			// Open to Closed
			cd.transferBlock(idx, &cd.blocksHeadOpen, &cd.blocksHeadClosed)
		}
	}

	// initialize the released node
	if !cd.Reduced {
		if label != 0 {
			cd.array[e].baseV = -1
		} else {
			cd.array[e].baseV = 0
		}
		cd.array[e].check = int32(from)
		if base < 0 {
			cd.array[from].baseV = int32(e ^ int(label))
		}

		return e
	}

	cd.array[e].baseV = ValLimit
	cd.array[e].check = int32(from)
	if base < 0 {
		cd.array[from].baseV = int32(-(e ^ int(label)) - 1)
	}

	return e
}

// Mark an edge `e` as free in a trie node.
// push empty node into empty ring
func (cd *Cedar) pushENode(e int) {
	idx := int32(e >> 8)
	b := &cd.blocks[idx]
	b.num++

	if b.num == 1 {
		b.eHead = int32(e)
		cd.array[e] = Node{baseV: int32(-e), check: int32(-e)}

		if idx != 0 {
			// Move the block from 'Full' to 'Closed' since it has one free slot now.
			cd.transferBlock(idx, &cd.blocksHeadFull, &cd.blocksHeadClosed)
		}
	} else {
		prev := b.eHead
		next := -cd.array[prev].check

		// Insert to the edge immediately after the e_head
		cd.array[e] = Node{baseV: -prev, check: -next}

		cd.array[prev].check = int32(-e)
		cd.array[next].baseV = int32(-e)

		// Move the block from 'Closed' to 'Open' since it has more than one free slot now.
		if b.num == 2 || b.trial == cd.maxTrial {
			if idx != 0 {
				// Closed to Open
				cd.transferBlock(idx, &cd.blocksHeadClosed, &cd.blocksHeadOpen)
			}
		}

		// Reset the trial stats
		b.trial = 0
	}

	if b.reject < cd.reject[b.num] {
		b.reject = cd.reject[b.num]
	}
	// reset ninfo; no child, no sibling
	cd.nInfos[e] = NInfo{}
}

// push the `label` into the sibling chain
// to from's child
func (cd *Cedar) pushSibling(from, base int, label byte, hasChild bool) {
	c := &cd.nInfos[from].child
	keepOrder := *c == 0
	if cd.ordered {
		keepOrder = label > *c
	}

	if hasChild && keepOrder {
		c = &cd.nInfos[base^int(*c)].sibling
		for cd.ordered && *c != 0 && *c < label {
			c = &cd.nInfos[base^int(*c)].sibling
		}

		// for {
		// 	c = &cd.nInfos[base^int(*c)].sibling
		// 	if cd.ordered && *c != 0 && *c < label {
		// 		break
		// 	}
		// }
	}

	cd.nInfos[base^int(label)].sibling = *c
	*c = label
}

// remove the `label` from the sibling chain.
func (cd *Cedar) popSibling(from, base int, label byte) {
	c := &cd.nInfos[from].child
	for *c != label {
		c = &cd.nInfos[base^int(*c)].sibling
	}
	*c = cd.nInfos[base^int(*c)].sibling
}

// Loop through the siblings to see which one reached the end first, which means
// it is the one with smaller in children size, and we should try ti relocate the smaller one.
// check whether to replace branching w/ the newly added node
func (cd *Cedar) consult(baseN, baseP int, cN, cP byte) bool {
	cN = cd.nInfos[baseN^int(cN)].sibling
	cP = cd.nInfos[baseP^int(cP)].sibling

	for cN != 0 && cP != 0 {
		cN = cd.nInfos[baseN^int(cN)].sibling
		cP = cd.nInfos[baseP^int(cP)].sibling
	}

	return cP != 0
}

// Collect the list of the children, and push the label as well if it is not terminal node.
// enumerate (equal to or more than one) child nodes.
//
// The result aliases cd.childBuf and is only valid until the next call.
func (cd *Cedar) setChild(base int, c, label byte, flag bool) []byte {
	child := cd.childBuf[:0]
	// 0: terminal
	if c == 0 {
		child = append(child, c)
		c = cd.nInfos[base^int(c)].sibling
	}

	if cd.ordered {
		for c != 0 && c <= label {
			child = append(child, c)
			c = cd.nInfos[base^int(c)].sibling
		}
	}

	if flag {
		child = append(child, label)
	}

	for c != 0 {
		child = append(child, c)
		c = cd.nInfos[base^int(c)].sibling
	}

	return child
}

// For the case where only one free slot is needed
func (cd *Cedar) findPlace() int {
	if cd.blocksHeadClosed != 0 {
		return int(cd.blocks[cd.blocksHeadClosed].eHead)
	}

	if cd.blocksHeadOpen != 0 {
		return int(cd.blocks[cd.blocksHeadOpen].eHead)
	}

	// the block is not enough, resize it and allocate it.
	return cd.addBlock() << 8
}

// For the case where multiple free slots are needed.
func (cd *Cedar) findPlaces(child []byte) int {
	idx := cd.blocksHeadOpen
	// still have available 'Open' blocks.
	if idx != 0 {
		e := cd.listIdx(idx, child)
		if e > 0 {
			return e
		}
	}

	return cd.addBlock() << 8
}

func (cd *Cedar) listIdx(idx int32, child []byte) int {
	n := int32(len(child))
	bo := cd.blocks[cd.blocksHeadOpen].prev

	// only proceed if the free slots are more than the number of children. Also, we
	// save the minimal number of attempts to fail in the `reject`, it only worths to
	// try out this block if the number of children is less than that number.
	for {
		b := &cd.blocks[idx]
		if b.num >= n && n < b.reject {
			e := cd.listEHead(b, child)
			if e > 0 {
				return e
			}
		}

		// we broke out of the loop, that means we failed. We save the information in
		// `reject` for future pruning.
		b.reject = n
		if b.reject < cd.reject[b.num] {
			// put this stats into the global array of information as well.
			cd.reject[b.num] = b.reject
		}

		idxN := b.next
		b.trial++
		// move this block to the 'Closed' block list since it has reached the max_trial
		if b.trial == cd.maxTrial {
			cd.transferBlock(idx, &cd.blocksHeadOpen, &cd.blocksHeadClosed)
		}

		// we have finsihed one round of this cyclic doubly-linked-list.
		if idx == bo {
			break
		}
		// going to the next in this linked list group
		idx = idxN
	}

	return 0
}

func (cd *Cedar) listEHead(b *Block, child []byte) int {
	arr := cd.array
	last := len(child) - 1
	for e := int(b.eHead); ; {
		base := e ^ int(child[0])
		// iterate through the children to see if they are available: (check < 0)
		for i := 0; arr[base^int(child[i])].check < 0; i++ {
			if i == last {
				// we have found the available block.
				b.eHead = int32(e)
				return e
			}
		}

		// save the next free block's information in `check`
		e = int(-arr[e].check)
		if int32(e) == b.eHead {
			break
		}
	}

	return 0
}

// resolve the conflict by moving one of the the nodes to a free block.
// resolve conflict on base_n ^ label_n = base_p ^ label_p
func (cd *Cedar) resolve(fromN, baseN int, labelN byte) int {
	toPn := baseN ^ int(labelN)

	// the `base` and `from` for the conflicting one.
	fromP := int(cd.array[toPn].check)
	baseP := cd.array[fromP].base(cd.Reduced)

	// whether to replace siblings of newly added
	flag := cd.consult(
		baseN, baseP,
		cd.nInfos[fromN].child,
		cd.nInfos[fromP].child,
	)

	// collect the list of children for the block that we are going to relocate.
	var children []byte
	if flag {
		children = cd.setChild(baseN, cd.nInfos[fromN].child, labelN, true)
	} else {
		children = cd.setChild(baseP, cd.nInfos[fromP].child, 255, false)
	}

	// decide which algorithm to allocate free block depending on the number of children
	// we have.
	base := 0
	if len(children) == 1 {
		base = cd.findPlace()
	} else {
		base = cd.findPlaces(children)
	}
	base ^= int(children[0])

	var from, nbase int
	if flag {
		from = fromN
		nbase = baseN
	} else {
		from = fromP
		nbase = baseP
	}

	if flag && children[0] == labelN {
		cd.nInfos[from].child = labelN
	}

	// #[cfg(feature != "reduced-trie")]
	if !cd.Reduced {
		cd.array[from].baseV = int32(base)
	} else {
		cd.array[from].baseV = int32(-base - 1)
	}

	base, labelN, toPn = cd.listN(base, from, nbase, fromN, toPn,
		labelN, children, flag)

	// return the position that is free now.
	if flag {
		return base ^ int(labelN)
	}

	return toPn
}

func (cd *Cedar) listN(base, from, nbase, fromN, toPn int,
	labelN byte, children []byte, flag bool) (int, byte, int) {
	// the actual work for relocating the chilren
	for i := 0; i < len(children); i++ {
		to := cd.popENode(base, from, children[i])
		newTo := nbase ^ int(children[i])

		if i == len(children)-1 {
			cd.nInfos[to].sibling = 0
		} else {
			cd.nInfos[to].sibling = children[i+1]
		}

		// new node has no children
		if flag && newTo == toPn {
			continue
		}

		arr := &cd.array[to]
		arrs := &cd.array[newTo]
		arr.baseV = arrs.baseV

		condition := false
		if !cd.Reduced {
			condition = arr.baseV > 0 && children[i] != 0
		} else {
			condition = arr.baseV < 0 && children[i] != 0
		}

		if condition {
			// this node has children, fix their check
			ab := arr.base(cd.Reduced)
			c := cd.nInfos[newTo].child
			cd.nInfos[to].child = c
			cd.array[ab^int(c)].check = int32(to)

			c = cd.nInfos[ab^int(c)].sibling
			for c != 0 {
				cd.array[ab^int(c)].check = int32(to)
				c = cd.nInfos[ab^int(c)].sibling
			}
		}

		// the parent node is moved
		if !flag && newTo == fromN {
			fromN = to
		}

		// clean up the space that was moved away from.
		if !flag && newTo == toPn {
			cd.pushSibling(fromN, toPn^int(labelN), labelN, true)
			cd.nInfos[newTo].child = 0

			if !cd.Reduced {
				if labelN != 0 {
					arrs.baseV = -1
				} else {
					arrs.baseV = 0
				}
			} else {
				arrs.baseV = ValLimit
			}
			arrs.check = int32(fromN)
		} else {
			cd.pushENode(newTo)
		}
	}

	return base, labelN, toPn
}

// pop a block at idx from the linked-list of type `from`, specially handled if it is the last
// one in the linked-list.
func (cd *Cedar) popBlock(idx int32, from *int32, last bool) {
	if last {
		*from = 0
		return
	}

	b := &cd.blocks[idx]
	cd.blocks[b.prev].next = b.next
	cd.blocks[b.next].prev = b.prev
	if idx == *from {
		*from = b.next
	}
}

// return the block at idx to the linked-list of `to`, specially handled
// if the linked-list is empty
func (cd *Cedar) pushBlock(idx int32, to *int32, empty bool) {
	b := &cd.blocks[idx]
	if empty {
		*to, b.prev, b.next = idx, idx, idx
		return
	}

	tailTo := &cd.blocks[*to].prev
	b.prev = *tailTo
	b.next = *to
	*to, *tailTo, cd.blocks[*tailTo].next = idx, idx, idx
}

// Reallocate more spaces so that we have more free blocks.
func (cd *Cedar) addBlock() int {
	if cd.size == cd.capacity {
		cd.capacity += cd.capacity

		array := cd.array
		cd.array = make([]Node, cd.capacity)
		copy(cd.array, array)

		nInfos := cd.nInfos
		cd.nInfos = make([]NInfo, cd.capacity)
		copy(cd.nInfos, nInfos)

		blocks := cd.blocks
		cd.blocks = make([]Block, cd.capacity>>8)
		copy(cd.blocks, blocks)
	}

	idx := int32(cd.size >> 8)
	cd.blocks[idx].init()
	cd.blocks[idx].eHead = int32(cd.size)

	// make it a doubley linked list
	size := int32(cd.size)
	cd.array[size] = Node{baseV: -(size + 255), check: -(size + 1)}
	for i := size + 1; i < size+255; i++ {
		cd.array[i] = Node{baseV: -(i - 1), check: -(i + 1)}
	}
	cd.array[size+255] = Node{baseV: -(size + 254), check: -size}

	// append to block Open
	cd.pushBlock(idx, &cd.blocksHeadOpen, cd.blocksHeadOpen == 0)
	cd.size += 256
	return int(idx)
}

// transfer the block at idx from the linked-list of `from` to the linked-list of `to`,
// specially handle the case where the destination linked-list is empty.
func (cd *Cedar) transferBlock(idx int32, from, to *int32) {
	b := &cd.blocks[idx]
	last, num := idx == b.next, b.num // b.next it's the last one if the next points to itself
	cd.popBlock(idx, from, last)
	cd.pushBlock(idx, to, *to == 0 && num != 0)
}
