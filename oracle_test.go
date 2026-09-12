// Copyright 2016 Evans. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cedar

import (
	"math/rand"
	"testing"

	"github.com/vcaesar/tt"
)

// Random insert/delete/update workload checked against a map oracle in both
// trie modes; keys never contain 0x00 since that is the terminal label.
func TestRandomOracle(t *testing.T) {
	for _, reduced := range []bool{true, false} {
		rng := rand.New(rand.NewSource(1))
		cd := New(reduced)
		oracle := map[string]int{}
		var keys []string

		randKey := func() string {
			b := make([]byte, 1+rng.Intn(6))
			for i := range b {
				b[i] = byte(1 + rng.Intn(5))
			}
			return string(b)
		}

		for step := 0; step < 100000; step++ {
			switch op := rng.Intn(10); {
			case op < 5:
				k, v := randKey(), rng.Intn(1000)
				tt.Nil(t, cd.Insert([]byte(k), v))
				if _, ok := oracle[k]; !ok {
					keys = append(keys, k)
				}
				oracle[k] = v
			case op < 8 && len(keys) > 0:
				i := rng.Intn(len(keys))
				k := keys[i]
				tt.Nil(t, cd.Delete([]byte(k)))
				delete(oracle, k)
				keys[i] = keys[len(keys)-1]
				keys = keys[:len(keys)-1]
			case len(keys) > 0:
				k, v := keys[rng.Intn(len(keys))], rng.Intn(10)
				tt.Nil(t, cd.Update([]byte(k), v))
				oracle[k] += v
			}

			if step%997 == 0 {
				for k, want := range oracle {
					got, err := cd.Get([]byte(k))
					tt.Nil(t, err)
					tt.Equal(t, want, got)
				}
				_, err := cd.Get([]byte(randKey() + "\x07"))
				tt.NotNil(t, err)
			}
		}
		for k, want := range oracle {
			got, err := cd.Get([]byte(k))
			tt.Nil(t, err)
			tt.Equal(t, want, got)
		}
	}
}
