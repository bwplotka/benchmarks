### Local gRPC

The Goal: What's the cleanest and most efficient in-process, transparent gRPC server streaming call?

Blog post about this will be available soon!

Bench command:

```
export bench=bench12-2024 && go test \
	  -run '^$' -bench '^BenchmarkLocal' \
	  -benchtime 5s -count 6 -cpu 2 -timeout 999m \
	  | tee ${bench}.txt 
```

Results:

```
benchstat -col /impl ./bench12-2024.txt 
goos: darwin
goarch: arm64
pkg: github.com/bwplotka/benchmarks/benchmarks/local-grpc
cpu: Apple M1 Pro
                     │   localhost   │              unixsocket               │             grpchannel              │                chan                 │               buffer               │                iter                │
                     │    sec/op     │     sec/op      vs base               │    sec/op     vs base               │    sec/op     vs base               │   sec/op     vs base               │   sec/op     vs base               │
Local/respSize=1-2      10.035m ± 2%     8.812m ± 13%  -12.19% (p=0.009 n=6)    7.521m ± 5%  -25.05% (p=0.002 n=6)   4.393m ±  4%  -56.22% (p=0.002 n=6)   1.662m ± 5%  -83.44% (p=0.002 n=6)   2.223m ± 5%  -77.84% (p=0.002 n=6)
Local/respSize=10-2     2283.8µ ± 6%    2309.3µ ± 25%        ~ (p=1.000 n=6)    895.9µ ± 6%  -60.77% (p=0.002 n=6)   444.7µ ± 12%  -80.53% (p=0.002 n=6)   166.4µ ± 3%  -92.71% (p=0.002 n=6)   228.7µ ± 2%  -89.99% (p=0.002 n=6)
Local/respSize=100-2   1137.32µ ± 8%   1150.77µ ±  5%        ~ (p=0.937 n=6)   194.77µ ± 2%  -82.87% (p=0.002 n=6)   44.97µ ±  4%  -96.05% (p=0.002 n=6)   17.02µ ± 4%  -98.50% (p=0.002 n=6)   23.08µ ± 4%  -97.97% (p=0.002 n=6)
geomean                  2.965m          2.861m         -3.51%                  1.095m       -63.07%                 444.5µ        -85.01%                 167.6µ       -94.35%                 227.3µ       -92.34%

                     │   localhost    │              unixsocket              │              grpchannel               │                chan                │                buffer                │               iter                │
                     │      B/op      │      B/op       vs base              │      B/op       vs base               │    B/op     vs base                │     B/op       vs base               │    B/op     vs base               │
Local/respSize=1-2     7548752.5 ± 1%   7056027.0 ± 0%  -6.53% (p=0.002 n=6)   1442457.5 ± 0%  -80.89% (p=0.002 n=6)   320.0 ± 0%  -100.00% (p=0.002 n=6)   310472.0 ± 0%  -95.89% (p=0.002 n=6)   504.0 ± 0%  -99.99% (p=0.002 n=6)
Local/respSize=10-2    3268171.5 ± 0%   2986950.0 ± 0%  -8.60% (p=0.002 n=6)    434227.5 ± 0%  -86.71% (p=0.002 n=6)   320.0 ± 0%   -99.99% (p=0.002 n=6)    17608.0 ± 0%  -99.46% (p=0.002 n=6)   504.0 ± 0%  -99.98% (p=0.002 n=6)
Local/respSize=100-2   1644130.5 ± 0%   1612898.0 ± 0%  -1.90% (p=0.002 n=6)    371799.0 ± 0%  -77.39% (p=0.002 n=6)   320.0 ± 0%   -99.98% (p=0.002 n=6)     2248.0 ± 0%  -99.86% (p=0.002 n=6)   504.0 ± 0%  -99.97% (p=0.002 n=6)
geomean                  3.277Mi          3.089Mi       -5.72%                   600.8Ki       -82.09%                 320.0        -99.99%                  22.54Ki       -99.33%                 504.0       -99.99%

                     │    localhost    │              unixsocket               │              grpchannel               │                chan                │               buffer               │                iter                │
                     │    allocs/op    │    allocs/op     vs base              │   allocs/op     vs base               │ allocs/op   vs base                │  allocs/op   vs base               │  allocs/op   vs base               │
Local/respSize=1-2     160269.500 ± 0%   160872.000 ± 0%  +0.38% (p=0.002 n=6)   40037.000 ± 0%  -75.02% (p=0.002 n=6)   4.000 ± 0%  -100.00% (p=0.002 n=6)   19.000 ± 0%  -99.99% (p=0.002 n=6)   15.000 ± 0%  -99.99% (p=0.002 n=6)
Local/respSize=10-2     26493.500 ± 0%    26682.500 ± 0%  +0.71% (p=0.002 n=6)    4035.000 ± 0%  -84.77% (p=0.002 n=6)   4.000 ± 0%   -99.98% (p=0.002 n=6)   12.000 ± 0%  -99.95% (p=0.002 n=6)   15.000 ± 0%  -99.94% (p=0.002 n=6)
Local/respSize=100-2    11945.000 ± 0%    12086.000 ± 1%  +1.18% (p=0.002 n=6)     435.000 ± 0%  -96.36% (p=0.002 n=6)   4.000 ± 0%   -99.97% (p=0.002 n=6)    9.000 ± 0%  -99.92% (p=0.002 n=6)   15.000 ± 0%  -99.87% (p=0.002 n=6)
geomean                    37.02k            37.30k       +0.76%                    4.127k       -88.85%                 4.000        -99.99%                  12.71       -99.97%                  15.00       -99.96%
```

### Resources



* https://github.com/grpc/grpc-go/issues/906
* https://github.com/thanos-io/thanos/pull/7796
* https://go.dev/blog/range-functions
* https://docs.google.com/presentation/d/1NuGOFDfb5sN-povUCouvGx05OtyFXdLFlANMZgGPLfg/edit#slide=id.g31a4995a9ac_0_376
* https://pkg.go.dev/github.com/fullstorydev/grpchan/inprocgrpc
* https://go.dev/src/runtime/coro.go
