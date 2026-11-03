// Copyright (c) Bartłomiej Płotka @bwplotka
// Licensed under the Apache License 2.0.

//! Criterion benchmarks for `decode_varints` and Prometheus Remote Write 2.0 decoding.
//!
//! Recommended CLI invocation:
//! ```sh
//! cargo bench --bench criterion_decode
//! benchfmt criterion target/criterion --pkg github.com/bwplotka/benchmarks/benchmarks/rust-go-decode --cpu 2 > results/rust_criterion.txt
//! ```

use std::fs;
use std::hint::black_box;
use std::path::Path;

use criterion::{Criterion, Throughput, criterion_group, criterion_main};
use rust_go_decode::{decode_remote_write2, decode_varints};

fn bench_varints(c: &mut Criterion) {
    // Resolve testdata/ from the crate directory baked in at compile time.
    let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("testdata");
    let raw = fs::read(root.join("varints_2000.bin")).unwrap();
    // Preallocate the output buffer outside the timed loop so each op does 0 heap allocs.
    let mut dst = vec![0u64; 2000];

    // Group + function ID join into "DecodeVarints/samples=2000/lang=rust" to match Go's b.Run.
    let mut group = c.benchmark_group("DecodeVarints");
    // Report decimal MB/s (1e6 B/s) to match Go's b.SetBytes; Throughput::Bytes prints MiB/s.
    group.throughput(Throughput::BytesDecimal(raw.len() as u64));
    group.bench_function("samples=2000/lang=rust", |b| {
        // b.iter runs the closure across Criterion's warmup and 100 linear-ramp samples.
        b.iter(|| {
            // std::hint::black_box is an LLVM optimization barrier: Unlike Go's b.Loop(), Rust
            // would constant-fold known inputs or delete unused work without black_box hints.
            let n = decode_varints(black_box(&raw), black_box(&mut dst)).unwrap();
            assert_eq!(n, 2000);
            // Observe the written buffer so LLVM cannot optimize away stores into dst.
            black_box(dst[n - 1]);
        });
    });
    group.finish();
}

fn bench_remote_write2(c: &mut Criterion) {
    let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("testdata");
    let mut group = c.benchmark_group("DecodeRemoteWrite2");

    for samples in [200usize, 2000usize] {
        let file = format!("rw2_{samples}.snappy");
        let compressed = fs::read(root.join(file)).unwrap();

        group.throughput(Throughput::BytesDecimal(compressed.len() as u64));
        let id = format!("samples={samples}/lang=rust");
        group.bench_function(&id, |b| {
            b.iter(|| {
                let req = decode_remote_write2(black_box(&compressed)).unwrap();
                assert_eq!(req.timeseries.len(), samples);
                // Force req to materialize before dropping at the end of the iteration.
                black_box(req);
            });
        });
    }
    group.finish();
}

// Generate the benchmark group runner and custom main() required by harness = false in Cargo.toml.
criterion_group!(benches, bench_varints, bench_remote_write2);
criterion_main!(benches);
