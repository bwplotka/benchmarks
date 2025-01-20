### Local gRPC

The Goal: What's the cleanest and most efficient in-process, transparent gRPC server streaming call?

Blog post about this will be available soon!

Bench command:

```
export bench=bench01-2025 && go test \
	  -run '^$' -bench '^BenchmarkLocal' \
	  -benchtime 5s -count 6 -cpu 2 -timeout 999m \
	  | tee ${bench}.txt 
```

Results:

```
benchstat -col /impl -filter "/respSize:1" ./bench01-2025.txt 
goos: darwin
goarch: arm64
pkg: github.com/bwplotka/benchmarks/benchmarks/local-grpc
cpu: Apple M1 Pro
                   │  localhost   │          unixsocket           │             grpchannel              │          grpchannel-nocpy          │                chan                │               buffer               │                iter                │
                   │    sec/op    │    sec/op     vs base         │    sec/op     vs base               │   sec/op     vs base               │   sec/op     vs base               │   sec/op     vs base               │   sec/op     vs base               │
Local/respSize=1-2   10.333m ± 7%   9.800m ± 31%  ~ (p=0.132 n=6)   8.002m ± 10%  -22.56% (p=0.002 n=6)   5.862m ± 5%  -43.27% (p=0.002 n=6)   4.500m ± 8%  -56.45% (p=0.002 n=6)   1.738m ± 3%  -83.18% (p=0.002 n=6)   2.222m ± 3%  -78.50% (p=0.002 n=6)

                   │   localhost    │              unixsocket              │              grpchannel               │           grpchannel-nocpy           │                chan                │                buffer                │               iter                │
                   │      B/op      │      B/op       vs base              │      B/op       vs base               │     B/op       vs base               │    B/op     vs base                │     B/op       vs base               │    B/op     vs base               │
Local/respSize=1-2   7479835.0 ± 0%   7055631.0 ± 0%  -5.67% (p=0.002 n=6)   1442452.5 ± 0%  -80.72% (p=0.002 n=6)   482182.5 ± 0%  -93.55% (p=0.002 n=6)   320.0 ± 0%  -100.00% (p=0.002 n=6)   310472.0 ± 0%  -95.85% (p=0.002 n=6)   504.0 ± 0%  -99.99% (p=0.002 n=6)

                   │    localhost    │              unixsocket               │              grpchannel               │           grpchannel-nocpy            │                chan                │               buffer               │                iter                │
                   │    allocs/op    │    allocs/op     vs base              │   allocs/op     vs base               │   allocs/op     vs base               │ allocs/op   vs base                │  allocs/op   vs base               │  allocs/op   vs base               │
Local/respSize=1-2   160279.000 ± 0%   160952.500 ± 0%  +0.42% (p=0.002 n=6)   40037.000 ± 0%  -75.02% (p=0.002 n=6)   10034.000 ± 0%  -93.74% (p=0.002 n=6)   4.000 ± 0%  -100.00% (p=0.002 n=6)   19.000 ± 0%  -99.99% (p=0.002 n=6)   15.000 ± 0%  -99.99% (p=0.002 n=6)
```

### Resources

* https://github.com/grpc/grpc-go/issues/906
* https://github.com/thanos-io/thanos/pull/7796
* https://go.dev/blog/range-functions
* https://docs.google.com/presentation/d/1NuGOFDfb5sN-povUCouvGx05OtyFXdLFlANMZgGPLfg/edit#slide=id.g31a4995a9ac_0_376
* https://pkg.go.dev/github.com/fullstorydev/grpchan/inprocgrpc
* https://go.dev/src/runtime/coro.go
