// Copyright (c) Bartłomiej Płotka @bwplotka
// Licensed under the Apache License 2.0.

package rustgodecode

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	writev2 "github.com/bwplotka/benchmarks/benchmarks/metrics-streaming/io/prometheus/write/v2"
	"github.com/golang/snappy"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/histogram"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/model/metadata"
	"google.golang.org/protobuf/proto"
)

type Manifest struct {
	VarintsCount int                      `json:"varints_count"`
	VarintsBytes int                      `json:"varints_bytes"`
	VarintsSum   uint64                   `json:"varints_sum"`
	RemoteWrite  map[string]DecodeSummary `json:"remote_write"`
}

type genConfig struct {
	counters, gauges        int
	classicHistograms       int
	classicHistogramBuckets int
	nativeHistograms        int
}

func (g genConfig) series() int {
	return g.counters + g.gauges + ((g.classicHistogramBuckets + 2) * g.classicHistograms) + g.nativeHistograms
}

type genSeries struct {
	seriesLabels     labels.Labels
	value            float64
	histogram        *histogram.Histogram
	metadata         *metadata.Metadata
	timestamp        int64
	createdTimestamp int64
	exemplarLabels   labels.Labels
}

func deterministicTraceID(i int) string {
	return fmt.Sprintf("01234567-89ab-4cde-8f01-%012x", i)
}

func deterministicLabels(mName string, i int) labels.Labels {
	return labels.FromStrings(
		labels.MetricName, mName,
		"reason", fmt.Sprintf("successerrororso%d", i%5),
		"remote", fmt.Sprintf("thisaasasfndfgdsngaa01%d", i%5),
		"job", fmt.Sprintf("jobPrometh3SDFSDFS1:%d", i%1000),
		"instance", fmt.Sprintf("localhost:%d", i%1000),
		"cluster", "apsdjosnv11231231",
		"namespace", fmt.Sprintf("sdf0pssd234:%d", i%16),
		"workload_type", fmt.Sprintf("deploymentdaemonsetorso:%d", i%4),
		"workload_controller", fmt.Sprintf("somenameofsjfnn2014:%d", i%10),
		"pod", fmt.Sprintf("sdfkpsdjgpsdf213=21:%d", i%30),
		"team", "dsffsjfjs[p=1=1==124",
	)
}

func deterministicHelp(mName string) string {
	return fmt.Sprintf("A %s of the request duration for HTTP requests, segmented by various dimensions. This metric provides detailed insights into the performance of your HTTP endpoints.", mName)
}

func generateDeterministicBatch(cfg genConfig) []genSeries {
	ret := make([]genSeries, cfg.series())
	i := 0
	for range cfg.counters {
		mName := fmt.Sprintf("metric_my_own_counter_bytes%d_total", i%10)
		ret[i] = genSeries{
			seriesLabels:     deterministicLabels(mName, i),
			value:            0.123093 + float64(i),
			metadata:         &metadata.Metadata{Type: model.MetricTypeCounter, Help: deterministicHelp(mName), Unit: "bytes"},
			timestamp:        1724844902198,
			createdTimestamp: 1724837702198,
		}
		if i%2 == 0 {
			ret[i].exemplarLabels = labels.FromStrings("trace_id", deterministicTraceID(i))
		}
		i++
	}
	for range cfg.gauges {
		mName := fmt.Sprintf("metric_my_own_gauge_operations%d", i%10)
		ret[i] = genSeries{
			seriesLabels: deterministicLabels(mName, i),
			value:        120412431 + float64(i),
			metadata:     &metadata.Metadata{Type: model.MetricTypeGauge, Help: deterministicHelp(mName), Unit: "operations"},
			timestamp:    1724844902198,
		}
		if i%2 == 0 {
			ret[i].exemplarLabels = labels.FromStrings("trace_id", deterministicTraceID(i))
		}
		i++
	}
	for h := range cfg.classicHistograms {
		baseName := fmt.Sprintf("metric_my_own_classic_histogram_seconds%d", h%10)
		meta := &metadata.Metadata{Type: model.MetricTypeHistogram, Help: deterministicHelp(baseName), Unit: "seconds"}
		traceID := deterministicTraceID(100000 + h)
		var lastCount float64
		for b := range cfg.classicHistogramBuckets {
			mName := baseName + "_bucket"
			ret[i] = genSeries{
				seriesLabels:     append(deterministicLabels(mName, h), labels.Label{Name: "le", Value: fmt.Sprintf("10023%d.0", h%10+b*100)}),
				value:            1123 + 10*float64(i),
				metadata:         meta,
				timestamp:        1724844902198,
				createdTimestamp: 1724837702198,
			}
			lastCount = ret[i].value
			if i%2 == 0 {
				ret[i].exemplarLabels = labels.FromStrings("trace_id", traceID)
			}
			i++
		}
		{
			mName := baseName + "_sum"
			ret[i] = genSeries{
				seriesLabels:     deterministicLabels(mName, h),
				value:            121.2123 + float64(i),
				metadata:         meta,
				timestamp:        1724844902198,
				createdTimestamp: 1724837702198,
			}
			if i%2 == 0 {
				ret[i].exemplarLabels = labels.FromStrings("trace_id", traceID)
			}
			i++
		}
		{
			mName := baseName + "_count"
			ret[i] = genSeries{
				seriesLabels:     deterministicLabels(mName, h),
				value:            lastCount,
				metadata:         meta,
				timestamp:        1724844902198,
				createdTimestamp: 1724837702198,
			}
			if i%2 == 0 {
				ret[i].exemplarLabels = labels.FromStrings("trace_id", traceID)
			}
			i++
		}
	}
	for _, hist := range histogram.GenerateBigTestHistograms(cfg.nativeHistograms, 100) {
		mName := fmt.Sprintf("metric_my_own_native_histogram_seconds%d", i%10)
		ret[i] = genSeries{
			seriesLabels:     deterministicLabels(mName, i),
			metadata:         &metadata.Metadata{Type: model.MetricTypeHistogram, Help: deterministicHelp(mName), Unit: "seconds"},
			timestamp:        1724844902198,
			createdTimestamp: 1724837702198,
			histogram:        hist,
		}
		if i%2 == 0 {
			ret[i].exemplarLabels = labels.FromStrings("trace_id", deterministicTraceID(i))
		}
		i++
	}
	return ret
}

func buildV2(batch []genSeries) *writev2.Request {
	s := writev2.NewSymbolTable()
	ret := &writev2.Request{
		Timeseries: make([]*writev2.TimeSeries, len(batch)),
	}
	for i, ts := range batch {
		ret.Timeseries[i] = &writev2.TimeSeries{
			LabelsRefs: s.SymbolizeLabels(ts.seriesLabels, nil),
			Metadata: &writev2.Metadata{
				HelpRef: s.Symbolize(ts.metadata.Help),
				UnitRef: s.Symbolize(ts.metadata.Unit),
				Type:    writev2.FromMetadataType(ts.metadata.Type),
			},
		}
		if ts.histogram != nil {
			ret.Timeseries[i].Histograms = []*writev2.Histogram{writev2.FromIntHistogram(ts.createdTimestamp, ts.timestamp, ts.histogram)}
		} else {
			ret.Timeseries[i].Samples = []*writev2.Sample{{
				Value:          ts.value,
				Timestamp:      ts.timestamp,
				StartTimestamp: ts.createdTimestamp,
			}}
		}
		if len(ts.exemplarLabels) > 0 {
			ret.Timeseries[i].Exemplars = []*writev2.Exemplar{{
				LabelsRefs: s.SymbolizeLabels(ts.exemplarLabels, nil),
				Value:      12414,
				Timestamp:  ts.timestamp,
			}}
		}
	}
	ret.Symbols = s.Symbols()
	return ret
}

func generateDeterministicVarints(n int) ([]uint64, uint64) {
	vals := make([]uint64, n)
	var sum uint64
	for i := range n {
		var v uint64
		switch i % 10 {
		case 0, 1, 2, 3:
			// 1-byte varints (0..127), typical for symbol table indices.
			v = uint64((i * 37) & 0x7f)
		case 4, 5, 6:
			// 2-byte varints (128..16383), typical for larger symbol tables and bucket spans.
			v = 128 + uint64((i*257)&0x3f00)
		case 7, 8:
			// 4-byte varints, typical for count deltas and millisecond offsets.
			v = 16384 + uint64(i)*131071
		default:
			// 6-byte varints, typical for millisecond Unix timestamps.
			v = 1724844902198 + uint64(i)*15000
		}
		vals[i] = v
		sum += v
	}
	return vals, sum
}

func ensureFixtures(tb testing.TB) Manifest {
	tb.Helper()

	manifestPath := filepath.Join("testdata", "manifest.json")
	if raw, err := os.ReadFile(manifestPath); err == nil {
		var m Manifest
		if err := json.Unmarshal(raw, &m); err == nil && len(m.RemoteWrite) == 2 {
			return m
		}
	}

	if err := os.MkdirAll("testdata", 0o755); err != nil {
		tb.Fatalf("mkdir testdata: %v", err)
	}

	vals, varintsSum := generateDeterministicVarints(2000)
	varintBytes := EncodeVarints(vals)
	if err := os.WriteFile(filepath.Join("testdata", "varints_2000.bin"), varintBytes, 0o644); err != nil {
		tb.Fatalf("write varints fixture: %v", err)
	}

	configs := []struct {
		samples int
		cfg     genConfig
	}{
		{
			samples: 200,
			cfg:     genConfig{counters: 50, gauges: 40, classicHistograms: 10, classicHistogramBuckets: 8, nativeHistograms: 10},
		},
		{
			samples: 2000,
			cfg:     genConfig{counters: 500, gauges: 400, classicHistograms: 100, classicHistogramBuckets: 8, nativeHistograms: 100},
		},
	}

	rwManifest := make(map[string]DecodeSummary, 2)
	for _, c := range configs {
		batch := generateDeterministicBatch(c.cfg)
		if len(batch) != c.samples {
			tb.Fatalf("expected %d series, got %d", c.samples, len(batch))
		}

		v2Msg := buildV2(batch)
		v2Raw, err := proto.MarshalOptions{Deterministic: true}.Marshal(v2Msg)
		if err != nil {
			tb.Fatalf("marshal v2: %v", err)
		}
		v2Snappy := snappy.Encode(nil, v2Raw)
		v2Name := fmt.Sprintf("rw2_%d.snappy", c.samples)
		if err := os.WriteFile(filepath.Join("testdata", v2Name), v2Snappy, 0o644); err != nil {
			tb.Fatalf("write %s: %v", v2Name, err)
		}
		v2Sum, err := SummarizeRemoteWrite2(v2Snappy, v2Msg)
		if err != nil {
			tb.Fatalf("summarize v2: %v", err)
		}
		rwManifest[v2Name] = v2Sum
	}

	m := Manifest{
		VarintsCount: len(vals),
		VarintsBytes: len(varintBytes),
		VarintsSum:   varintsSum,
		RemoteWrite:  rwManifest,
	}
	encodedManifest, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		tb.Fatalf("marshal manifest: %v", err)
	}
	if err := os.WriteFile(manifestPath, append(encodedManifest, '\n'), 0o644); err != nil {
		tb.Fatalf("write manifest: %v", err)
	}
	return m
}
