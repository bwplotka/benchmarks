module github.com/bwplotka/benchmarks/benchmarks/rust-go-decode

go 1.24.0

require (
	github.com/bwplotka/benchmarks/benchmarks/metrics-streaming v0.0.0
	github.com/golang/snappy v0.0.4
	github.com/prometheus/common v0.55.0
	github.com/prometheus/prometheus v0.54.2-0.20240828062354-3a82cd5a7ed4
	google.golang.org/protobuf v1.34.2
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/grafana/regexp v0.0.0-20240518133315-a468a5bfb3bc // indirect
	github.com/planetscale/vtprotobuf v0.6.0 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	golang.org/x/text v0.16.0 // indirect
)

replace github.com/bwplotka/benchmarks/benchmarks/metrics-streaming => ../metrics-streaming
