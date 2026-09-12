// Copyright 2016 Evans. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package aho implements Aho-Corasick multi-pattern matching on top of the
// cedar double-array trie.
package aho

import (
	"bytes"

	"github.com/vcaesar/cedar"
)

// Matcher is an Aho-Corasick automaton built over a cedar trie.
//
// Patterns are added with Insert; the failure and output links are (re)built
// lazily by the next Match, or explicitly with Compile. Patterns must not
// contain a NUL byte, cedar uses it as the key terminator.
type Matcher struct {
	da       *cedar.Cedar
	fails    []int32 // failure link of every node
	outputs  []int32 // nearest terminal node on the failure chain, -1 if none
	depth    []int32 // key length of every node
	compiled bool
}

// Token is one pattern occurrence found in the text.
type Token struct {
	Value int // value stored with the pattern
	At    int // start offset in the text
	Len   int // pattern length in bytes
}

// New returns an empty Matcher; `reduced` is forwarded to cedar.New.
func New(reduced ...bool) *Matcher {
	return &Matcher{da: cedar.New(reduced...)}
}

// NewStrings returns a Matcher holding every pattern with its index as value.
func NewStrings(patterns ...string) *Matcher {
	m := New()
	for i, p := range patterns {
		if err := m.Insert([]byte(p), i); err != nil {
			panic("aho: " + err.Error())
		}
	}

	return m
}

// Cedar returns the underlying trie.
func (m *Matcher) Cedar() *cedar.Cedar {
	return m.da
}

// Insert adds a pattern with its value.
func (m *Matcher) Insert(key []byte, val int) error {
	if len(key) == 0 || bytes.IndexByte(key, 0) >= 0 {
		return cedar.ErrInvalidKey
	}
	if err := m.da.Insert(key, val); err != nil {
		return err
	}

	m.compiled = false
	return nil
}

// Delete removes a pattern.
func (m *Matcher) Delete(key []byte) error {
	if err := m.da.Delete(key); err != nil {
		return err
	}

	m.compiled = false
	return nil
}

// Compile builds the failure and output links with a breadth-first walk of
// the trie; every node is visited after its failure target, so the output
// link of the target is already final.
func (m *Matcher) Compile() {
	da := m.da
	n := da.Size()
	m.fails = make([]int32, n)
	m.outputs = make([]int32, n)
	m.depth = make([]int32, n)
	for i := range m.outputs {
		m.outputs[i] = -1
	}

	var labels []byte
	queue := []int32{0}
	for len(queue) > 0 {
		nid := int(queue[0])
		queue = queue[1:]

		labels = da.Children(nid, labels[:0])
		for i := range labels {
			label := labels[i : i+1]
			child, err := da.Jump(label, nid)
			if err != nil {
				panic("aho: cedar child not reachable")
			}
			queue = append(queue, int32(child))
			m.depth[child] = m.depth[nid] + 1

			fail := 0
			if nid != 0 {
				for f := int(m.fails[nid]); ; f = int(m.fails[f]) {
					if to, err := da.Jump(label, f); err == nil {
						fail = to
						break
					}
					if f == 0 {
						break
					}
				}
			}
			m.fails[child] = int32(fail)

			if m.isEnd(fail) {
				m.outputs[child] = int32(fail)
			} else {
				m.outputs[child] = m.outputs[fail]
			}
		}
	}

	m.compiled = true
}

func (m *Matcher) isEnd(nid int) bool {
	if nid == 0 {
		return false
	}
	_, err := m.da.Value(nid)
	return err == nil
}

// Match returns every pattern occurrence in text, ordered by end offset.
func (m *Matcher) Match(text []byte) []Token {
	var tokens []Token
	m.scan(text, func(t Token) bool {
		tokens = append(tokens, t)
		return true
	})

	return tokens
}

// MatchString is Match for a string.
func (m *Matcher) MatchString(text string) []Token {
	return m.Match([]byte(text))
}

// Has reports whether any pattern occurs in text.
func (m *Matcher) Has(text []byte) bool {
	found := false
	m.scan(text, func(Token) bool {
		found = true
		return false
	})

	return found
}

// Key returns the bytes of text matched by the token.
func (m *Matcher) Key(text []byte, t Token) []byte {
	return text[t.At : t.At+t.Len]
}

// scan runs the automaton over text and calls emit for every occurrence
// until it returns false.
func (m *Matcher) scan(text []byte, emit func(Token) bool) {
	if !m.compiled {
		m.Compile()
	}

	da := m.da
	nid := 0
	for i := range text {
		// no pattern contains NUL, so it cannot be part of a match
		if text[i] == 0 {
			nid = 0
			continue
		}

		label := text[i : i+1]
		for {
			if to, err := da.Jump(label, nid); err == nil {
				nid = to
				break
			}
			if nid == 0 {
				break
			}
			nid = int(m.fails[nid])
		}

		out := nid
		if !m.isEnd(out) {
			out = int(m.outputs[out])
		}
		for ; out >= 0; out = int(m.outputs[out]) {
			val, err := da.Value(out)
			if err != nil {
				panic("aho: output node has no value")
			}
			n := int(m.depth[out])
			if !emit(Token{Value: val, At: i + 1 - n, Len: n}) {
				return
			}
		}
	}
}
