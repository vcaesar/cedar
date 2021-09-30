# cedar

[![Build Status](https://github.com/vcaesar/cedar/workflows/Go/badge.svg)](https://github.com/vcaesar/cedar/commits/master)
[![Build Status](https://travis-ci.org/vcaesar/cedar.svg)](https://travis-ci.org/vcaesar/cedar)
[![CircleCI Status](https://circleci.com/gh/vcaesar/cedar.svg?style=shield)](https://circleci.com/gh/vcaesar/cedar)
[![codecov](https://codecov.io/gh/vcaesar/cedar/branch/master/graph/badge.svg)](https://codecov.io/gh/vcaesar/cedar)
[![Go Report Card](https://goreportcard.com/badge/github.com/vcaesar/cedar)](https://goreportcard.com/report/github.com/vcaesar/cedar)
[![GoDoc](https://godoc.org/github.com/vcaesar/cedar?status.svg)](https://godoc.org/github.com/vcaesar/cedar)
[![Release](https://github-release-version.herokuapp.com/github/vcaesar/cedar/release.svg?style=flat)](https://github.com/vcaesar/cedar/releases/latest)
<!-- [![Join the chat at https://gitter.im/go-ego/ego](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/go-ego/ego?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge) -->

Package `cedar` implementes double-array trie and aho corasick

It is implements the [cedar](http://www.tkl.iis.u-tokyo.ac.jp/~ynaga/cedar) and [paper](http://www.tkl.iis.u-tokyo.ac.jp/~ynaga/papers/ynaga-coling2014.pdf) by golang.

## Install
```
go get -u github.com/vcaesar/cedar
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
	// fmt.Println(d.PrefixMatch([]byte("bc"), 0))
}
```

## License

This is released under the BSD-2 license, following the original license of C++ cedar.

## Reference

* [cedar - C++ implementation of efficiently-updatable double-array trie](http://www.tkl.iis.u-tokyo.ac.jp/~ynaga/cedar/), and thanks for [cedarwood](https://github.com/MnO2/cedarwood).