# cedar

[![Build Status](https://github.com/vcaesar/cedar/actions/workflows/go.yml/badge.svg)](https://github.com/vcaesar/cedar/actions/workflows/go.yml)
[![CircleCI Status](https://dl.circleci.com/status-badge/img/gh/vcaesar/cedar/tree/main.svg?style=shield)](https://dl.circleci.com/status-badge/redirect/gh/vcaesar/cedar/tree/main)
[![codecov](https://codecov.io/gh/vcaesar/cedar/branch/main/graph/badge.svg)](https://codecov.io/gh/vcaesar/cedar)
[![Go Report Card](https://goreportcard.com/badge/github.com/vcaesar/cedar)](https://goreportcard.com/report/github.com/vcaesar/cedar)
[![Go Reference](https://pkg.go.dev/badge/github.com/vcaesar/cedar.svg)](https://pkg.go.dev/github.com/vcaesar/cedar)
[![Release](https://img.shields.io/github/v/release/vcaesar/cedar)](https://github.com/vcaesar/cedar/releases/latest)

<!-- [![Join the chat at https://gitter.im/go-ego/ego](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/go-ego/ego?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge) -->

Package `cedar` implements an updatable double-array trie and aho corasick.

It is a Go port of [cedar](http://www.tkl.iis.u-tokyo.ac.jp/~ynaga/cedar) (see the [paper](http://www.tkl.iis.u-tokyo.ac.jp/~ynaga/papers/ynaga-coling2014.pdf)).

## Install

```
go get github.com/vcaesar/cedar
```

## Usage

```go
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
	fmt.Println(d.Find([]byte("bc"), 0))

	fmt.Println(d.PrefixMatch([]byte("bc"), 0))
	fmt.Println(d.ExactMatch([]byte("ab")))
}
```

### Aho-Corasick

Package `aho` builds an Aho-Corasick automaton on top of the trie for
multi-pattern search. Patterns can be inserted and deleted at any time; the
failure links are rebuilt lazily on the next `Match`.

```go
package main

import (
	"fmt"

	"github.com/vcaesar/cedar/aho"
)

func main() {
	// NewStrings uses the pattern index as its value,
	// or Insert patterns with your own values.
	m := aho.NewStrings("he", "she", "his", "hers")
	m.Insert([]byte("太阳系"), 100)

	text := []byte("ushers 太阳系")
	fmt.Println(m.Has(text))
	for _, t := range m.Match(text) {
		fmt.Printf("value=%d at=%d len=%d key=%q\n", t.Value, t.At, t.Len, m.Key(text, t))
	}

	// persist the trie as "gob" or "json"
	m.SaveToFile("patterns.json", "json")
	loaded := aho.New()
	loaded.LoadFromFile("patterns.json", "json")

	// stream the node ids of the patterns starting with "h"
	for id := range loaded.PrefixPredict([]byte("h"), 0, 4) {
		fmt.Println(loaded.Cedar().Value(id))
	}

	// visualise: dot -Tsvg trie.gv -o trie.svg
	loaded.DumpGraph("trie.gv")
}
```

Output:

```
true
value=1 at=1 len=3 key="she"
value=0 at=2 len=2 key="he"
value=3 at=2 len=4 key="hers"
value=100 at=7 len=9 key="太阳系"
```

See [examples/aho](examples/aho/main.go) for the full demo (`go run ./examples/aho`).

## License

This is released under the BSD-2 license, following the original license of C++ cedar.

## Reference

- [cedar - C++ implementation of efficiently-updatable double-array trie](http://www.tkl.iis.u-tokyo.ac.jp/~ynaga/cedar/), and thanks for [cedarwood](https://github.com/MnO2/cedarwood).
