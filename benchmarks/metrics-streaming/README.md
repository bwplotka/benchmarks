# Metric streaming

A micro-benchmark for Prometheus Remote Write (v1, v2) and OpenTelemetry OTLP metric streaming protocols across compressions (`none`, `snappy`, `zstd`, `gzip`), protobuf implementations (`google.golang.org/protobuf` vs `vtprotobuf`), and transports (`http1`, `http2`, `grpc`).

See [benchmark_test.go](./benchmark_test.go) for `BenchmarkEncode`, `BenchmarkDecode`, and `BenchmarkTransport` and their recommended `go test` CLI invocations.

See the summarization of September 9th 2024 benchmark in [PromCon2024](https://docs.google.com/spreadsheets/d/1LF-jKLssmYC89M8VoGi5MYEXfJvxppoDcwd_7n0uz0Q/edit?gid=1469183762#gid=1469183762) sheet, and the October 2026 benchmark logs in [bench-encode-10-2026.txt](./bench-encode-10-2026.txt), [bench-decode-10-2026.txt](./bench-decode-10-2026.txt), and [bench-transport-10-2026.txt](./bench-transport-10-2026.txt).
