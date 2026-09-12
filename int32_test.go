// Copyright 2016 Evans. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cedar

import (
	"fmt"
	"math"
	"testing"
	"unsafe"

	"github.com/vcaesar/tt"
)

func TestNodeSize(t *testing.T) {
	tt.Equal(t, 8, unsafe.Sizeof(Node{}))
	tt.Equal(t, 24, unsafe.Sizeof(Block{}))
	tt.Equal(t, math.MaxInt32, ValLimit)
}

// Jump is specialised per trie mode; make sure both branches agree with the
// generic semantics (empty key -> 0, missing prefix -> ErrNoKey, cursor reuse).
func TestJumpModes(t *testing.T) {
	for _, reduced := range []bool{true, false} {
		cd := New(reduced)
		tt.Nil(t, cd.Insert([]byte("abc"), 1))
		tt.Nil(t, cd.Insert([]byte("abd"), 2))

		to, err := cd.Jump(nil, 0)
		tt.Nil(t, err)
		tt.Equal(t, 0, to)

		ab, err := cd.Jump([]byte("ab"), 0)
		tt.Nil(t, err)
		tt.Bool(t, ab > 0)

		abc, err := cd.Jump([]byte("c"), ab)
		tt.Nil(t, err)
		v, err := cd.Value(abc)
		tt.Nil(t, err)
		tt.Equal(t, 1, v)

		from, err := cd.Jump([]byte("x"), ab)
		tt.Equal(t, ErrNoKey, err)
		tt.Equal(t, ab, from)

		_, err = cd.Jump([]byte("abcd"), 0)
		tt.Equal(t, ErrNoKey, err)
	}
}

// resolve() reuses a single scratch buffer for the children list; drive many
// relocations of wide sibling sets (all 255 labels + terminal) and make sure
// nothing is lost or corrupted. Label 0 is the terminal marker so it is not a
// valid key byte.
func TestWideSiblingRelocation(t *testing.T) {
	for _, reduced := range []bool{true, false} {
		cd := New(reduced)
		var keys [][]byte
		for p := 1; p <= 8; p++ {
			for c := 1; c < 256; c++ {
				keys = append(keys, []byte{byte(p), byte(c)})
				keys = append(keys, []byte{byte(p), byte(c), byte(p ^ c | 1)})
			}
		}
		for i, k := range keys {
			tt.Nil(t, cd.Insert(k, i))
		}
		for i, k := range keys {
			v, err := cd.Get(k)
			tt.Nil(t, err)
			tt.Equal(t, i, v)
		}
	}
}

func TestValLimit(t *testing.T) {
	for _, reduced := range []bool{true, false} {
		cd := New(reduced)
		tt.Equal(t, ErrInvalidVal, cd.Insert([]byte("a"), ValLimit))
		tt.Equal(t, ErrInvalidVal, cd.Insert([]byte("a"), -1))

		tt.Nil(t, cd.Insert([]byte("a"), ValLimit-1))
		v, err := cd.Get([]byte("a"))
		tt.Nil(t, err)
		tt.Equal(t, ValLimit-1, v)

		// Update must not overflow the int32 storage
		tt.Equal(t, ErrInvalidVal, cd.Update([]byte("a"), 1))
		tt.Equal(t, ErrInvalidVal, cd.Update([]byte("a"), -ValLimit))
		tt.Equal(t, ErrInvalidVal, cd.Update([]byte("b"), 1<<40))
		v, err = cd.Get([]byte("a"))
		tt.Nil(t, err)
		tt.Equal(t, ValLimit-1, v)

		tt.Nil(t, cd.Update([]byte("a"), -1))
		v, err = cd.Get([]byte("a"))
		tt.Nil(t, err)
		tt.Equal(t, ValLimit-2, v)
	}
}

// insert enough keys to grow the array over many blocks and force
// node relocations, then check every key and value survives
func TestManyKeys(t *testing.T) {
	const n = 200000
	for _, reduced := range []bool{true, false} {
		cd := New(reduced)
		for i := 0; i < n; i++ {
			tt.Nil(t, cd.Insert([]byte(fmt.Sprintf("键%d/k%x", i, i*7)), i))
		}
		tt.Bool(t, cd.size > 256*1024)

		for i := 0; i < n; i++ {
			v, err := cd.Get([]byte(fmt.Sprintf("键%d/k%x", i, i*7)))
			tt.Nil(t, err)
			tt.Equal(t, i, v)
		}

		_, err := cd.Get([]byte("键"))
		tt.Equal(t, ErrNoVal, err)
		_, err = cd.Get([]byte("missing"))
		tt.Equal(t, ErrNoKey, err)

		for i := 0; i < n; i += 3 {
			tt.Nil(t, cd.Delete([]byte(fmt.Sprintf("键%d/k%x", i, i*7))))
		}
		for i := 0; i < n; i++ {
			v, err := cd.Get([]byte(fmt.Sprintf("键%d/k%x", i, i*7)))
			if i%3 == 0 {
				tt.NotNil(t, err)
				continue
			}
			tt.Nil(t, err)
			tt.Equal(t, i, v)
		}
	}
}
