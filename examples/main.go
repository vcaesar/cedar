package main

import (
	"fmt"

	"github.com/vcaesar/cedar"
)

func main() {
	// Create a new cedar trie.
	d := cedar.New()
	d.Insert([]byte("ab"), 1)
	d.Insert([]byte("abc"), 2)
	d.Insert([]byte("abcd"), 3)

	fmt.Println(d.Jump([]byte("ab"), 0))
	fmt.Println(d.Jump([]byte("bc"), 0))

	fmt.Println(d.PrefixMatch([]byte("bc"), 0))
	fmt.Println(d.ExactMatch([]byte("ab")))
}
