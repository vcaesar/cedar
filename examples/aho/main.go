// Copyright 2016 Evans. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/vcaesar/cedar/aho"
)

func main() {
	// Build a matcher; NewStrings uses the pattern index as its value.
	m := aho.NewStrings("he", "she", "his", "hers")

	// Or insert patterns with your own values, the automaton is
	// (re)compiled lazily on the next Match.
	if err := m.Insert([]byte("太阳系"), 100); err != nil {
		log.Fatal(err)
	}

	text := []byte("ushers 太阳系")
	fmt.Println("has match:", m.Has(text))
	for _, t := range m.Match(text) {
		fmt.Printf("value=%d at=%d len=%d key=%q\n", t.Value, t.At, t.Len, m.Key(text, t))
	}

	// Patterns can be removed; the automaton recompiles on the next Match.
	if err := m.Delete([]byte("he")); err != nil {
		log.Fatal(err)
	}
	fmt.Println("after delete:", m.MatchString("ushers"))

	// Persist the trie as gob or json and load it back.
	dir, err := os.MkdirTemp("", "cedar-aho")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(dir)

	file := filepath.Join(dir, "patterns.json")
	if err := m.SaveToFile(file, "json"); err != nil {
		log.Fatal(err)
	}

	loaded := aho.New()
	if err := loaded.LoadFromFile(file, "json"); err != nil {
		log.Fatal(err)
	}
	fmt.Println("loaded:", loaded.MatchString("ushers"))

	// Stream the node ids of every pattern that starts with "h".
	for id := range loaded.PrefixPredict([]byte("h"), 0, 4) {
		v, err := loaded.Cedar().Value(id)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("prefix h -> node", id, "value", v)
	}

	// Dump the automaton for Graphviz: dot -Tsvg trie.gv -o trie.svg
	if err := loaded.DumpGraph("trie.gv"); err != nil {
		log.Fatal(err)
	}
	fmt.Println("wrote trie.gv")
}
