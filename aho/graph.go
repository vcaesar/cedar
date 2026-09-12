// Copyright 2016 Evans. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package aho

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

// DumpGraph writes the automaton to fname in Graphviz DOT format,
// e.g. `dot -Tsvg trie.gv -o trie.svg`.
func (m *Matcher) DumpGraph(fname string) error {
	file, err := os.Create(fname)
	if err != nil {
		return err
	}
	defer file.Close()

	return m.WriteGraph(file)
}

// WriteGraph writes the automaton in Graphviz DOT format: solid edges are
// trie transitions labelled with their byte, terminal nodes are double
// circles annotated with their value, dashed red edges are failure links
// (links to the root are omitted).
func (m *Matcher) WriteGraph(w io.Writer) error {
	if !m.compiled {
		m.Compile()
	}

	out := bufio.NewWriter(w)
	fmt.Fprintln(out, "digraph cedar {")
	fmt.Fprintln(out, "\trankdir=LR;")
	fmt.Fprintln(out, "\tnode [shape=circle];")
	fmt.Fprintln(out, "\t0 [label=\"root\"];")

	da := m.da
	var labels []byte
	queue := []int{0}
	for len(queue) > 0 {
		nid := queue[0]
		queue = queue[1:]

		labels = da.Children(nid, labels[:0])
		for i := range labels {
			child, err := da.Jump(labels[i:i+1], nid)
			if err != nil {
				return err
			}
			queue = append(queue, child)

			if val, err := da.Value(child); err == nil {
				fmt.Fprintf(out, "\t%d [shape=doublecircle label=\"%d\\n=%d\"];\n", child, child, val)
			}
			fmt.Fprintf(out, "\t%d -> %d [label=\"%s\"];\n", nid, child, dotLabel(labels[i]))
			if f := m.fails[child]; f != 0 {
				fmt.Fprintf(out, "\t%d -> %d [style=dashed color=red constraint=false];\n", child, f)
			}
		}
	}
	fmt.Fprintln(out, "}")

	return out.Flush()
}

// dotLabel renders a byte as a DOT edge label: printable ASCII as-is
// (quotes and backslashes escaped), anything else as hex.
func dotLabel(b byte) string {
	switch {
	case b == '"' || b == '\\':
		return `\` + string(b)
	case b > ' ' && b < 0x7f:
		return string(b)
	}

	return fmt.Sprintf("0x%02X", b)
}
