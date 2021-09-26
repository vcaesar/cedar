// Copyright 2016 Evans. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cedar

// NInfo stores the information about the trie
type NInfo struct {
	sibling, child byte // uint8
}

// Node contains the array of `base` and `check` as specified in the paper:
// "An efficient implementation of trie structures"
// https://dl.acm.org/citation.cfm?id=146691
type Node struct {
	baseV, check int // int32
}

func (n *Node) base(reduced ...bool) int {
	if len(reduced) > 0 && !reduced[0] {
		return n.baseV
	}

	return -(n.baseV + 1)
}

// Block stores the linked-list pointers and the stats info for blocks.
type Block struct {
	prev   int // int32   // previous block's index, 3 bytes width
	next   int // next block's index, 3 bytes width
	num    int // the number of slots that is free, the range is 0-256
	reject int // a heuristic number to make the search for free space faster...
	trial  int // the number of times this block has been probed by `find_places` for the free block.
	eHead  int // the index of the first empty elemenet in this block
}

func (b *Block) init() {
	b.num = 256    // each of block has 256 free slots at the beginning
	b.reject = 257 // initially every block need to be fully iterated through so that we can reject it to be unusable.
}

// Cedar holds all of the information about double array trie.
type Cedar struct {
	array  []Node // storing the `base` and `check` info from the original paper.
	nInfos []NInfo
	blocks []Block
	reject [257]int

	blocksHeadFull   int // the index of the first 'Full' block, 0 means no 'Full' block
	blocksHeadClosed int // the index of the first 'Closed' block, 0 means no ' Closed' block
	blocksHeadOpen   int // the index of the first 'Open' block, 0 means no 'Open' block

	capacity int
	size     int
	ordered  bool
	maxTrial int // the parameter for cedar, it could be tuned for more, but the default is 1.
}

const (
	// ValLimit cedar value limit
	ValLimit = int(^uint(0) >> 1)
	NoVal    = -1
)

// type PrefixIter struct {
// }

// New initialize the Cedar for further use
func New(reduced ...bool) *Cedar {
	cd := Cedar{
		array:  make([]Node, 256),
		nInfos: make([]NInfo, 256),
		blocks: make([]Block, 1),

		capacity: 256,
		size:     256,
		ordered:  true,
		maxTrial: 1,
	}

	if !isReduced(reduced...) {
		cd.array[0] = Node{baseV: 0, check: -1}
	} else {
		cd.array[0] = Node{baseV: -1, check: -1}
	}
	// make `baseV` point to the previous element, and make `check` point to the next element
	for i := 1; i < 256; i++ {
		cd.array[i] = Node{baseV: -(i - 1), check: -(i + 1)}
	}
	// make them link as a cyclic doubly-linked list
	cd.array[1].baseV = -255
	cd.array[255].check = -1

	cd.blocks[0].eHead = 1
	cd.blocks[0].init()

	for i := 0; i <= 256; i++ {
		cd.reject[i] = i + 1
	}

	return &cd
}
