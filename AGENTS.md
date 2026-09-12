# AGENTS.md — cedar

Go port of [cedar](http://www.tkl.iis.u-tokyo.ac.jp/~ynaga/cedar) (updatable double-array trie).
Module `github.com/vcaesar/cedar`, BSD-2, single package. Only dep: `github.com/vcaesar/tt` (test asserts).

## Layout

- `cedar.go` — `Node`/`Block`/`NInfo`/`Cedar`, `New()`, block alloc, `follow`/`resolve` relocation
- `fn.go` — public API (`Jump`, `Find`, `Value`, `Insert`, `Update`, `Delete`, `Get`, `ExactMatch`, `PrefixMatch`, `PrefixPredict`) and `Err*` sentinels
- `io.go` — `GobEncode`/`GobDecode` and `MarshalJSON`/`UnmarshalJSON` on `Cedar`, both via the mirror struct `cedarData` (`toData`/`fromData`; `childBuf` is scratch and not persisted)
- `aho.go` — empty root stub; the implementation is the `aho/` package
- `aho/aho.go` — `Matcher` (Aho-Corasick over `*cedar.Cedar`): `Insert`/`Delete`, lazy `Compile` (BFS fail/output/depth links), `Match`/`Has`/`Key`; `aho/io.go` — `Save`/`Load` (`"gob"` or `"json"`, else `ErrDataType`) and channel `PrefixPredict`; `aho/graph.go` — `DumpGraph(fname)`/`WriteGraph(w)` emit Graphviz DOT (trie edges, terminal values, fail links except to root)
- `aho/aho_test.go` — classic/NUL/recompile/save-load cases plus a random oracle vs naive search, both trie modes
- `cedar_test.go`, `int32_test.go`, `io_test.go`, `oracle_test.go` — tests; `cedar_bm_test.go` — benchmarks, its `init()` builds the shared test trie
- `examples/main.go` — trie usage demo; `examples/aho/main.go` — Aho-Corasick demo (match, delete, save/load, PrefixPredict, DumpGraph)

## Commands

```sh
go vet ./... && go build -v ./... && go test -v -race ./...   # CI (go.mod 1.17; CI pins 1.27.x)
go test -bench . -run '^$' -benchmem .
gofmt -l . && go vet ./...
```

## Rules

- `gofmt`, BSD header on every file, receiver `cd`, doc comments on exports.
- Return `Err*` sentinels unwrapped (`fn.go`).
- Public API: `[]byte` keys, `int` values; internals store `int32` in `Node.baseV`.
- `New()` = reduced trie; `New(false)` = non-reduced. Test both (`int32_test.go` loops `{true,false}`).
- Valid values: `0 <= v < ValLimit` (`math.MaxInt32`); `-1`/`ValLimit` → `ErrInvalidVal`.
- Size invariants: `Node` 8 bytes, `Block` 24 bytes (`TestNodeSize`). Block/head indexes are `int32`.
- `Node.base(reduced bool)` is non-variadic and hot; keep it inlinable.
- `setChild` returns a slice aliasing `cd.childBuf` — valid only until the next call.
- `Children(from, dst)` skips the terminal (label 0) edge; the root's terminal slot is the root itself (base 0), so its chain starts at `nInfos[0].sibling`. `Size()` bounds node indexes.
- `aho`: patterns must not contain NUL (rejected with `ErrInvalidKey`); NUL in text resets the automaton. Any mutation or `Load` clears `compiled`.
- Keep algorithm comments accurate to the C++ cedar semantics.

## Test pitfalls

- `TestFind`/`TestPrefixMatch`/`TestPrefixPredict` depend on the global `cd` filled by `TestLoadData`; they fail under `-run` isolation or `-shuffle`.
- `TestFind` hardcodes node index `352` for `Jump("abc")`; any allocation-order change breaks it — confirm intent before updating.
- `TestManyKeys` (200k UTF-8 keys, both modes) is the regression guard for `resolve`/`popENode`.
- Benchmarks mutate the global `cd`. New tests should build their own `New(...)`.
