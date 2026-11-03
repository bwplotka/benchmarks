// Copyright (c) Bartłomiej Płotka @bwplotka
// Licensed under the Apache License 2.0.

package rustgodecode

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	writev2 "github.com/bwplotka/benchmarks/benchmarks/metrics-streaming/io/prometheus/write/v2"
)

var (
	sinkUint64 uint64
	sinkRW2    *writev2.Request
)

func TestFixturesAndDecoders(t *testing.T) {
	manifest := ensureFixtures(t)

	rawVarints, err := os.ReadFile(filepath.Join("testdata", "varints_2000.bin"))
	if err != nil {
		t.Fatalf("read varints: %v", err)
	}
	dst := make([]uint64, manifest.VarintsCount)
	n, err := DecodeVarints(rawVarints, dst)
	if err != nil {
		t.Fatalf("DecodeVarints: %v", err)
	}
	if n != manifest.VarintsCount {
		t.Fatalf("expected %d varints, got %d", manifest.VarintsCount, n)
	}
	var sum uint64
	for _, v := range dst[:n] {
		sum += v
	}
	if sum != manifest.VarintsSum {
		t.Fatalf("varint sum mismatch: expected %d, got %d", manifest.VarintsSum, sum)
	}

	for _, samples := range []int{200, 2000} {
		v2File := fmt.Sprintf("rw2_%d.snappy", samples)
		v2Compressed, err := os.ReadFile(filepath.Join("testdata", v2File))
		if err != nil {
			t.Fatalf("read %s: %v", v2File, err)
		}
		v2Req, err := DecodeRemoteWrite2(v2Compressed)
		if err != nil {
			t.Fatalf("DecodeRemoteWrite2(%s): %v", v2File, err)
		}
		gotV2, err := SummarizeRemoteWrite2(v2Compressed, v2Req)
		if err != nil {
			t.Fatalf("SummarizeRemoteWrite2(%s): %v", v2File, err)
		}
		if gotV2 != manifest.RemoteWrite[v2File] {
			t.Fatalf("%s summary mismatch:\ngot:  %+v\nwant: %+v", v2File, gotV2, manifest.RemoteWrite[v2File])
		}
	}
}

// Recommended CLI invocation:
//
//	export bench=results/go && go test \
//	    -run '^$' -bench '^BenchmarkDecode' \
//	    -benchtime 2s -count 6 -cpu 2 -timeout 999m -benchmem \
//	| tee ${bench}.txt
func BenchmarkDecodeVarints(b *testing.B) {
	manifest := ensureFixtures(b)
	raw, err := os.ReadFile(filepath.Join("testdata", "varints_2000.bin"))
	if err != nil {
		b.Fatalf("read varints: %v", err)
	}
	dst := make([]uint64, manifest.VarintsCount)

	b.Run("samples=2000/lang=go", func(b *testing.B) {
		b.SetBytes(int64(len(raw)))
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			n, err := DecodeVarints(raw, dst)
			if err != nil || n != manifest.VarintsCount {
				b.Fatalf("unexpected decode result: n=%d err=%v", n, err)
			}
			sinkUint64 = dst[n-1]
		}
	})
}

// Recommended CLI invocation:
//
//	export bench=results/go && go test \
//	    -run '^$' -bench '^BenchmarkDecode' \
//	    -benchtime 2s -count 6 -cpu 2 -timeout 999m -benchmem \
//	| tee ${bench}.txt
func BenchmarkDecodeRemoteWrite2(b *testing.B) {
	ensureFixtures(b)

	for _, samples := range []int{200, 2000} {
		file := fmt.Sprintf("rw2_%d.snappy", samples)
		compressed, err := os.ReadFile(filepath.Join("testdata", file))
		if err != nil {
			b.Fatalf("read %s: %v", file, err)
		}

		name := fmt.Sprintf("samples=%d/lang=go", samples)
		b.Run(name, func(b *testing.B) {
			b.SetBytes(int64(len(compressed)))
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				req, err := DecodeRemoteWrite2(compressed)
				if err != nil || len(req.Timeseries) != samples {
					b.Fatalf("unexpected v2 decode: err=%v", err)
				}
				sinkRW2 = req
			}
		})
	}
}
