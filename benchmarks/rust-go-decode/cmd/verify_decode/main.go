// Copyright (c) Bartłomiej Płotka @bwplotka
// Licensed under the Apache License 2.0.

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	writev2 "github.com/bwplotka/benchmarks/benchmarks/metrics-streaming/io/prometheus/write/v2"
	rustgodecode "github.com/bwplotka/benchmarks/benchmarks/rust-go-decode"
)

func main() {
	caseFlag := flag.String("case", "rw2_2000", "varints or rw2_2000")
	iters := flag.Int("iters", 1000, "iterations to execute")
	retain := flag.Int("retain", 0, "number of decoded messages to retain in memory")
	flag.Parse()

	switch *caseFlag {
	case "varints":
		raw, err := os.ReadFile(filepath.Join("testdata", "varints_2000.bin"))
		if err != nil {
			panic(err)
		}
		dst := make([]uint64, 2000)
		var acc uint64
		start := time.Now()
		for range *iters {
			n, err := rustgodecode.DecodeVarints(raw, dst)
			if err != nil {
				panic(err)
			}
			acc += dst[n-1]
		}
		elapsed := time.Since(start)
		fmt.Printf("case=%s iters=%d retain=%d elapsed=%v ns_per_op=%.1f checksum=%d\n",
			*caseFlag, *iters, *retain, elapsed, float64(elapsed.Nanoseconds())/float64(*iters), acc)

	case "rw2_2000":
		compressed, err := os.ReadFile(filepath.Join("testdata", "rw2_2000.snappy"))
		if err != nil {
			panic(err)
		}
		retained := make([]*writev2.Request, 0, *retain)
		seriesTotal := 0
		start := time.Now()
		for range *iters {
			req, err := rustgodecode.DecodeRemoteWrite2(compressed)
			if err != nil {
				panic(err)
			}
			seriesTotal += len(req.Timeseries)
			if *retain > 0 && len(retained) < *retain {
				retained = append(retained, req)
			} else {
				runtime.KeepAlive(req)
			}
		}
		elapsed := time.Since(start)
		runtime.KeepAlive(retained)
		fmt.Printf("case=%s iters=%d retain=%d elapsed=%v ns_per_op=%.0f series_total=%d\n",
			*caseFlag, *iters, len(retained), elapsed, float64(elapsed.Nanoseconds())/float64(*iters), seriesTotal)

	default:
		panic("unsupported -case " + *caseFlag)
	}
}
