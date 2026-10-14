// Copyright (c) Bartłomiej Płotka @bwplotka
// Licensed under the Apache License 2.0.

package metricsstreaming

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"testing"

	colmetricsv1 "github.com/bwplotka/benchmarks/benchmarks/metrics-streaming/io/opentelemetry/proto/collector/metrics/v1"
	prompb "github.com/bwplotka/benchmarks/benchmarks/metrics-streaming/io/prometheus/write/v1"
	writev2 "github.com/bwplotka/benchmarks/benchmarks/metrics-streaming/io/prometheus/write/v2"
	"github.com/efficientgo/core/testutil"
	"github.com/golang/snappy"
	"github.com/google/go-cmp/cmp"
	"github.com/klauspost/compress/gzip"
	"github.com/klauspost/compress/zstd"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
)

type Compression string

const (
	CompressionNone   Compression = "none"
	CompressionSnappy Compression = "snappy"
	CompressionZstd   Compression = "zstd"
	CompressionGzip   Compression = "gzip"
)

var allCompressions = []Compression{
	CompressionNone,
	CompressionSnappy,
	CompressionZstd,
	CompressionGzip,
}

type vtprotobufEnhancedMessage interface {
	proto.Message
	MarshalVT() (dAtA []byte, err error)
	UnmarshalVT(dAtA []byte) (err error)
	CloneMessageVT() proto.Message
}

// Recommended CLI invocation:
//
//	export bench=bench-encode-10-2026 && go test \
//	    -run '^$' -bench '^BenchmarkEncode' \
//	    -benchtime 5s -count 6 -cpu 2 -timeout 999m -benchmem \
//	| tee ${bench}.txt
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
			nhcbBatch := convertClassicToCustom(batch)

			for _, compr := range allCompressions {
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
					b.Run("proto=io.prometheus.write.v2.Request+nhcb", func(b testutil.TB) {
						v2Msg := toV2(nhcbBatch)
						benchEncoding(b, v2Msg, compr)
					})
					b.Run("proto=opentelemetry.proto.collector.metrics.v1.ExportMetricsServiceRequest", func(b testutil.TB) {
						otlpMsg := toOTLP(nhcbBatch)
						benchEncoding(b, otlpMsg, compr)
					})
				})
			}
		})
	}
}

func benchEncoding(b testutil.TB, msg vtprotobufEnhancedMessage, compression Compression) {
	b.Helper()

	b.Run("encoder=protobuf", func(b testutil.TB) {
		b.Skip("let's ignore non-optimized protobuf compiler for now")

		marshalOpts := proto.MarshalOptions{UseCachedSize: true}
		z, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedFastest))
		testutil.Ok(b, err)
		gz, err := gzip.NewWriterLevel(nil, gzip.DefaultCompression)
		testutil.Ok(b, err)

		b.ReportAllocs()
		b.ResetTimer()
		for range b.N() {
			out, err := marshalOpts.Marshal(msg)
			testutil.Ok(b, err)

			out = compressBytes(b, out, compression, z, gz)
			b.ReportMetric(float64(len(out)), "bytes/message")

			if !b.IsBenchmark() {
				assertDecodability(b, out, msg, compression)
			}
		}
	})
	b.Run("encoder=vtprotobuf", func(b testutil.TB) {
		z, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedFastest))
		testutil.Ok(b, err)
		gz, err := gzip.NewWriterLevel(nil, gzip.DefaultCompression)
		testutil.Ok(b, err)

		b.ReportAllocs()
		b.ResetTimer()
		for range b.N() {
			out, err := msg.MarshalVT()
			testutil.Ok(b, err)

			out = compressBytes(b, out, compression, z, gz)
			b.ReportMetric(float64(len(out)), "bytes/message")

			if !b.IsBenchmark() {
				assertDecodability(b, out, msg, compression)
			}
		}
	})
}

func compressBytes(tb testing.TB, raw []byte, compression Compression, z *zstd.Encoder, gz *gzip.Writer) []byte {
	tb.Helper()

	switch compression {
	case CompressionZstd:
		return z.EncodeAll(raw, nil)
	case CompressionSnappy:
		return snappy.Encode(nil, raw)
	case CompressionGzip:
		var buf bytes.Buffer
		gz.Reset(&buf)
		_, err := gz.Write(raw)
		testutil.Ok(tb, err)
		testutil.Ok(tb, gz.Close())
		return buf.Bytes()
	case CompressionNone, "":
		return raw
	default:
		tb.Fatalf("unknown compression: %v", compression)
		return nil
	}
}

func decompressBytes(tb testing.TB, enc []byte, compression Compression, z *zstd.Decoder, gz *gzip.Reader) []byte {
	tb.Helper()

	switch compression {
	case CompressionZstd:
		out, err := z.DecodeAll(enc, nil)
		testutil.Ok(tb, err)
		return out
	case CompressionSnappy:
		out, err := snappy.Decode(nil, enc)
		testutil.Ok(tb, err)
		return out
	case CompressionGzip:
		testutil.Ok(tb, gz.Reset(bytes.NewReader(enc)))
		out, err := io.ReadAll(gz)
		testutil.Ok(tb, err)
		testutil.Ok(tb, gz.Close())
		return out
	case CompressionNone, "":
		return enc
	default:
		tb.Fatalf("unknown compression: %v", compression)
		return nil
	}
}

func assertDecodability(t testing.TB, got []byte, expected vtprotobufEnhancedMessage, compression Compression) {
	t.Helper()

	z, err := zstd.NewReader(nil)
	testutil.Ok(t, err)
	var gz gzip.Reader

	got = decompressBytes(t, got, compression, z, &gz)

	gotMsg := proto.Clone(expected)
	proto.Reset(gotMsg)

	testutil.Ok(t, proto.Unmarshal(got, gotMsg))
	if !proto.Equal(expected, gotMsg) {
		diff := cmp.Diff(expected, gotMsg, protocmp.Transform())
		t.Fatalf("expected the same got: %v, diff: %v", gotMsg, diff)
	}
}

// Recommended CLI invocation:
//
//	export bench=bench-decode-10-2026 && go test \
//	    -run '^$' -bench '^BenchmarkDecode' \
//	    -benchtime 5s -count 6 -cpu 2 -timeout 999m -benchmem \
//	| tee ${bench}.txt
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
			nhcbBatch := convertClassicToCustom(batch)

			for _, compr := range allCompressions {
				b.Run(fmt.Sprintf("compression=%v", compr), func(b testutil.TB) {
					b.Run("proto=prometheus.WriteRequest", func(b testutil.TB) {
						v1Msg := toV1(batch, false, false)
						benchDecoding(b, encodeMsg(b, v1Msg, compr), func() vtprotobufEnhancedMessage {
							return &prompb.WriteRequest{}
						}, compr)
					})
					b.Run("proto=prometheus.WriteRequest+experiments", func(b testutil.TB) {
						v1Msg := toV1(batch, false, true)
						benchDecoding(b, encodeMsg(b, v1Msg, compr), func() vtprotobufEnhancedMessage {
							return &prompb.WriteRequest{}
						}, compr)
					})
					b.Run("proto=prometheus.WriteRequest+experiments+metadata", func(b testutil.TB) {
						v1Msg := toV1(batch, true, true)
						benchDecoding(b, encodeMsg(b, v1Msg, compr), func() vtprotobufEnhancedMessage {
							return &prompb.WriteRequest{}
						}, compr)
					})
					b.Run("proto=io.prometheus.write.v2.Request", func(b testutil.TB) {
						v2Msg := toV2(batch)
						benchDecoding(b, encodeMsg(b, v2Msg, compr), func() vtprotobufEnhancedMessage {
							return &writev2.Request{}
						}, compr)
					})
					b.Run("proto=io.prometheus.write.v2.Request+nhcb", func(b testutil.TB) {
						v2Msg := toV2(nhcbBatch)
						benchDecoding(b, encodeMsg(b, v2Msg, compr), func() vtprotobufEnhancedMessage {
							return &writev2.Request{}
						}, compr)
					})
					b.Run("proto=opentelemetry.proto.collector.metrics.v1.ExportMetricsServiceRequest", func(b testutil.TB) {
						otlpMsg := toOTLP(nhcbBatch)
						benchDecoding(b, encodeMsg(b, otlpMsg, compr), func() vtprotobufEnhancedMessage {
							return &colmetricsv1.ExportMetricsServiceRequest{}
						}, compr)
					})
				})
			}
		})
	}
}

func encodeMsg(b testutil.TB, msg proto.Message, compr Compression) []byte {
	b.Helper()

	encoded, err := proto.Marshal(msg)
	testutil.Ok(b, err)

	z, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedFastest))
	testutil.Ok(b, err)
	gz, err := gzip.NewWriterLevel(nil, gzip.DefaultCompression)
	testutil.Ok(b, err)

	return compressBytes(b, encoded, compr, z, gz)
}

func benchDecoding(b testutil.TB, encMsg []byte, newMsg func() vtprotobufEnhancedMessage, compression Compression) {
	b.Helper()

	b.Run("encoder=protobuf", func(b testutil.TB) {
		b.Skip("let's ignore non-optimized protobuf compiler for now")

		unmarshalOpts := proto.UnmarshalOptions{}
		z, err := zstd.NewReader(nil)
		testutil.Ok(b, err)
		var gz gzip.Reader

		b.ReportAllocs()
		b.ResetTimer()
		for range b.N() {
			decMsg := decompressBytes(b, encMsg, compression, z, &gz)
			out := newMsg()
			testutil.Ok(b, unmarshalOpts.Unmarshal(decMsg, out))
		}
	})
	b.Run("encoder=vtprotobuf", func(b testutil.TB) {
		z, err := zstd.NewReader(nil)
		testutil.Ok(b, err)
		var gz gzip.Reader

		b.ReportAllocs()
		b.ResetTimer()
		for range b.N() {
			decMsg := decompressBytes(b, encMsg, compression, z, &gz)
			out := newMsg()
			testutil.Ok(b, out.UnmarshalVT(decMsg))
		}
	})
}

// Recommended CLI invocation:
//
//	export bench=bench-transport-10-2026 && go test \
//	    -run '^$' -bench '^BenchmarkTransport' \
//	    -benchtime 5s -count 6 -cpu 2 -timeout 999m -benchmem \
//	| tee ${bench}.txt
func BenchmarkTransport(b *testing.B) {
	benchmarkTransport(testutil.NewTB(b))
}

func TestBenchmarkTransport(t *testing.T) {
	benchmarkTransport(testutil.NewTB(t))
}

func benchmarkTransport(b testutil.TB) {
	b.Helper()

	registerGRPCCompressors()

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
			nhcbBatch := convertClassicToCustom(batch)

			v2Msg := toV2(batch)
			v2NHCBMsg := toV2(nhcbBatch)
			otlpMsg := toOTLP(nhcbBatch)

			for _, compr := range allCompressions {
				b.Run(fmt.Sprintf("compression=%v", compr), func(b testutil.TB) {
					for _, httpVer := range []string{"http1", "http2"} {
						b.Run(fmt.Sprintf("proto=io.prometheus.write.v2.Request/transport=%v", httpVer), func(b testutil.TB) {
							benchHTTPTransport(b, v2Msg, func() vtprotobufEnhancedMessage {
								return &writev2.Request{}
							}, nil, compr, httpVer == "http2")
						})
						b.Run(fmt.Sprintf("proto=io.prometheus.write.v2.Request+nhcb/transport=%v", httpVer), func(b testutil.TB) {
							benchHTTPTransport(b, v2NHCBMsg, func() vtprotobufEnhancedMessage {
								return &writev2.Request{}
							}, nil, compr, httpVer == "http2")
						})
						b.Run(fmt.Sprintf("proto=opentelemetry.proto.collector.metrics.v1.ExportMetricsServiceRequest/transport=%v", httpVer), func(b testutil.TB) {
							benchHTTPTransport(b, otlpMsg, func() vtprotobufEnhancedMessage {
								return &colmetricsv1.ExportMetricsServiceRequest{}
							}, &colmetricsv1.ExportMetricsServiceResponse{}, compr, httpVer == "http2")
						})
					}
					b.Run("proto=opentelemetry.proto.collector.metrics.v1.ExportMetricsServiceRequest/transport=grpc", func(b testutil.TB) {
						benchGRPCTransport(b, otlpMsg, compr)
					})
				})
			}
		})
	}
}

func benchHTTPTransport(
	b testutil.TB,
	msg vtprotobufEnhancedMessage,
	newReqMsg func() vtprotobufEnhancedMessage,
	respMsg vtprotobufEnhancedMessage,
	compression Compression,
	useHTTP2 bool,
) {
	b.Helper()

	srvZ, err := zstd.NewReader(nil)
	testutil.Ok(b, err)
	var (
		srvGz gzip.Reader
		srvMu sync.Mutex
	)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		srvMu.Lock()
		defer srvMu.Unlock()

		encBody, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		decBody := decompressBytes(b, encBody, Compression(r.Header.Get("Content-Encoding")), srvZ, &srvGz)
		reqOut := newReqMsg()
		if err := reqOut.UnmarshalVT(decBody); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if respMsg != nil {
			respBytes, err := respMsg.MarshalVT()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/x-protobuf")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(respBytes)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	l, err := net.Listen("tcp", "127.0.0.1:0")
	testutil.Ok(b, err)

	var srvHandler http.Handler = handler
	if useHTTP2 {
		srvHandler = h2c.NewHandler(handler, &http2.Server{})
	}
	srv := &http.Server{Handler: srvHandler}
	go func() { _ = srv.Serve(l) }()
	b.Cleanup(func() { _ = srv.Close() })

	var rt http.RoundTripper
	if useHTTP2 {
		rt = &http2.Transport{
			AllowHTTP: true,
			DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, network, addr)
			},
		}
	} else {
		rt = &http.Transport{
			MaxIdleConnsPerHost: 10,
		}
	}
	client := &http.Client{Transport: rt}
	url := "http://" + l.Addr().String() + "/v1/metrics"

	z, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedFastest))
	testutil.Ok(b, err)
	gz, err := gzip.NewWriterLevel(nil, gzip.DefaultCompression)
	testutil.Ok(b, err)

	ctx := context.Background()

	sendOnce := func() int {
		out, err := msg.MarshalVT()
		testutil.Ok(b, err)
		out = compressBytes(b, out, compression, z, gz)

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(out))
		testutil.Ok(b, err)
		req.Header.Set("Content-Type", "application/x-protobuf")
		req.Header.Set("Content-Encoding", string(compression))

		resp, err := client.Do(req)
		testutil.Ok(b, err)
		if respMsg != nil {
			respBody, err := io.ReadAll(resp.Body)
			testutil.Ok(b, err)
			testutil.Equals(b, http.StatusOK, resp.StatusCode)
			decResp := &colmetricsv1.ExportMetricsServiceResponse{}
			testutil.Ok(b, decResp.UnmarshalVT(respBody))
		} else {
			_, err = io.Copy(io.Discard, resp.Body)
			testutil.Ok(b, err)
			testutil.Equals(b, http.StatusNoContent, resp.StatusCode)
		}
		testutil.Ok(b, resp.Body.Close())
		return len(out)
	}

	// Warm up connection before starting the benchmark timer.
	_ = sendOnce()

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N() {
		n := sendOnce()
		b.ReportMetric(float64(n), "bytes/message")
	}
}

type noopOTLPServer struct {
	colmetricsv1.UnimplementedMetricsServiceServer
}

func (s *noopOTLPServer) Export(_ context.Context, _ *colmetricsv1.ExportMetricsServiceRequest) (*colmetricsv1.ExportMetricsServiceResponse, error) {
	return &colmetricsv1.ExportMetricsServiceResponse{}, nil
}

func benchGRPCTransport(b testutil.TB, msg *colmetricsv1.ExportMetricsServiceRequest, compression Compression) {
	b.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	testutil.Ok(b, err)

	insec := insecure.NewCredentials()
	grpcSrv := grpc.NewServer(
		grpc.Creds(insec),
		grpc.ForceServerCodec(vtprotoCodec{}),
		grpc.MaxRecvMsgSize(64*1024*1024),
	)
	colmetricsv1.RegisterMetricsServiceServer(grpcSrv, &noopOTLPServer{})
	go func() { _ = grpcSrv.Serve(l) }()
	b.Cleanup(grpcSrv.Stop)

	var callOpts []grpc.CallOption
	callOpts = append(callOpts, grpc.ForceCodec(vtprotoCodec{}))
	if compression != CompressionNone && compression != "" {
		callOpts = append(callOpts, grpc.UseCompressor(string(compression)))
	}

	cc, err := grpc.NewClient(
		l.Addr().String(),
		grpc.WithTransportCredentials(insec),
		grpc.WithDefaultCallOptions(callOpts...),
	)
	testutil.Ok(b, err)
	b.Cleanup(func() { _ = cc.Close() })

	client := colmetricsv1.NewMetricsServiceClient(cc)
	ctx := context.Background()

	// Warm up connection and compute wire payload size for reporting.
	_, err = client.Export(ctx, msg)
	testutil.Ok(b, err)
	encodedSize := len(encodeMsg(b, msg, compression))

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N() {
		_, err := client.Export(ctx, msg)
		testutil.Ok(b, err)
		b.ReportMetric(float64(encodedSize), "bytes/message")
	}
}

type vtprotoCodec struct{}

func (vtprotoCodec) Name() string { return "proto" }

func (vtprotoCodec) Marshal(v any) ([]byte, error) {
	if vt, ok := v.(vtprotobufEnhancedMessage); ok {
		return vt.MarshalVT()
	}
	if pm, ok := v.(proto.Message); ok {
		return proto.Marshal(pm)
	}
	return nil, fmt.Errorf("unexpected type %T", v)
}

func (vtprotoCodec) Unmarshal(data []byte, v any) error {
	if vt, ok := v.(vtprotobufEnhancedMessage); ok {
		return vt.UnmarshalVT(data)
	}
	if pm, ok := v.(proto.Message); ok {
		return proto.Unmarshal(data, pm)
	}
	return fmt.Errorf("unexpected type %T", v)
}

var registerCompressorsOnce sync.Once

func registerGRPCCompressors() {
	registerCompressorsOnce.Do(func() {
		encoding.RegisterCompressor(&snappyCompressor{})
		encoding.RegisterCompressor(&zstdCompressor{})
		encoding.RegisterCompressor(&gzipCompressor{})
	})
}

type snappyCompressor struct {
	writerPool sync.Pool
}

func (s *snappyCompressor) Name() string { return string(CompressionSnappy) }

func (s *snappyCompressor) Compress(w io.Writer) (io.WriteCloser, error) {
	sw, ok := s.writerPool.Get().(*snappyBlockWriteCloser)
	if !ok {
		sw = &snappyBlockWriteCloser{parent: s}
	}
	sw.w = w
	sw.buf.Reset()
	return sw, nil
}

func (s *snappyCompressor) Decompress(r io.Reader) (io.Reader, error) {
	enc, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	dec, err := snappy.Decode(nil, enc)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(dec), nil
}

type snappyBlockWriteCloser struct {
	parent *snappyCompressor
	w      io.Writer
	buf    bytes.Buffer
}

func (s *snappyBlockWriteCloser) Write(p []byte) (int, error) {
	return s.buf.Write(p)
}

func (s *snappyBlockWriteCloser) Close() error {
	enc := snappy.Encode(nil, s.buf.Bytes())
	_, err := s.w.Write(enc)
	s.w = nil
	s.parent.writerPool.Put(s)
	return err
}

type zstdCompressor struct {
	writerPool sync.Pool
	decPool    sync.Pool
}

func (z *zstdCompressor) Name() string { return string(CompressionZstd) }

func (z *zstdCompressor) Compress(w io.Writer) (io.WriteCloser, error) {
	zw, ok := z.writerPool.Get().(*zstdBlockWriteCloser)
	if !ok {
		enc, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedFastest))
		if err != nil {
			return nil, err
		}
		zw = &zstdBlockWriteCloser{parent: z, enc: enc}
	}
	zw.w = w
	zw.buf.Reset()
	return zw, nil
}

func (z *zstdCompressor) Decompress(r io.Reader) (io.Reader, error) {
	dec, ok := z.decPool.Get().(*zstd.Decoder)
	if !ok {
		var err error
		dec, err = zstd.NewReader(nil)
		if err != nil {
			return nil, err
		}
	}
	defer z.decPool.Put(dec)

	encBytes, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	decBytes, err := dec.DecodeAll(encBytes, nil)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(decBytes), nil
}

type zstdBlockWriteCloser struct {
	parent *zstdCompressor
	enc    *zstd.Encoder
	w      io.Writer
	buf    bytes.Buffer
}

func (z *zstdBlockWriteCloser) Write(p []byte) (int, error) {
	return z.buf.Write(p)
}

func (z *zstdBlockWriteCloser) Close() error {
	out := z.enc.EncodeAll(z.buf.Bytes(), nil)
	_, err := z.w.Write(out)
	z.w = nil
	z.parent.writerPool.Put(z)
	return err
}

type gzipCompressor struct {
	writerPool sync.Pool
	readerPool sync.Pool
}

func (g *gzipCompressor) Name() string { return string(CompressionGzip) }

func (g *gzipCompressor) Compress(w io.Writer) (io.WriteCloser, error) {
	gz, ok := g.writerPool.Get().(*gzip.Writer)
	if !ok {
		var err error
		gz, err = gzip.NewWriterLevel(w, gzip.DefaultCompression)
		if err != nil {
			return nil, err
		}
	} else {
		gz.Reset(w)
	}
	return &pooledGzipWriter{parent: g, Writer: gz}, nil
}

func (g *gzipCompressor) Decompress(r io.Reader) (io.Reader, error) {
	gz, ok := g.readerPool.Get().(*gzip.Reader)
	if !ok {
		var err error
		gz, err = gzip.NewReader(r)
		if err != nil {
			return nil, err
		}
	} else if err := gz.Reset(r); err != nil {
		return nil, err
	}
	defer g.readerPool.Put(gz)

	decBytes, err := io.ReadAll(gz)
	if err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	return bytes.NewReader(decBytes), nil
}

type pooledGzipWriter struct {
	parent *gzipCompressor
	*gzip.Writer
}

func (p *pooledGzipWriter) Close() error {
	err := p.Writer.Close()
	p.parent.writerPool.Put(p.Writer)
	return err
}
