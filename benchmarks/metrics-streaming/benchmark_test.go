// Copyright (c) Bartłomiej Płotka @bwplotka
// Licensed under the Apache License 2.0.

package metricsstreaming

import (
	"testing"

	prompb "github.com/bwplotka/benchmarks/benchmarks/metrics-streaming/io/prometheus/write/v1"
	writev2 "github.com/bwplotka/benchmarks/benchmarks/metrics-streaming/io/prometheus/write/v2"
	"google.golang.org/protobuf/proto"
)

type vtprotobufEnhancedMessage interface {
	proto.Message
	MarshalVT() (dAtA []byte, err error)
	UnmarshalVT(dAtA []byte) (err error)
}

// Test things https://github.com/efficientgo/core/blob/main/testutil/testorbench.go

func BenchmarkEncode_2000samples(b *testing.B) {
	batch := generatePrometheusMetricsBatch(generateConfig2000samples)
	if len(batch) != 2000 {
		b.Fatalf("expected input batch with 2000 samples, got %v", len(batch))
	}

	b.Run("prometheus.WriteRequest", func(b *testing.B) {
		v1Msg := toV1(batch)
		benchEncoding(b, v1Msg)
	})
	b.Run("io.prometheus.write.v2.Request", func(b *testing.B) {
		v2Msg := toV2(batch)
		benchEncoding(b, v2Msg)
	})
}

func benchEncoding(b *testing.B, msg vtprotobufEnhancedMessage) {
	b.Helper()

	b.Run("protobuf", func(b *testing.B) {
		marshalOpts := proto.MarshalOptions{UseCachedSize: true}

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			out, err := marshalOpts.Marshal(msg)
			if err != nil {
				b.Fatal(err)
			}
			b.ReportMetric(float64(len(out)), "bytes/message")
		}
	})
	b.Run("vtprotobuf", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			out, err := msg.MarshalVT()
			if err != nil {
				b.Fatal(err)
			}
			b.ReportMetric(float64(len(out)), "bytes/message")
		}
	})
}

func BenchmarkDecode_2000samples(b *testing.B) {
	batch := generatePrometheusMetricsBatch(generateConfig2000samples)
	if len(batch) != 2000 {
		b.Fatalf("expected input batch with 2000 samples, got %v", len(batch))
	}

	b.Run("prometheus.WriteRequest", func(b *testing.B) {
		v1Msg := toV1(batch)
		v1Encoded, err := proto.Marshal(v1Msg)
		if err != nil {
			b.Fatal(err)
		}
		benchDecoding(b, v1Encoded, func() vtprotobufEnhancedMessage {
			return &prompb.WriteRequest{}
		})
	})
	b.Run("io.prometheus.write.v2.Request", func(b *testing.B) {
		v2Msg := toV2(batch)
		v2Encoded, err := proto.Marshal(v2Msg)
		if err != nil {
			b.Fatal(err)
		}
		benchDecoding(b, v2Encoded, func() vtprotobufEnhancedMessage {
			return &writev2.Request{}
		})
	})
}

func benchDecoding(b *testing.B, encMsg []byte, newMsg func() vtprotobufEnhancedMessage) {
	b.Helper()

	b.Run("protobuf", func(b *testing.B) {
		unmarshalOpts := proto.UnmarshalOptions{}

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			out := newMsg()
			if err := unmarshalOpts.Unmarshal(encMsg, out); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("vtprotobuf", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			out := newMsg()
			if err := out.UnmarshalVT(encMsg); err != nil {
				b.Fatal(err)
			}

		}
	})
}
