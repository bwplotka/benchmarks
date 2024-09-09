// Copyright (c) Bartłomiej Płotka @bwplotka
// Licensed under the Apache License 2.0.

package metricsstreaming

import (
	"fmt"
	"testing"

	prompb "github.com/bwplotka/benchmarks/benchmarks/metrics-streaming/io/prometheus/write/v1"
	writev2 "github.com/bwplotka/benchmarks/benchmarks/metrics-streaming/io/prometheus/write/v2"
	"github.com/efficientgo/core/testutil"
	"github.com/golang/snappy"
	"github.com/google/go-cmp/cmp"
	"github.com/klauspost/compress/zstd"
	"github.com/prometheus/prometheus/storage/remote"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
)

type vtprotobufEnhancedMessage interface {
	proto.Message
	MarshalVT() (dAtA []byte, err error)
	UnmarshalVT(dAtA []byte) (err error)
	CloneMessageVT() proto.Message
}

/*
	export bench=bench-encode-09-2024 && go test \
		 -run '^$' -bench '^BenchmarkEncode' \
		 -benchtime 5s -count 6 -cpu 2 -benchmem -timeout 999m \
	 | tee ${bench}.txt
*/
func BenchmarkEncode(b *testing.B) {
	benchmarkEncode(testutil.NewTB(b))
}

func TestBenchmarkEncode(t *testing.T) {
	benchmarkEncode(testutil.NewTB(t))
}

func benchmarkEncode(b testutil.TB) {
	b.Helper()

	for _, tcase := range []struct {
		samples int
		config  generateConfig
	}{
		{samples: 200, config: generateConfig200samples},
		{samples: 2000, config: generateConfig2000samples},
		{samples: 10000, config: generateConfig10000samples},
	} {
		b.Run(fmt.Sprintf("sample=%v", tcase.samples), func(b testutil.TB) {
			batch := generatePrometheusMetricsBatch(tcase.config)
			testutil.Equals(b, tcase.samples, len(batch))

			for _, compr := range []remote.Compression{"", remote.SnappyBlockCompression, "zstd"} {
				b.Run(fmt.Sprintf("compression=%v", compr), func(b testutil.TB) {
					b.Run("proto=prometheus.WriteRequest", func(b testutil.TB) {
						v1Msg := toV1(batch, false, false)
						benchEncoding(b, v1Msg, compr)
					})
					b.Run("proto=prometheus.WriteRequest+experiments", func(b testutil.TB) {
						v1Msg := toV1(batch, false, true)
						benchEncoding(b, v1Msg, compr)
					})
					b.Run("proto=prometheus.WriteRequest+experiments+metadata", func(b testutil.TB) {
						v1Msg := toV1(batch, true, true)
						benchEncoding(b, v1Msg, compr)
					})
					b.Run("proto=io.prometheus.write.v2.Request", func(b testutil.TB) {
						v2Msg := toV2(batch)
						benchEncoding(b, v2Msg, compr)
					})
				})
			}
		})
	}
}

func benchEncoding(b testutil.TB, msg vtprotobufEnhancedMessage, compression remote.Compression) {
	b.Helper()

	b.Run("encoder=protobuf", func(b testutil.TB) {
		b.Skip("let's ignore non-optimized protobuf compiler for now")

		marshalOpts := proto.MarshalOptions{UseCachedSize: true}
		z, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedFastest))
		testutil.Ok(b, err)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N(); i++ {
			out, err := marshalOpts.Marshal(msg)
			testutil.Ok(b, err)

			switch compression {
			case "zstd":
				out = z.EncodeAll(out, nil)
			case remote.SnappyBlockCompression:
				out = snappy.Encode(nil, out)
			default:
				// No compression.
			}

			b.ReportMetric(float64(len(out)), "bytes/message")

			if !b.IsBenchmark() {
				assertDecodability(b, out, msg, compression)
			}
		}
	})
	b.Run("encoder=vtprotobuf", func(b testutil.TB) {
		z, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedFastest))
		testutil.Ok(b, err)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N(); i++ {
			out, err := msg.MarshalVT()
			testutil.Ok(b, err)

			switch compression {
			case "zstd":
				out = z.EncodeAll(out, nil)
			case remote.SnappyBlockCompression:
				out = snappy.Encode(nil, out)
			default:
				// No compression.
			}

			b.ReportMetric(float64(len(out)), "bytes/message")

			if !b.IsBenchmark() {
				assertDecodability(b, out, msg, compression)
			}
		}
	})
}

func assertDecodability(t testing.TB, got []byte, expected vtprotobufEnhancedMessage, compression remote.Compression) {
	t.Helper()

	switch compression {
	case "zstd":
		z, err := zstd.NewReader(nil)
		testutil.Ok(t, err)

		got, err = z.DecodeAll(got, nil)
		testutil.Ok(t, err)
	case remote.SnappyBlockCompression:
		var err error
		got, err = snappy.Decode(nil, got)
		testutil.Ok(t, err)
	default:
		// No compression.
	}

	gotMsg := proto.Clone(expected)
	proto.Reset(gotMsg)

	testutil.Ok(t, proto.Unmarshal(got, gotMsg))
	if diff := cmp.Diff(expected, gotMsg, protocmp.Transform()); diff != "" {
		t.Fatalf("expected the same got: %v, diff: %v", gotMsg, diff)
	}
}

/*
	export bench=bench-decode-09-2024 && go test \
		 -run '^$' -bench '^BenchmarkDecode' \
		 -benchtime 5s -count 6 -cpu 2 -benchmem -timeout 999m \
	 | tee ${bench}.txt
*/
func BenchmarkDecode(b *testing.B) {
	benchmarkDecode(testutil.NewTB(b))
}

func TestBenchmarkDecode(t *testing.T) {
	benchmarkDecode(testutil.NewTB(t))
}

func benchmarkDecode(b testutil.TB) {
	b.Helper()

	for _, tcase := range []struct {
		samples int
		config  generateConfig
	}{
		{samples: 200, config: generateConfig200samples},
		{samples: 2000, config: generateConfig2000samples},
		{samples: 10000, config: generateConfig10000samples},
	} {
		b.Run(fmt.Sprintf("sample=%v", tcase.samples), func(b testutil.TB) {
			batch := generatePrometheusMetricsBatch(tcase.config)
			testutil.Equals(b, tcase.samples, len(batch))

			for _, compr := range []remote.Compression{"", remote.SnappyBlockCompression, "zstd"} {
				b.Run("proto=prometheus.WriteRequest", func(b testutil.TB) {
					v1Msg := toV1(batch, false, false)
					benchDecoding(b, encodeV1(b, v1Msg, compr), func() vtprotobufEnhancedMessage {
						return &prompb.WriteRequest{}
					}, compr)
				})
				b.Run("proto=prometheus.WriteRequest+experiments", func(b testutil.TB) {
					v1Msg := toV1(batch, false, true)
					benchDecoding(b, encodeV1(b, v1Msg, compr), func() vtprotobufEnhancedMessage {
						return &prompb.WriteRequest{}
					}, compr)
				})
				b.Run("proto=prometheus.WriteRequest+experiments+metadata", func(b testutil.TB) {
					v1Msg := toV1(batch, true, true)
					benchDecoding(b, encodeV1(b, v1Msg, compr), func() vtprotobufEnhancedMessage {
						return &prompb.WriteRequest{}
					}, compr)
				})
				b.Run("proto=io.prometheus.write.v2.Request", func(b testutil.TB) {
					v2Msg := toV2(batch)
					v2Encoded, err := proto.Marshal(v2Msg)
					testutil.Ok(b, err)

					switch compr {
					case "zstd":
						z, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedFastest))
						testutil.Ok(b, err)
						v2Encoded = z.EncodeAll(v2Encoded, nil)
					case remote.SnappyBlockCompression:
						v2Encoded = snappy.Encode(nil, v2Encoded)
					default:
						// No compression.
					}
					benchDecoding(b, v2Encoded, func() vtprotobufEnhancedMessage {
						return &writev2.Request{}
					}, compr)
				})
			}
		})
	}
}

func encodeV1(b testutil.TB, v1Msg *prompb.WriteRequest, compr remote.Compression) []byte {
	v1Encoded, err := proto.Marshal(v1Msg)
	testutil.Ok(b, err)

	switch compr {
	case "zstd":
		z, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedFastest))
		testutil.Ok(b, err)
		v1Encoded = z.EncodeAll(v1Encoded, nil)
	case remote.SnappyBlockCompression:
		v1Encoded = snappy.Encode(nil, v1Encoded)
	default:
		// No compression.
	}
	return v1Encoded
}

func benchDecoding(b testutil.TB, encMsg []byte, newMsg func() vtprotobufEnhancedMessage, compression remote.Compression) {
	b.Helper()

	b.Run("encoder=protobuf", func(b testutil.TB) {
		b.Skip("let's ignore non-optimized protobuf compiler for now")

		unmarshalOpts := proto.UnmarshalOptions{}
		z, err := zstd.NewReader(nil)
		testutil.Ok(b, err)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N(); i++ {
			switch compression {
			case "zstd":
				encMsg, err = z.DecodeAll(encMsg, nil)
				testutil.Ok(b, err)
			case remote.SnappyBlockCompression:
				var err error
				encMsg, err = snappy.Decode(nil, encMsg)
				testutil.Ok(b, err)
			default:
				// No compression.
			}

			out := newMsg()
			testutil.Ok(b, unmarshalOpts.Unmarshal(encMsg, out))
		}
	})
	b.Run("encoder=vtprotobuf", func(b testutil.TB) {
		z, err := zstd.NewReader(nil)
		testutil.Ok(b, err)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N(); i++ {
			switch compression {
			case "zstd":
				encMsg, err = z.DecodeAll(encMsg, nil)
				testutil.Ok(b, err)
			case remote.SnappyBlockCompression:
				var err error
				encMsg, err = snappy.Decode(nil, encMsg)
				testutil.Ok(b, err)
			default:
				// No compression.
			}

			out := newMsg()
			testutil.Ok(b, out.UnmarshalVT(encMsg))
		}
	})
}
