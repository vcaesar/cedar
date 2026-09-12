// Copyright 2016 Evans. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package aho

import (
	"bufio"
	"encoding/gob"
	"encoding/json"
	"errors"
	"io"
	"os"
)

// ErrDataType is returned by Save/Load for an unsupported data type.
var ErrDataType = errors.New("aho: unsupported data type")

// Save writes the matcher's trie to out; dataType is "gob" or "json".
// The failure links are rebuilt on the next Match after Load.
func (m *Matcher) Save(out io.Writer, dataType string) error {
	switch dataType {
	case "gob", "GOB":
		return gob.NewEncoder(out).Encode(m.da)
	case "json", "JSON":
		return json.NewEncoder(out).Encode(m.da)
	}

	return ErrDataType
}

// SaveToFile writes the matcher's trie to a file.
func (m *Matcher) SaveToFile(fileName, dataType string) error {
	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

	out := bufio.NewWriter(file)
	if err := m.Save(out, dataType); err != nil {
		return err
	}

	return out.Flush()
}

// Load replaces the matcher's trie with one written by Save.
func (m *Matcher) Load(in io.Reader, dataType string) error {
	var err error
	switch dataType {
	case "gob", "GOB":
		err = gob.NewDecoder(in).Decode(m.da)
	case "json", "JSON":
		err = json.NewDecoder(in).Decode(m.da)
	default:
		return ErrDataType
	}
	if err != nil {
		return err
	}

	m.compiled = false
	return nil
}

// LoadFromFile replaces the matcher's trie with one written by SaveToFile.
func (m *Matcher) LoadFromFile(fileName, dataType string) error {
	file, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	return m.Load(bufio.NewReader(file), dataType)
}

// PrefixPredict streams the node ids of the patterns that have key as their
// prefix, at most num (0 for all), through a channel of the given size.
func (m *Matcher) PrefixPredict(key []byte, num, channelSize int) chan int {
	ret := make(chan int, channelSize)
	go func() {
		defer close(ret)
		for _, id := range m.da.PrefixPredict(key, num) {
			ret <- id
		}
	}()

	return ret
}
