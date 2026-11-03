// Copyright (c) Bartłomiej Płotka @bwplotka
// Licensed under the Apache License 2.0.

//! `benchfmt::Suite` benchmarks with `CountingAlloc` and optional `pprof` profiling.
//!
//! Recommended CLI invocation:
//! ```sh
//! cargo bench --bench suite_decode -- -bench '^BenchmarkDecode' -benchtime 2s -count 6 -cpu 2 -benchmem | tee results/rust_suite.txt
//! ```

use std::fs;
use std::hint::black_box;
use std::path::Path;

use benchfmt::{CountingAlloc, Suite};
use rust_go_decode::{decode_remote_write2, decode_varints};

// Wrap Rust's System allocator globally so every heap alloc/realloc on the benchmark thread
// increments thread-local (allocs, bytes) counters for B/op and allocs/op.
#[global_allocator]
static ALLOC: CountingAlloc = CountingAlloc::new();

fn main() {
    let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("testdata");
    let raw_varints = fs::read(root.join("varints_2000.bin")).unwrap();
    let rw2_200 = fs::read(root.join("rw2_200.snappy")).unwrap();
    let rw2_2000 = fs::read(root.join("rw2_2000.snappy")).unwrap();
    let mut dst = vec![0u64; 2000];

    // Parse go-test-compatible CLI flags (-bench, -benchtime, -count, -cpu, -benchmem,
    // -cpuprofile, -memprofile) from std::env::args() and set the default "pkg:" header.
    // Using the same pkg path as Go's go.mod lets `benchstat -col /lang` join Go and Rust rows.
    Suite::from_env("github.com/bwplotka/benchmarks/benchmarks/rust-go-decode")
        // Attach ALLOC so each benchmark snapshots thread-local allocation deltas per epoch.
        .with_alloc(&ALLOC)
        // Suite prepends "Benchmark" in benchfmt output, matching Go's BenchmarkDecodeVarints.
        .bench("DecodeVarints/samples=2000/lang=rust", |b| {
            b.set_bytes(raw_varints.len() as u64);
            b.report_allocs();
            // Go b.Loop() equivalent: starts timing, alloc snapshots, and any pprof capture on the
            // first call, runs the calibrated N steady-state iterations, then stops all three.
            while b.keep_running() {
                let n = decode_varints(black_box(&raw_varints), black_box(&mut dst)).unwrap();
                assert_eq!(n, 2000);
                black_box(dst[n - 1]);
            }
        })
        .bench("DecodeRemoteWrite2/samples=200/lang=rust", |b| {
            b.set_bytes(rw2_200.len() as u64);
            b.report_allocs();
            while b.keep_running() {
                let req = decode_remote_write2(black_box(&rw2_200)).unwrap();
                assert_eq!(req.timeseries.len(), 200);
                black_box(req);
            }
        })
        .bench("DecodeRemoteWrite2/samples=2000/lang=rust", |b| {
            b.set_bytes(rw2_2000.len() as u64);
            b.report_allocs();
            while b.keep_running() {
                let req = decode_remote_write2(black_box(&rw2_2000)).unwrap();
                assert_eq!(req.timeseries.len(), 2000);
                black_box(req);
            }
        })
        .run();
}
