# Metric streaming

A micro-benchmark for Remote Write protocols in various configurations.

See [benchmark_test.go](./benchmark_test.go) file and `BenchmarkDecode` and `BenchmarkEncode` tests and their commentary on the
recommended `go test` benchmark configuration.

## Last ran results

Encoding:

```
benchstat -col /proto ./bench-encode-09-2024.txt                                           
goos: darwin
goarch: arm64
pkg: github.com/bwplotka/benchmarks/benchmarks/metrics-streaming
                                                            │ prometheus.WriteRequest │ prometheus.WriteRequest+experiments │ prometheus.WriteRequest+experiments+metadata │   io.prometheus.write.v2.Request   │
                                                            │         sec/op          │    sec/op     vs base               │        sec/op          vs base               │   sec/op     vs base               │
Encode/sample=200/compression=/encoder=vtprotobuf-2                      39.58µ ± 27%    46.47µ ± 3%        ~ (p=0.065 n=6)             59.62µ ± 4%  +50.64% (p=0.002 n=6)   33.48µ ± 3%  -15.41% (p=0.002 n=6)
Encode/sample=200/compression=snappy/encoder=vtprotobuf-2                64.09µ ±  8%    80.66µ ± 4%  +25.85% (p=0.002 n=6)            109.74µ ± 4%  +71.22% (p=0.002 n=6)   55.98µ ± 4%  -12.65% (p=0.002 n=6)
Encode/sample=200/compression=zstd/encoder=vtprotobuf-2                  122.9µ ±  5%    138.5µ ± 4%  +12.65% (p=0.002 n=6)             182.4µ ± 5%  +48.35% (p=0.002 n=6)   114.9µ ± 3%   -6.57% (p=0.002 n=6)
Encode/sample=2000/compression=/encoder=vtprotobuf-2                     391.7µ ±  2%    463.3µ ± 6%  +18.29% (p=0.002 n=6)             591.9µ ± 5%  +51.13% (p=0.002 n=6)   308.0µ ± 2%  -21.37% (p=0.002 n=6)
Encode/sample=2000/compression=snappy/encoder=vtprotobuf-2               638.8µ ±  3%    803.5µ ± 3%  +25.80% (p=0.002 n=6)            1190.9µ ± 1%  +86.44% (p=0.002 n=6)   515.4µ ± 5%  -19.31% (p=0.002 n=6)
Encode/sample=2000/compression=zstd/encoder=vtprotobuf-2                1173.5µ ±  2%   1450.1µ ± 4%  +23.57% (p=0.002 n=6)            1831.6µ ± 4%  +56.08% (p=0.002 n=6)   964.9µ ± 4%  -17.78% (p=0.002 n=6)
Encode/sample=10000/compression=/encoder=vtprotobuf-2                    1.931m ±  4%    2.458m ± 4%  +27.32% (p=0.002 n=6)             3.561m ± 1%  +84.41% (p=0.002 n=6)   1.452m ± 6%  -24.82% (p=0.002 n=6)
Encode/sample=10000/compression=snappy/encoder=vtprotobuf-2              3.229m ±  1%    4.285m ± 2%  +32.71% (p=0.002 n=6)             5.944m ± 2%  +84.11% (p=0.002 n=6)   2.400m ± 3%  -25.67% (p=0.002 n=6)
Encode/sample=10000/compression=zstd/encoder=vtprotobuf-2                5.876m ±  3%    7.249m ± 2%  +23.37% (p=0.002 n=6)             9.122m ± 5%  +55.24% (p=0.002 n=6)   3.741m ± 1%  -36.33% (p=0.002 n=6)
geomean                                                                  530.9µ          652.3µ       +22.86%                           873.9µ       +64.60%                 422.6µ       -20.41%

                                                            │ prometheus.WriteRequest │ prometheus.WriteRequest+experiments  │ prometheus.WriteRequest+experiments+metadata │    io.prometheus.write.v2.Request    │
                                                            │      bytes/message      │ bytes/message  vs base               │    bytes/message      vs base                │ bytes/message  vs base               │
Encode/sample=200/compression=/encoder=vtprotobuf-2                      80.01Ki ± 0%    89.18Ki ± 0%  +11.46% (p=0.002 n=6)          161.39Ki ± 0%  +101.71% (p=0.002 n=6)    40.04Ki ± 0%  -49.95% (p=0.002 n=6)
Encode/sample=200/compression=snappy/encoder=vtprotobuf-2                7.914Ki ± 0%   11.156Ki ± 0%  +40.97% (p=0.002 n=6)          15.609Ki ± 0%   +97.24% (p=0.002 n=6)   10.297Ki ± 0%  +30.11% (p=0.002 n=6)
Encode/sample=200/compression=zstd/encoder=vtprotobuf-2                  3.500Ki ± 0%    4.911Ki ± 0%  +40.32% (p=0.002 n=6)           5.955Ki ± 0%   +70.15% (p=0.002 n=6)    6.625Ki ± 0%  +89.29% (p=0.002 n=6)
Encode/sample=2000/compression=/encoder=vtprotobuf-2                     804.6Ki ± 0%    896.3Ki ± 0%  +11.40% (p=0.002 n=6)          1618.3Ki ± 0%  +101.14% (p=0.002 n=6)    249.1Ki ± 0%  -69.04% (p=0.002 n=6)
Encode/sample=2000/compression=snappy/encoder=vtprotobuf-2               70.78Ki ± 0%   102.83Ki ± 0%  +45.28% (p=0.002 n=6)          143.18Ki ± 0%  +102.28% (p=0.002 n=6)    76.02Ki ± 0%   +7.41% (p=0.002 n=6)
Encode/sample=2000/compression=zstd/encoder=vtprotobuf-2                 27.51Ki ± 0%    41.22Ki ± 0%  +49.84% (p=0.002 n=6)           45.99Ki ± 0%   +67.20% (p=0.002 n=6)    41.91Ki ± 0%  +52.36% (p=0.002 n=6)
Encode/sample=10000/compression=/encoder=vtprotobuf-2                   4031.8Ki ± 0%   4490.7Ki ± 0%  +11.38% (p=0.002 n=6)          8101.0Ki ± 0%  +100.93% (p=0.002 n=6)   1022.9Ki ± 0%  -74.63% (p=0.002 n=6)
Encode/sample=10000/compression=snappy/encoder=vtprotobuf-2              350.3Ki ± 0%    513.1Ki ± 0%  +46.50% (p=0.002 n=6)           713.6Ki ± 0%  +103.72% (p=0.002 n=6)    332.2Ki ± 0%   -5.15% (p=0.002 n=6)
Encode/sample=10000/compression=zstd/encoder=vtprotobuf-2                133.4Ki ± 0%    208.6Ki ± 0%  +56.38% (p=0.002 n=6)           228.0Ki ± 0%   +70.86% (p=0.002 n=6)    157.8Ki ± 0%  +18.30% (p=0.002 n=6)
geomean                                                                  95.39Ki         127.5Ki       +33.69%                         181.2Ki        +89.96%                  78.74Ki       -17.46%

                                                            │ prometheus.WriteRequest │ prometheus.WriteRequest+experiments  │ prometheus.WriteRequest+experiments+metadata │   io.prometheus.write.v2.Request    │
                                                            │          B/op           │     B/op       vs base               │         B/op          vs base                │     B/op      vs base               │
Encode/sample=200/compression=/encoder=vtprotobuf-2                      88.00Ki ± 0%    96.00Ki ± 0%   +9.09% (p=0.002 n=6)          168.00Ki ± 0%   +90.91% (p=0.002 n=6)   48.00Ki ± 0%  -45.45% (p=0.002 n=6)
Encode/sample=200/compression=snappy/encoder=vtprotobuf-2               184.00Ki ± 0%   208.00Ki ± 0%  +13.04% (p=0.002 n=6)          360.00Ki ± 0%   +95.65% (p=0.002 n=6)   96.00Ki ± 0%  -47.83% (p=0.002 n=6)
Encode/sample=200/compression=zstd/encoder=vtprotobuf-2                 176.36Ki ± 0%   192.40Ki ± 0%   +9.10% (p=0.002 n=6)          336.53Ki ± 0%   +90.82% (p=0.002 n=6)   96.02Ki ± 0%  -45.55% (p=0.002 n=6)
Encode/sample=2000/compression=/encoder=vtprotobuf-2                     808.0Ki ± 0%    904.0Ki ± 0%  +11.88% (p=0.002 n=6)          1624.0Ki ± 0%  +100.99% (p=0.002 n=6)   256.0Ki ± 0%  -68.32% (p=0.002 n=6)
Encode/sample=2000/compression=snappy/encoder=vtprotobuf-2              1752.0Ki ± 0%   1952.0Ki ± 0%  +11.42% (p=0.002 n=6)          3520.0Ki ± 0%  +100.91% (p=0.002 n=6)   552.0Ki ± 0%  -68.49% (p=0.002 n=6)
Encode/sample=2000/compression=zstd/encoder=vtprotobuf-2                1619.6Ki ± 0%   1812.4Ki ± 0%  +11.90% (p=0.002 n=6)          1805.3Ki ± 0%   +11.47% (p=0.002 n=6)   515.0Ki ± 0%  -68.20% (p=0.002 n=6)
Encode/sample=10000/compression=/encoder=vtprotobuf-2                    3.938Mi ± 0%    4.391Mi ± 0%  +11.51% (p=0.002 n=6)           7.914Mi ± 0%  +100.99% (p=0.002 n=6)   1.000Mi ± 0%  -74.60% (p=0.002 n=6)
Encode/sample=10000/compression=snappy/encoder=vtprotobuf-2              8.531Mi ± 0%    9.508Mi ± 0%  +11.45% (p=0.002 n=6)          17.148Mi ± 0%  +101.01% (p=0.002 n=6)   2.172Mi ± 0%  -74.54% (p=0.002 n=6)
Encode/sample=10000/compression=zstd/encoder=vtprotobuf-2                4.507Mi ± 0%    5.247Mi ± 0%  +16.42% (p=0.002 n=6)           9.042Mi ± 0%  +100.61% (p=0.002 n=6)   2.011Mi ± 0%  -55.38% (p=0.002 n=6)
geomean                                                                 1006.9Ki         1.099Mi       +11.74%                         1.824Mi        +85.50%                 376.2Ki       -62.64%

                                                            │ prometheus.WriteRequest │ prometheus.WriteRequest+experiments │ prometheus.WriteRequest+experiments+metadata │   io.prometheus.write.v2.Request    │
                                                            │        allocs/op        │  allocs/op   vs base                │     allocs/op       vs base                  │ allocs/op   vs base                 │
Encode/sample=200/compression=/encoder=vtprotobuf-2                        1.000 ± 0%    1.000 ± 0%       ~ (p=1.000 n=6) ¹           1.000 ± 0%         ~ (p=1.000 n=6) ¹   1.000 ± 0%        ~ (p=1.000 n=6) ¹
Encode/sample=200/compression=snappy/encoder=vtprotobuf-2                  2.000 ± 0%    2.000 ± 0%       ~ (p=1.000 n=6) ¹           2.000 ± 0%         ~ (p=1.000 n=6) ¹   2.000 ± 0%        ~ (p=1.000 n=6) ¹
Encode/sample=200/compression=zstd/encoder=vtprotobuf-2                    2.000 ± 0%    2.000 ± 0%       ~ (p=1.000 n=6) ¹           2.000 ± 0%         ~ (p=1.000 n=6) ¹   2.000 ± 0%        ~ (p=1.000 n=6) ¹
Encode/sample=2000/compression=/encoder=vtprotobuf-2                       1.000 ± 0%    1.000 ± 0%       ~ (p=1.000 n=6) ¹           1.000 ± 0%         ~ (p=1.000 n=6) ¹   1.000 ± 0%        ~ (p=1.000 n=6) ¹
Encode/sample=2000/compression=snappy/encoder=vtprotobuf-2                 2.000 ± 0%    2.000 ± 0%       ~ (p=1.000 n=6) ¹           2.000 ± 0%         ~ (p=1.000 n=6) ¹   2.000 ± 0%        ~ (p=1.000 n=6) ¹
Encode/sample=2000/compression=zstd/encoder=vtprotobuf-2                   2.000 ± 0%    2.000 ± 0%       ~ (p=1.000 n=6) ¹          10.000 ± 0%  +400.00% (p=0.002 n=6)     2.000 ± 0%        ~ (p=1.000 n=6) ¹
Encode/sample=10000/compression=/encoder=vtprotobuf-2                      1.000 ± 0%    1.000 ± 0%       ~ (p=1.000 n=6) ¹           1.000 ± 0%         ~ (p=1.000 n=6) ¹   1.000 ± 0%        ~ (p=1.000 n=6) ¹
Encode/sample=10000/compression=snappy/encoder=vtprotobuf-2                2.000 ± 0%    2.000 ± 0%       ~ (p=1.000 n=6) ¹           2.000 ± 0%         ~ (p=1.000 n=6) ¹   2.000 ± 0%        ~ (p=1.000 n=6) ¹
Encode/sample=10000/compression=zstd/encoder=vtprotobuf-2                 15.000 ± 0%   15.000 ± 0%       ~ (p=1.000 n=6) ¹          16.000 ± 0%    +6.67% (p=0.002 n=6)     2.000 ± 0%  -86.67% (p=0.002 n=6)
geomean                                                                    1.986         1.986       +0.00%                           2.392        +20.44%                   1.587       -20.06%
¹ all samples are equal

```

