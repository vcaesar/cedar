// Copyright 2016 Evans. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cedar

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/vcaesar/tt"
)

// codecs under test: each encodes src and decodes into dst
var codecs = map[string]func(src, dst *Cedar) error{
	"gob": func(src, dst *Cedar) error {
		var buf bytes.Buffer
		if err := gob.NewEncoder(&buf).Encode(src); err != nil {
			return err
		}
		return gob.NewDecoder(&buf).Decode(dst)
	},
	"json": func(src, dst *Cedar) error {
		b, err := json.Marshal(src)
		if err != nil {
			return err
		}
		return json.Unmarshal(b, dst)
	},
}

func TestCodecRoundTrip(t *testing.T) {
	for name, codec := range codecs {
		for _, reduced := range []bool{true, false} {
			cd := New(reduced)
			for i := 0; i < 3000; i++ {
				tt.Nil(t, cd.Insert([]byte(fmt.Sprintf("key-%d-太阳系", i)), i))
			}
			for i := 0; i < 3000; i += 3 {
				tt.Nil(t, cd.Delete([]byte(fmt.Sprintf("key-%d-太阳系", i))))
			}

			got := New(!reduced) // must be fully overwritten
			tt.Nil(t, codec(cd, got), name)
			// childBuf is scratch and not persisted; compare everything else
			got.childBuf = cd.childBuf
			tt.Bool(t, reflect.DeepEqual(cd, got), name)

			for i := 0; i < 3000; i++ {
				v, err := got.Get([]byte(fmt.Sprintf("key-%d-太阳系", i)))
				if i%3 == 0 {
					tt.Equal(t, ErrNoKey, err)
					continue
				}
				tt.Nil(t, err)
				tt.Equal(t, i, v)
			}

			// the restored trie keeps working, including block allocation
			for i := 0; i < 3000; i++ {
				tt.Nil(t, got.Insert([]byte(fmt.Sprintf("new-%d", i)), i))
			}
			v, err := got.Get([]byte("new-2999"))
			tt.Nil(t, err)
			tt.Equal(t, 2999, v)
		}
	}
}

func TestDecodeInvalid(t *testing.T) {
	cd := New()
	tt.NotNil(t, cd.GobDecode([]byte("garbage")))
	tt.NotNil(t, cd.UnmarshalJSON([]byte("garbage")))

	bad := &cedarData{Size: 300, Capacity: 300}
	var buf bytes.Buffer
	tt.Nil(t, gob.NewEncoder(&buf).Encode(bad))
	tt.Equal(t, ErrInvalidData, cd.GobDecode(buf.Bytes()))

	b, err := json.Marshal(bad)
	tt.Nil(t, err)
	tt.Equal(t, ErrInvalidData, cd.UnmarshalJSON(b))
}
