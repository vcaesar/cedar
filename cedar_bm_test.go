package cedar

import (
	"fmt"
	"testing"

	"github.com/vcaesar/tt"
)

func init() {
	InitCd()
}

func BenchmarkInsert(t *testing.B) {
	fn := func() {
		cd.Insert([]byte("a"), 1)
		cd.Insert([]byte("b"), 3)
		cd.Insert([]byte("d"), 6)
	}

	tt.BM(t, fn)
}

func BenchmarkJump(t *testing.B) {
	fn := func() {
		cd.Jump([]byte("a"), 1)
	}

	tt.BM(t, fn)
}

func BenchmarkFind(t *testing.B) {
	fn := func() {
		cd.Find([]byte("a"), 1)
	}

	tt.BM(t, fn)
}

func BenchmarkValue(t *testing.B) {
	fn := func() {
		cd.Value(1)
	}

	tt.BM(t, fn)
}

func BenchmarkUpdate(t *testing.B) {
	fn := func() {
		cd.Update([]byte("a"), 1)
	}

	tt.BM(t, fn)
}

func BenchmarkDelete(t *testing.B) {
	fn := func() {
		cd.Delete([]byte("b"))
	}

	tt.BM(t, fn)
}

// bulk keys shared by the bulk benchmarks below
var bulkKeys = func() [][]byte {
	keys := make([][]byte, 50000)
	for i := range keys {
		keys[i] = []byte(fmt.Sprintf("键%d/k%x", i, i*7))
	}
	return keys
}()

func benchmarkBulkInsert(b *testing.B, reduced bool) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		d := New(reduced)
		for j, k := range bulkKeys {
			d.Insert(k, j)
		}
	}
}

func BenchmarkBulkInsert(b *testing.B)        { benchmarkBulkInsert(b, false) }
func BenchmarkBulkInsertReduced(b *testing.B) { benchmarkBulkInsert(b, true) }

func benchmarkBulkGet(b *testing.B, reduced bool) {
	d := New(reduced)
	for j, k := range bulkKeys {
		d.Insert(k, j)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, k := range bulkKeys {
			d.Get(k)
		}
	}
}

func BenchmarkBulkGet(b *testing.B)        { benchmarkBulkGet(b, false) }
func BenchmarkBulkGetReduced(b *testing.B) { benchmarkBulkGet(b, true) }

func BenchmarkBulkDelete(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		d := New(false)
		for j, k := range bulkKeys {
			d.Insert(k, j)
		}
		b.StartTimer()
		for _, k := range bulkKeys {
			d.Delete(k)
		}
	}
}
