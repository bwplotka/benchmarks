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
    let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("testdata");
    let raw = fs::read(root.join("varints_2000.bin")).unwrap();
    let mut dst = vec![0u64; 2000];

    let mut group = c.benchmark_group("DecodeVarints");
    group.throughput(Throughput::Bytes(raw.len() as u64));
    group.bench_function("samples=2000/lang=rust", |b| {
        b.iter(|| {
            let n = decode_varints(black_box(&raw), black_box(&mut dst)).unwrap();
            assert_eq!(n, 2000);
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

        group.throughput(Throughput::Bytes(compressed.len() as u64));
        let id = format!("samples={samples}/lang=rust");
        group.bench_function(&id, |b| {
            b.iter(|| {
                let req = decode_remote_write2(black_box(&compressed)).unwrap();
                assert_eq!(req.timeseries.len(), samples);
                black_box(req);
            });
        });
    }
    group.finish();
}

criterion_group!(benches, bench_varints, bench_remote_write2);
criterion_main!(benches);
