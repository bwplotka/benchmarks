// Copyright (c) Bartłomiej Płotka @bwplotka
// Licensed under the Apache License 2.0.

package metricsstreaming

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	colmetricsv1 "github.com/bwplotka/benchmarks/benchmarks/metrics-streaming/io/opentelemetry/proto/collector/metrics/v1"
	commonv1 "github.com/bwplotka/benchmarks/benchmarks/metrics-streaming/io/opentelemetry/proto/common/v1"
	metricsv1 "github.com/bwplotka/benchmarks/benchmarks/metrics-streaming/io/opentelemetry/proto/metrics/v1"
	resourcev1 "github.com/bwplotka/benchmarks/benchmarks/metrics-streaming/io/opentelemetry/proto/resource/v1"
	prompb "github.com/bwplotka/benchmarks/benchmarks/metrics-streaming/io/prometheus/write/v1"
	writev2 "github.com/bwplotka/benchmarks/benchmarks/metrics-streaming/io/prometheus/write/v2"
	"github.com/efficientgo/core/testutil"
	"github.com/google/uuid"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/histogram"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/model/metadata"
	"google.golang.org/protobuf/proto"
)

// Trying to mimic variability of string here we would see in practice, while keeping it deterministic.
func seriesLabels(mName string, i int) labels.Labels {
	return labels.FromStrings(
		labels.MetricName, mName,
		"reason", fmt.Sprintf("successerrororso%v", i%5),
		"remote", fmt.Sprintf("thisaasasfndfgdsngaa01%v", i%5),
		"job", fmt.Sprintf("jobPrometh3SDFSDFS1:%v", i%1000),
		"instance", fmt.Sprintf("localhost:%v", i%1000),
		"cluster", "apsdjosnv11231231",
		"namespace", fmt.Sprintf("sdf0pssd234:%v", i%16),
		"workload_type", fmt.Sprintf("deploymentdaemonsetorso:%v", i%4),
		"workload_controller", fmt.Sprintf("somenameofsjfnn2014:%v", i%10),
		"pod", fmt.Sprintf("sdfkpsdjgpsdf213=21:%v", i%30),
		"team", "dsffsjfjs[p=1=1==124",
	)
}

func help(mName string) string {
	return fmt.Sprintf("A %v of the request duration for HTTP requests, segmented by various dimensions. This metric provides detailed insights into the performance of your HTTP endpoints, allowing you to identify bottlenecks, track trends, and optimize your application for improved user experience.", mName)
}

var (
	generateConfig200samples = generateConfig{
		counters: 50, gauges: 40,
		classicHistograms: 10, classicHistogramBuckets: 8,
		nativeHistograms: 10,

		metricNameVariability: 10,
		exemplarRatio:         0.5,
	}
	generateConfig2000samples = generateConfig{
		counters: 500, gauges: 400,
		classicHistograms: 100, classicHistogramBuckets: 8,
		nativeHistograms: 100,

		metricNameVariability: 10,
		exemplarRatio:         0.5,
	}
	generateConfig10000samples = generateConfig{
		counters: 2500, gauges: 2000,
		classicHistograms: 500, classicHistogramBuckets: 8,
		nativeHistograms: 500,

		metricNameVariability: 10,
		exemplarRatio:         0.5,
	}
)

type generateConfig struct {
	counters, gauges        int
	classicHistograms       int
	classicHistogramBuckets int
	nativeHistograms        int

	metricNameVariability int
	exemplarRatio         float64
}

func (g generateConfig) Series() int {
	return g.counters + g.gauges + ((g.classicHistogramBuckets + 2) * g.classicHistograms) + g.nativeHistograms
}

// Remote write default is [ max_samples_per_send: <int> | default = 2000]
// But depending on backends we see batches vary from 200 to 10k (https://github.com/prometheus/prometheus/issues/5166#issuecomment-616618613).
func generatePrometheusMetricsBatch(cfg generateConfig) []timeSeries {
	ret := make([]timeSeries, cfg.Series())
	i := 0
	exemplarInterval := int(1 / cfg.exemplarRatio)

	for c := 0; c < cfg.counters; c++ {
		mName := fmt.Sprintf("metric_my_own_counter_bytes%v_total", i%cfg.metricNameVariability)
		ret[i] = timeSeries{
			seriesLabels: seriesLabels(mName, i),
			value:        0.123093 + float64(i),
			metadata:     &metadata.Metadata{Type: model.MetricTypeCounter, Help: help(mName), Unit: "bytes"},
			// Add jitter?
			timestamp:        1724844902198,
			createdTimestamp: 1724837702198,
		}
		if i%exemplarInterval == 0 {
			ret[i].exemplarLabels = labels.FromStrings("trace_id", uuid.NewString())
		}
		i++
	}

	for g := 0; g < cfg.gauges; g++ {
		mName := fmt.Sprintf("metric_my_own_gauge_operations%v", i%cfg.metricNameVariability)
		ret[i] = timeSeries{
			seriesLabels: seriesLabels(mName, i),
			value:        120412431 + float64(i),
			metadata:     &metadata.Metadata{Type: model.MetricTypeGauge, Help: help(mName), Unit: "operations"},
			// Add jitter?
			timestamp: 1724844902198,
		}
		if i%exemplarInterval == 0 {
			ret[i].exemplarLabels = labels.FromStrings("trace_id", uuid.NewString())
		}
		i++
	}

	for h := 0; h < cfg.classicHistograms; h++ {
		mName := fmt.Sprintf("metric_my_own_classic_histogram_seconds%v", h%cfg.metricNameVariability)
		meta := &metadata.Metadata{Type: model.MetricTypeHistogram, Help: help(mName), Unit: "seconds"}
		traceID := uuid.NewString()

		var lastCount float64
		for buckets := 0; buckets < cfg.classicHistogramBuckets; buckets++ {
			mName := mName + "_bucket"
			ret[i] = timeSeries{
				seriesLabels: append(seriesLabels(mName, h), labels.Label{Name: "le", Value: fmt.Sprintf("10023%v.0", h%10+buckets*100)}),
				value:        1123 + 10*float64(i),
				metadata:     meta,
				// Add jitter?
				timestamp:        1724844902198,
				createdTimestamp: 1724837702198,
			}
			lastCount = ret[i].value

			if i%exemplarInterval == 0 {
				ret[i].exemplarLabels = labels.FromStrings("trace_id", traceID)
			}
			i++
		}

		// Sum and count.
		{
			mName := mName + "_sum"
			ret[i] = timeSeries{
				seriesLabels: seriesLabels(mName, h),
				value:        121.2123 + float64(i), // In accurate, obviously.
				metadata:     meta,
				// Add jitter?
				timestamp:        1724844902198,
				createdTimestamp: 1724837702198,
			}
			if i%exemplarInterval == 0 {
				ret[i].exemplarLabels = labels.FromStrings("trace_id", traceID)
			}
			i++
		}
		{
			mName := mName + "_count"
			ret[i] = timeSeries{
				seriesLabels: seriesLabels(mName, h),
				value:        lastCount,
				metadata:     meta,
				// Add jitter?
				timestamp:        1724844902198,
				createdTimestamp: 1724837702198,
			}
			if i%exemplarInterval == 0 {
				ret[i].exemplarLabels = labels.FromStrings("trace_id", traceID)
			}
			i++
		}
	}

	for _, hist := range histogram.GenerateBigTestHistograms(cfg.nativeHistograms, 100) {
		mName := fmt.Sprintf("metric_my_own_native_histogram_seconds%v", i%cfg.metricNameVariability)
		ret[i] = timeSeries{
			seriesLabels: seriesLabels(mName, i),
			metadata:     &metadata.Metadata{Type: model.MetricTypeHistogram, Help: help(mName), Unit: "seconds"},
			// Add jitter?
			timestamp:        1724844902198,
			createdTimestamp: 1724837702198,
			histogram:        hist,
		}
		if i%exemplarInterval == 0 {
			ret[i].exemplarLabels = labels.FromStrings("trace_id", uuid.NewString())
		}
		i++
	}
	return ret
}

// Mimicking what we have in https://github.com/prometheus/prometheus/blob/main/storage/remote/queue_manager.go#L1377-L1387
type timeSeries struct {
	seriesLabels     labels.Labels
	value            float64
	histogram        *histogram.Histogram
	floatHistogram   *histogram.FloatHistogram
	metadata         *metadata.Metadata
	timestamp        int64
	createdTimestamp int64
	exemplarLabels   labels.Labels
}

func toV1(batch []timeSeries, withMetadata bool, withHistogramsAndExemplars bool) *prompb.WriteRequest {
	ret := &prompb.WriteRequest{
		Timeseries: make([]*prompb.TimeSeries, len(batch)),
	}
	if withMetadata {
		// This is not entirely correct, we had more complex protocol for this (stateful), but let's do this to fairly compare.
		ret.Metadata = make([]*prompb.MetricMetadata, len(batch))
	}
	for i, ts := range batch {
		ret.Timeseries[i] = &prompb.TimeSeries{
			Labels: prompb.FromLabels(ts.seriesLabels, nil),
		}
		if ts.histogram != nil {
			if withHistogramsAndExemplars {
				ret.Timeseries[i].Histograms = []*prompb.Histogram{prompb.FromIntHistogram(ts.timestamp, ts.histogram)}
			}
		} else {
			ret.Timeseries[i].Samples = []*prompb.Sample{{Value: ts.value, Timestamp: ts.timestamp}}
		}

		if withMetadata {
			ret.Metadata[i] = &prompb.MetricMetadata{
				MetricFamilyName: ts.seriesLabels.Get(labels.MetricName),
				Help:             ts.metadata.Help,
				Unit:             ts.metadata.Unit,
				Type:             prompb.FromMetadataType(ts.metadata.Type),
			}
		}

		if withHistogramsAndExemplars && len(ts.exemplarLabels) > 0 {
			ret.Timeseries[i].Exemplars = []*prompb.Exemplar{{
				Labels:    prompb.FromLabels(ts.exemplarLabels, nil),
				Value:     12414,
				Timestamp: ts.timestamp,
			}}
		}
	}
	return ret
}

func convertClassicToCustom(batch []timeSeries) []timeSeries {
	customHistograms := map[uint64]*timeSeries{}

	converted := make([]timeSeries, 0, len(batch))
	for _, ts := range batch {
		if ts.histogram != nil || ts.metadata.Type != model.MetricTypeHistogram {
			converted = append(converted, ts)
			continue
		}
		name := ts.seriesLabels.Get(labels.MetricName)
		familyLabels := ts.seriesLabels.Map()
		if strings.HasSuffix(name, "_bucket") {
			familyLabels[labels.MetricName] = strings.TrimSuffix(name, "_bucket")
			delete(familyLabels, labels.BucketLabel)
		}
		if strings.HasSuffix(name, "_count") {
			familyLabels[labels.MetricName] = strings.TrimSuffix(name, "_count")
		}
		if strings.HasSuffix(name, "_sum") {
			familyLabels[labels.MetricName] = strings.TrimSuffix(name, "_sum")
		}

		l := labels.FromMap(familyLabels)
		lHash := l.Hash()
		histTs, ok := customHistograms[lHash]
		if !ok {
			var exemplarLabels labels.Labels
			if len(customHistograms)%2 == 0 && len(ts.exemplarLabels) > 0 {
				exemplarLabels = ts.exemplarLabels
			}
			converted = append(converted, timeSeries{
				seriesLabels: l,
				histogram: &histogram.Histogram{
					Schema: histogram.CustomBucketsSchema,
				},
				metadata:         ts.metadata,
				timestamp:        ts.timestamp,
				createdTimestamp: ts.createdTimestamp,
				exemplarLabels:   exemplarLabels,
			})
			histTs = &converted[len(converted)-1]
			customHistograms[lHash] = histTs
		}

		// TODO(bwplotka): Yolo, test if this is accurate, but for benchmark it does not matter a lot.
		// TODO: Sort later if unsorted input.
		if strings.HasSuffix(name, "_bucket") {
			bound, err := strconv.ParseFloat(ts.seriesLabels.Get(labels.BucketLabel), 64)
			if err != nil {
				panic(err) // TODO: It's benchmark only after all...
			}
			histTs.histogram.CustomValues = append(histTs.histogram.CustomValues, bound)
			// Compact will compact this.
			histTs.histogram.PositiveSpans = append(histTs.histogram.PositiveSpans, histogram.Span{Offset: 0, Length: 1})

			// TODO: Fix precision (use float histogram if needed), but for benchmarks it's fine.
			prevTotalCount := int64(0)
			prevCount := int64(0)
			for _, b := range histTs.histogram.PositiveBuckets {
				prevCount += b
				prevTotalCount += prevCount
			}
			exactCount := int64(ts.value) - prevTotalCount
			histTs.histogram.PositiveBuckets = append(histTs.histogram.PositiveBuckets, exactCount-prevCount)
		}
		if strings.HasSuffix(name, "_count") {
			// TODO: Fix precision (use float histogram if needed), but for benchmarks it's fine.
			histTs.histogram.Count = uint64(ts.value)
		}
		if strings.HasSuffix(name, "_sum") {
			histTs.histogram.Sum = ts.value
		}
	}
	for _, histTs := range customHistograms {
		histTs.histogram.Compact(0)
	}
	return converted
}

func TestConvertClassicToCustom(t *testing.T) {
	cfg := generateConfig10000samples
	batch := generatePrometheusMetricsBatch(cfg)
	batch = convertClassicToCustom(batch)
	testutil.Equals(t, 10e3-cfg.classicHistograms*(cfg.classicHistogramBuckets+1), len(batch)) // buckets + 2 extra series, but one is for the resulting custom histograms).

	dups := map[uint64]struct{}{}
	for _, ts := range batch {
		if ts.histogram != nil && histogram.IsCustomBucketsSchema(ts.histogram.Schema) {
			testutil.Ok(t, ts.histogram.Validate())
			h := ts.seriesLabels.Hash()
			if _, ok := dups[h]; ok {
				t.Errorf("found a duplicate series %v", ts.seriesLabels.String())
			}
			dups[h] = struct{}{}
		}
	}
	testutil.Equals(t, cfg.classicHistograms, len(dups))
}

func toV2(batch []timeSeries, withHelp bool) *writev2.Request {
	s := writev2.NewSymbolTable()

	ret := &writev2.Request{
		Timeseries: make([]*writev2.TimeSeries, len(batch)),
	}

	for i, ts := range batch {
		labelsRefs := s.SymbolizeLabels(ts.seriesLabels, nil)
		var helpRef uint32
		if withHelp {
			helpRef = s.Symbolize(ts.metadata.Help)
		}
		ret.Timeseries[i] = &writev2.TimeSeries{
			LabelsRefs: labelsRefs,
			Metadata: &writev2.Metadata{
				HelpRef: helpRef,
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

func isResourceLabel(name string) bool {
	switch name {
	case "cluster", "instance", "job", "namespace", "pod", "team", "workload_controller", "workload_type":
		return true
	default:
		return false
	}
}

// toOTLP converts the Prometheus batch into an OTLP ExportMetricsServiceRequest
// with the exact same information: target/k8s labels are mapped to Resource
// attributes, metrics are grouped by family (Name, Description/Help, Unit), and
// metric-specific labels (reason, remote) are placed on DataPoints.
func toOTLP(batch []timeSeries) *colmetricsv1.ExportMetricsServiceRequest {
	type resourceEntry struct {
		rm            *metricsv1.ResourceMetrics
		metricsByName map[string]*metricsv1.Metric
	}

	var (
		resourcesByHash = map[uint64]*resourceEntry{}
		resourceMetrics []*metricsv1.ResourceMetrics
		resBuilder      labels.ScratchBuilder
	)

	for _, ts := range batch {
		var (
			metricName string
			pointAttrs []*commonv1.KeyValue
		)
		resBuilder.Reset()
		ts.seriesLabels.Range(func(l labels.Label) {
			if l.Name == labels.MetricName {
				metricName = l.Value
				return
			}
			if isResourceLabel(l.Name) {
				resBuilder.Add(l.Name, l.Value)
				return
			}
			pointAttrs = append(pointAttrs, &commonv1.KeyValue{
				Key:   l.Name,
				Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: l.Value}},
			})
		})
		resLabels := resBuilder.Labels()
		resHash := resLabels.Hash()

		resEntry, ok := resourcesByHash[resHash]
		if !ok {
			var resAttrs []*commonv1.KeyValue
			resLabels.Range(func(l labels.Label) {
				resAttrs = append(resAttrs, &commonv1.KeyValue{
					Key:   l.Name,
					Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: l.Value}},
				})
			})
			rm := &metricsv1.ResourceMetrics{
				Resource: &resourcev1.Resource{
					Attributes: resAttrs,
				},
				ScopeMetrics: []*metricsv1.ScopeMetrics{{}},
			}
			resEntry = &resourceEntry{
				rm:            rm,
				metricsByName: map[string]*metricsv1.Metric{},
			}
			resourcesByHash[resHash] = resEntry
			resourceMetrics = append(resourceMetrics, rm)
		}

		m, ok := resEntry.metricsByName[metricName]
		if !ok {
			m = &metricsv1.Metric{
				Name:        metricName,
				Description: ts.metadata.Help,
				Unit:        ts.metadata.Unit,
			}
			switch ts.metadata.Type {
			case model.MetricTypeCounter:
				m.Data = &metricsv1.Metric_Sum{
					Sum: &metricsv1.Sum{
						AggregationTemporality: metricsv1.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE,
						IsMonotonic:            true,
					},
				}
			case model.MetricTypeGauge:
				m.Data = &metricsv1.Metric_Gauge{
					Gauge: &metricsv1.Gauge{},
				}
			case model.MetricTypeHistogram:
				if ts.histogram != nil && histogram.IsCustomBucketsSchema(ts.histogram.Schema) {
					m.Data = &metricsv1.Metric_Histogram{
						Histogram: &metricsv1.Histogram{
							AggregationTemporality: metricsv1.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE,
						},
					}
				} else {
					m.Data = &metricsv1.Metric_ExponentialHistogram{
						ExponentialHistogram: &metricsv1.ExponentialHistogram{
							AggregationTemporality: metricsv1.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE,
						},
					}
				}
			default:
				panic(fmt.Sprintf("unsupported metric type: %v", ts.metadata.Type))
			}
			resEntry.metricsByName[metricName] = m
			resEntry.rm.ScopeMetrics[0].Metrics = append(resEntry.rm.ScopeMetrics[0].Metrics, m)
		}

		var exemplars []*metricsv1.Exemplar
		if len(ts.exemplarLabels) > 0 {
			ex := &metricsv1.Exemplar{
				TimeUnixNano: uint64(ts.timestamp) * 1e6,
				Value:        &metricsv1.Exemplar_AsDouble{AsDouble: 12414},
			}
			ts.exemplarLabels.Range(func(l labels.Label) {
				if l.Name == "trace_id" {
					ex.TraceId = []byte(l.Value)
					return
				}
				ex.FilteredAttributes = append(ex.FilteredAttributes, &commonv1.KeyValue{
					Key:   l.Name,
					Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: l.Value}},
				})
			})
			exemplars = []*metricsv1.Exemplar{ex}
		}

		timeNano := uint64(ts.timestamp) * 1e6
		startTimeNano := uint64(ts.createdTimestamp) * 1e6

		switch data := m.Data.(type) {
		case *metricsv1.Metric_Sum:
			data.Sum.DataPoints = append(data.Sum.DataPoints, &metricsv1.NumberDataPoint{
				Attributes:        pointAttrs,
				StartTimeUnixNano: startTimeNano,
				TimeUnixNano:      timeNano,
				Value:             &metricsv1.NumberDataPoint_AsDouble{AsDouble: ts.value},
				Exemplars:         exemplars,
			})
		case *metricsv1.Metric_Gauge:
			data.Gauge.DataPoints = append(data.Gauge.DataPoints, &metricsv1.NumberDataPoint{
				Attributes:   pointAttrs,
				TimeUnixNano: timeNano,
				Value:        &metricsv1.NumberDataPoint_AsDouble{AsDouble: ts.value},
				Exemplars:    exemplars,
			})
		case *metricsv1.Metric_Histogram:
			data.Histogram.DataPoints = append(data.Histogram.DataPoints, &metricsv1.HistogramDataPoint{
				Attributes:        pointAttrs,
				StartTimeUnixNano: startTimeNano,
				TimeUnixNano:      timeNano,
				Count:             ts.histogram.Count,
				Sum:               proto.Float64(ts.histogram.Sum),
				BucketCounts:      customBucketsToOTLPBucketCounts(ts.histogram),
				ExplicitBounds:    ts.histogram.CustomValues,
				Exemplars:         exemplars,
			})
		case *metricsv1.Metric_ExponentialHistogram:
			data.ExponentialHistogram.DataPoints = append(data.ExponentialHistogram.DataPoints, &metricsv1.ExponentialHistogramDataPoint{
				Attributes:        pointAttrs,
				StartTimeUnixNano: startTimeNano,
				TimeUnixNano:      timeNano,
				Count:             ts.histogram.Count,
				Sum:               proto.Float64(ts.histogram.Sum),
				Scale:             ts.histogram.Schema,
				ZeroCount:         ts.histogram.ZeroCount,
				ZeroThreshold:     ts.histogram.ZeroThreshold,
				Positive:          spansAndDeltasToOTLPBuckets(ts.histogram.PositiveSpans, ts.histogram.PositiveBuckets),
				Negative:          spansAndDeltasToOTLPBuckets(ts.histogram.NegativeSpans, ts.histogram.NegativeBuckets),
				Exemplars:         exemplars,
			})
		}
	}

	return &colmetricsv1.ExportMetricsServiceRequest{
		ResourceMetrics: resourceMetrics,
	}
}

func customBucketsToOTLPBucketCounts(h *histogram.Histogram) []uint64 {
	bucketCounts := make([]uint64, len(h.CustomValues)+1)
	var (
		bucketIdx int
		curCount  int64
		deltaIdx  int
	)
	for i, s := range h.PositiveSpans {
		if i == 0 {
			bucketIdx = int(s.Offset)
		} else {
			bucketIdx += int(s.Offset)
		}
		for range s.Length {
			curCount += h.PositiveBuckets[deltaIdx]
			deltaIdx++
			if bucketIdx < len(bucketCounts) {
				bucketCounts[bucketIdx] = uint64(curCount)
			}
			bucketIdx++
		}
	}
	return bucketCounts
}

func spansAndDeltasToOTLPBuckets(spans []histogram.Span, deltas []int64) *metricsv1.ExponentialHistogramDataPoint_Buckets {
	if len(spans) == 0 {
		return nil
	}
	var totalLen int
	for i, s := range spans {
		if i > 0 {
			totalLen += int(s.Offset)
		}
		totalLen += int(s.Length)
	}
	counts := make([]uint64, 0, totalLen)
	var (
		curCount int64
		deltaIdx int
	)
	for i, s := range spans {
		if i > 0 {
			for range s.Offset {
				counts = append(counts, 0)
			}
		}
		for range s.Length {
			curCount += deltas[deltaIdx]
			deltaIdx++
			counts = append(counts, uint64(curCount))
		}
	}
	return &metricsv1.ExponentialHistogramDataPoint_Buckets{
		Offset:       spans[0].Offset,
		BucketCounts: counts,
	}
}
