// Copyright (c) Bartłomiej Płotka @bwplotka
// Licensed under the Apache License 2.0.

package rustgodecode

import (
	"math"

	writev2 "github.com/bwplotka/benchmarks/benchmarks/metrics-streaming/io/prometheus/write/v2"
	"github.com/golang/snappy"
)

// DecodeSummary captures deterministic counts and a checksum over a decoded Remote Write 2.0 payload
// so Go and Rust tests verify byte-for-byte semantic equivalence.
type DecodeSummary struct {
	UncompressedBytes int     `json:"uncompressed_bytes"`
	CompressedBytes   int     `json:"compressed_bytes"`
	Timeseries        int     `json:"timeseries"`
	Samples           int     `json:"samples"`
	Histograms        int     `json:"histograms"`
	Exemplars         int     `json:"exemplars"`
	Symbols           int     `json:"symbols"`
	ValueBitsXOR      uint64  `json:"value_bits_xor"`
	SampleValueSum    float64 `json:"sample_value_sum"`
}

// DecodeRemoteWrite2 decompresses a Snappy block payload and unmarshals a Remote Write 2.0
// io.prometheus.write.v2.Request using vtprotobuf generated code.
func DecodeRemoteWrite2(compressed []byte) (*writev2.Request, error) {
	raw, err := snappy.Decode(nil, compressed)
	if err != nil {
		return nil, err
	}
	req := &writev2.Request{}
	if err := req.UnmarshalVT(raw); err != nil {
		return nil, err
	}
	return req, nil
}

// SummarizeRemoteWrite2 computes structural counts and sample checksums for a Remote Write 2.0 request.
func SummarizeRemoteWrite2(compressed []byte, req *writev2.Request) (DecodeSummary, error) {
	raw, err := snappy.Decode(nil, compressed)
	if err != nil {
		return DecodeSummary{}, err
	}
	s := DecodeSummary{
		UncompressedBytes: len(raw),
		CompressedBytes:   len(compressed),
		Timeseries:        len(req.Timeseries),
		Symbols:           len(req.Symbols),
	}
	for _, ts := range req.Timeseries {
		s.Samples += len(ts.Samples)
		s.Histograms += len(ts.Histograms)
		s.Exemplars += len(ts.Exemplars)
		for _, smp := range ts.Samples {
			s.ValueBitsXOR ^= math.Float64bits(smp.Value)
			s.SampleValueSum += smp.Value
		}
		for _, h := range ts.Histograms {
			s.ValueBitsXOR ^= math.Float64bits(h.Sum)
		}
	}
	return s, nil
}
