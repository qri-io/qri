package oplog

import (
	"testing"

	"github.com/dustin/go-humanize"
)

func BenchmarkSave10kOpsOneAuthor(b *testing.B) {
	tr, cleanup := newTestRunner(b)
	defer cleanup()

	init := Op{
		Type:  OpTypeInit,
		Model: 0xFFFF,
	}

	l := tr.RandomLog(init, 10000)
	book := tr.Book
	book.AppendLog(l)

	data, err := book.FlatbufferCipher()
	if err != nil {
		b.Fatal(err)
	}

	b.Logf("one simulated log with 10k ops weighs %s as encrypted data", humanize.Bytes(uint64(len(data))))
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		if _, err := book.FlatbufferCipher(); err != nil {
			b.Fatal(err)
		}
	}
}