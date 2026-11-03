// Copyright (c) Bartłomiej Płotka @bwplotka
// Licensed under the Apache License 2.0.

//! Standalone release binary (using Rust's default `System` allocator, no benchmark harness)
//! to cross-check microbenchmark `ns/op` and `B/op` numbers against `/usr/bin/time -v`.

use std::env;
use std::fs;
use std::hint::black_box;
use std::path::Path;
use std::time::Instant;

use rust_go_decode::{decode_remote_write2, decode_varints};

fn main() {
    let mut case = "rw2_2000".to_owned();
    let mut iters: usize = 1_000;
    let mut retain: usize = 0;

    let mut args = env::args().skip(1);
    while let Some(arg) = args.next() {
        match arg.as_str() {
            "--case" => case = args.next().expect("missing --case value"),
            "--iters" => iters = args.next().expect("missing --iters value").parse().unwrap(),
            "--retain" => retain = args.next().expect("missing --retain value").parse().unwrap(),
            other => panic!("unknown flag {other}"),
        }
    }

    let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("testdata");

    match case.as_str() {
        "varints" => {
            let raw = fs::read(root.join("varints_2000.bin")).unwrap();
            let mut dst = vec![0u64; 2000];
            let mut acc = 0u64;
            let start = Instant::now();
            for _ in 0..iters {
                let n = decode_varints(black_box(&raw), black_box(&mut dst)).unwrap();
                acc = acc.wrapping_add(dst[n - 1]);
            }
            let elapsed = start.elapsed();
            println!(
                "case={case} iters={iters} retain={retain} elapsed={elapsed:?} ns_per_op={:.1} checksum={acc}",
                elapsed.as_nanos() as f64 / iters as f64
            );
        }
        "rw2_2000" => {
            let compressed = fs::read(root.join("rw2_2000.snappy")).unwrap();
            let mut retained = Vec::with_capacity(retain);
            let mut series_total = 0usize;
            let start = Instant::now();
            for _ in 0..iters {
                let req = decode_remote_write2(black_box(&compressed)).unwrap();
                series_total += req.timeseries.len();
                if retain > 0 && retained.len() < retain {
                    retained.push(req);
                } else {
                    black_box(req);
                }
            }
            let elapsed = start.elapsed();
            black_box(&retained);
            println!(
                "case={case} iters={iters} retain={} elapsed={elapsed:?} ns_per_op={:.0} series_total={series_total}",
                retained.len(),
                elapsed.as_nanos() as f64 / iters as f64
            );
        }
        other => panic!("unsupported --case {other}; expected varints or rw2_2000"),
    }
}
