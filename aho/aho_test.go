// Copyright 2016 Evans. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package aho

import (
	"bytes"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/vcaesar/cedar"
	"github.com/vcaesar/tt"
)

func sortTokens(ts []Token) {
	sort.Slice(ts, func(i, j int) bool {
		if ts[i].At != ts[j].At {
			return ts[i].At < ts[j].At
		}
		return ts[i].Len < ts[j].Len
	})
}

// naive reference: every occurrence of every pattern
func naive(patterns map[string]int, text []byte) []Token {
	var ts []Token
	for p, v := range patterns {
		for i := 0; i+len(p) <= len(text); i++ {
			if string(text[i:i+len(p)]) == p {
				ts = append(ts, Token{Value: v, At: i, Len: len(p)})
			}
		}
	}
	sortTokens(ts)
	return ts
}

func TestClassic(t *testing.T) {
	for _, reduced := range []bool{true, false} {
		m := New(reduced)
		for i, p := range []string{"he", "she", "his", "hers"} {
			tt.Nil(t, m.Insert([]byte(p), i))
		}

		text := []byte("ushers")
		got := m.Match(text)
		tt.Equal(t, []Token{{1, 1, 3}, {0, 2, 2}, {3, 2, 4}}, got)
		tt.Equal(t, "she", string(m.Key(text, got[0])))
		tt.Equal(t, "hers", string(m.Key(text, got[2])))

		tt.Bool(t, m.Has(text))
		tt.Bool(t, !m.Has([]byte("nothing")))
		tt.Equal(t, 0, len(m.MatchString("")))
	}
}

func TestNewStrings(t *testing.T) {
	m := NewStrings("太阳", "太阳系", "地球")
	got := m.MatchString("太阳系地球")
	tt.Equal(t, []Token{{0, 0, 6}, {1, 0, 9}, {2, 9, 6}}, got)
	tt.NotNil(t, m.Cedar())
}

func TestInvalidPattern(t *testing.T) {
	m := New()
	tt.Equal(t, cedar.ErrInvalidKey, m.Insert(nil, 1))
	tt.Equal(t, cedar.ErrInvalidKey, m.Insert([]byte("a\x00b"), 1))
	tt.Equal(t, cedar.ErrInvalidVal, m.Insert([]byte("ab"), -1))
	tt.Equal(t, cedar.ErrNoKey, m.Delete([]byte("ab")))
}

// NUL in the text can never be inside a match and must reset the automaton.
func TestNulInText(t *testing.T) {
	for _, reduced := range []bool{true, false} {
		m := New(reduced)
		tt.Nil(t, m.Insert([]byte("ab"), 1))
		tt.Nil(t, m.Insert([]byte("abc"), 2))

		got := m.Match([]byte("ab\x00abc\x00"))
		tt.Equal(t, []Token{{1, 0, 2}, {1, 3, 2}, {2, 3, 3}}, got)
	}
}

func TestRecompile(t *testing.T) {
	for _, reduced := range []bool{true, false} {
		m := New(reduced)
		tt.Nil(t, m.Insert([]byte("ab"), 1))
		tt.Equal(t, 1, len(m.Match([]byte("xabx"))))

		tt.Nil(t, m.Insert([]byte("b"), 2))
		tt.Equal(t, []Token{{1, 1, 2}, {2, 2, 1}}, m.Match([]byte("xabx")))

		tt.Nil(t, m.Delete([]byte("ab")))
		tt.Equal(t, []Token{{2, 2, 1}}, m.Match([]byte("xabx")))

		m.Compile()
		tt.Bool(t, m.compiled)
	}
}

func TestEmpty(t *testing.T) {
	m := New()
	tt.Equal(t, 0, len(m.Match([]byte("abc"))))
	tt.Bool(t, !m.Has([]byte("abc")))
}

func TestRandomOracle(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	alphabet := []byte("abc")

	for _, reduced := range []bool{true, false} {
		for round := 0; round < 30; round++ {
			m := New(reduced)
			patterns := map[string]int{}
			for i := 0; i < 40; i++ {
				n := 1 + rng.Intn(5)
				p := make([]byte, n)
				for j := range p {
					p[j] = alphabet[rng.Intn(len(alphabet))]
				}
				patterns[string(p)] = i
				tt.Nil(t, m.Insert(p, i))
			}

			text := make([]byte, 200)
			for j := range text {
				text[j] = alphabet[rng.Intn(len(alphabet))]
			}

			got := m.Match(text)
			sortTokens(got)
			tt.Equal(t, naive(patterns, text), got)
		}
	}
}

func BenchmarkMatch(b *testing.B) {
	m := NewStrings("he", "she", "his", "hers", "cedar", "trie", "aho")
	text := bytes.Repeat([]byte("ushers use the cedar trie aho automaton "), 64)
	m.Compile()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Match(text)
	}
}

func TestSaveLoad(t *testing.T) {
	for _, dataType := range []string{"gob", "JSON"} {
		for _, reduced := range []bool{true, false} {
			m := New(reduced)
			for i, p := range []string{"he", "she", "his", "hers"} {
				tt.Nil(t, m.Insert([]byte(p), i))
			}
			want := m.MatchString("ushers")

			var buf bytes.Buffer
			tt.Nil(t, m.Save(&buf, dataType))
			tt.Equal(t, ErrDataType, m.Save(&buf, "xml"))

			got := New(!reduced)
			tt.Nil(t, got.Load(&buf, dataType))
			tt.Equal(t, ErrDataType, got.Load(&buf, "xml"))
			tt.Equal(t, want, got.MatchString("ushers"))

			// keeps working as an updatable trie after Load
			tt.Nil(t, got.Insert([]byte("us"), 9))
			tt.Equal(t, Token{9, 0, 2}, got.MatchString("ushers")[0])
		}
	}

	// a bad payload is reported and does not leave a half-loaded matcher
	m := NewStrings("he")
	tt.NotNil(t, m.Load(bytes.NewBufferString("{}"), "json"))
	tt.Equal(t, 1, len(m.MatchString("he")))
}

func TestSaveLoadFile(t *testing.T) {
	for _, dataType := range []string{"gob", "json"} {
		m := NewStrings("he", "she")
		name := filepath.Join(t.TempDir(), "aho."+dataType)
		tt.Nil(t, m.SaveToFile(name, dataType))

		got := New()
		tt.Nil(t, got.LoadFromFile(name, dataType))
		tt.Equal(t, m.MatchString("ushers"), got.MatchString("ushers"))

		tt.NotNil(t, got.LoadFromFile(filepath.Join(t.TempDir(), "missing"), dataType))
		tt.NotNil(t, m.SaveToFile(filepath.Join(t.TempDir(), "no", "dir"), dataType))
	}
}

func TestPrefixPredict(t *testing.T) {
	m := NewStrings("he", "hers", "his", "she")
	var ids []int
	for id := range m.PrefixPredict([]byte("h"), 0, 2) {
		ids = append(ids, id)
	}
	tt.Equal(t, m.Cedar().PrefixPredict([]byte("h")), ids)
	tt.Equal(t, 3, len(ids))

	n := 0
	for range m.PrefixPredict([]byte("h"), 2, 0) {
		n++
	}
	tt.Equal(t, 2, n)
}

func TestWriteGraph(t *testing.T) {
	patterns := []string{"he", "she", "his", "hers"}
	m := NewStrings(patterns...)
	tt.Nil(t, m.Insert([]byte{'a', '"', '\\', 0xff}, 7))

	var buf bytes.Buffer
	tt.Nil(t, m.WriteGraph(&buf))
	g := buf.String()
	tt.Bool(t, strings.HasPrefix(g, "digraph cedar {\n"))
	tt.Bool(t, strings.HasSuffix(g, "}\n"))

	cd := m.Cedar()
	for i, p := range patterns {
		nid, err := cd.Jump([]byte(p), 0)
		tt.Nil(t, err)
		tt.Bool(t, strings.Contains(g, fmt.Sprintf("\t%d [shape=doublecircle label=\"%d\\n=%d\"];\n", nid, nid, i)), p)
	}

	// classic failure link she -> he; links to root are not drawn
	she, _ := cd.Jump([]byte("she"), 0)
	he, _ := cd.Jump([]byte("he"), 0)
	tt.Bool(t, strings.Contains(g, fmt.Sprintf("\t%d -> %d [style=dashed", she, he)))
	tt.Bool(t, !strings.Contains(g, "-> 0 [style=dashed"))

	// one solid edge per trie node: h,he,her,hers,hi,his,s,sh,she + the 4 bytes of the escaped key
	tt.Equal(t, 13, strings.Count(g, "[label=\"")-1) // minus the root label

	// label escaping
	tt.Bool(t, strings.Contains(g, `[label="\""]`))
	tt.Bool(t, strings.Contains(g, `[label="\\"]`))
	tt.Bool(t, strings.Contains(g, `[label="0xFF"]`))
}

func TestDumpGraph(t *testing.T) {
	m := NewStrings("he", "she")
	name := filepath.Join(t.TempDir(), "trie.gv")
	tt.Nil(t, m.DumpGraph(name))

	var want bytes.Buffer
	tt.Nil(t, m.WriteGraph(&want))
	got, err := os.ReadFile(name)
	tt.Nil(t, err)
	tt.Equal(t, want.String(), string(got))

	tt.NotNil(t, m.DumpGraph(filepath.Join(t.TempDir(), "no", "trie.gv")))
}
