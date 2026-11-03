# Rust vs Go decoding benchmarks with `benchfmt-rs`

Companion benchmark suite comparing Go `testing.B` and Rust `Criterion`, `benchfmt criterion`, and `benchfmt::Suite` across two workloads:

1. **`DecodeVarints`.** Decodes `2,000` unsigned LEB128 `uint64` varints from `4,798` bytes into a preallocated slice with `0 B/op` and `0 allocs/op`.
2. **`DecodeRemoteWrite2`.** Runs Snappy block decompression and Protobuf decoding for Prometheus Remote Write 2.0 `io.prometheus.write.v2.Request` payloads of `200` and `2,000` series, comparing `vtprotobuf` and `golang/snappy` in Go against `prost` and `snap` in Rust.

## Commands

```bash
# 1. Verify byte-for-byte fixture equivalence across Go and Rust:
go test ./...
cargo test --lib

# 2. Run Go benchmarks:
go test -run '^$' -bench '^BenchmarkDecode' -benchtime 2s -count 6 -cpu 2 -timeout 999m -benchmem | tee results/go.txt

# 3. Run Rust Criterion benchmarks and convert to Go benchfmt:
cargo bench --bench criterion_decode | tee results/criterion_console.txt
benchfmt criterion target/criterion --pkg github.com/bwplotka/benchmarks/benchmarks/rust-go-decode --cpu 2 | tee results/rust_criterion.txt
benchstat -col /lang results/go.txt results/rust_criterion.txt

# 4. Run Rust benchfmt::Suite benchmarks with CountingAlloc:
cargo bench --bench suite_decode -- -bench '^BenchmarkDecode' -benchtime 2s -count 6 -cpu 2 -benchmem | tee results/rust_suite.txt
benchstat -col /lang results/go.txt results/rust_suite.txt

# 5. Profile CPU and heap allocations across Go and Rust with go tool pprof:
go test -run '^$' -bench '^BenchmarkDecodeVarints$' -benchtime 3s -count 1 -cpu 1 -cpuprofile /tmp/go_varints_cpu.pb.gz
cargo bench --bench suite_decode -- -bench '^BenchmarkDecodeVarints/' -benchtime 3s -count 1 -cpu 1 -cpuprofile /tmp/rust_varints_cpu.pb.gz
go tool pprof -list DecodeVarints /tmp/go_varints_cpu.pb.gz
go tool pprof -list decode_varints /tmp/rust_varints_cpu.pb.gz

go test -run '^$' -bench '^BenchmarkDecodeRemoteWrite2/samples=2000/' -benchtime 3s -count 1 -cpu 1 -cpuprofile /tmp/go_rw2_cpu.pb.gz
cargo bench --bench suite_decode -- -bench 'DecodeRemoteWrite2/samples=2000' -benchtime 3s -count 1 -cpu 1 -cpuprofile /tmp/rust_rw2_cpu.pb.gz
go test -run '^$' -bench '^BenchmarkDecodeRemoteWrite2/samples=2000/' -benchtime 50x -count 1 -cpu 1 -memprofile /tmp/go_rw2_mem.pb.gz -memprofilerate 1
cargo bench --bench suite_decode -- -bench 'DecodeRemoteWrite2/samples=2000' -benchtime 50x -count 1 -cpu 1 -memprofile /tmp/rust_rw2_mem.pb.gz -memprofilerate 1

# 6. Cross-check microbenchmarks with standalone release binaries under /usr/bin/time -v:
cargo build --release --bin verify_decode
go build -o verify_decode_go ./cmd/verify_decode
/usr/bin/time -v ./target/release/verify_decode --case rw2_2000 --iters 1000 --retain 0
/usr/bin/time -v ./target/release/verify_decode --case rw2_2000 --iters 1000 --retain 1000
GOMAXPROCS=2 /usr/bin/time -v ./verify_decode_go -case rw2_2000 -iters 1000 -retain 0
```
