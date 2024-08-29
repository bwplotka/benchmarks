// Copyright (c) Bartłomiej Płotka @bwplotka
// Licensed under the Apache License 2.0.

package metricsstreaming

import (
	"fmt"

	prompb "github.com/bwplotka/benchmarks/benchmarks/metrics-streaming/io/prometheus/write/v1"
	writev2 "github.com/bwplotka/benchmarks/benchmarks/metrics-streaming/io/prometheus/write/v2"
	"github.com/google/uuid"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/histogram"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/model/metadata"
	"github.com/prometheus/prometheus/tsdb/tsdbutil"
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
		mName := fmt.Sprintf("metric_my_own_classic_histogram_seconds%v", i%cfg.metricNameVariability)
		meta := &metadata.Metadata{Type: model.MetricTypeHistogram, Help: help(mName), Unit: "seconds"}
		traceID := uuid.NewString()
		// Sum and count.
		{
			mName = mName + "_sum"
			ret[i] = timeSeries{
				seriesLabels: seriesLabels(mName, i),
				value:        121.2123 + float64(i),
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
			mName = mName + "_count"
			ret[i] = timeSeries{
				seriesLabels: seriesLabels(mName, i),
				value:        2456 + float64(i),
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
		for buckets := 0; buckets < cfg.classicHistogramBuckets; buckets++ {
			ret[i] = timeSeries{
				seriesLabels: append(seriesLabels(mName, i), labels.Label{Name: "le", Value: fmt.Sprintf("10023%v", i%10)}),
				value:        1123 + float64(i),
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

	for h := 0; h < cfg.nativeHistograms; h++ {
		mName := fmt.Sprintf("metric_my_own_native_histogram_seconds%v", i%cfg.metricNameVariability)
		ret[i] = timeSeries{
			seriesLabels: seriesLabels(mName, i),
			metadata:     &metadata.Metadata{Type: model.MetricTypeGauge, Help: help(mName), Unit: "seconds"},
			// Add jitter?
			timestamp:        1724844902198,
			createdTimestamp: 1724837702198,
			histogram:        tsdbutil.GenerateTestHistogram(i),
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

func toV1(batch []timeSeries, withMetadata bool) *prompb.WriteRequest {
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
			ret.Timeseries[i].Histograms = []*prompb.Histogram{prompb.FromIntHistogram(ts.timestamp, ts.histogram)}
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

		if len(ts.exemplarLabels) > 0 {
			ret.Timeseries[i].Exemplars = []*prompb.Exemplar{{
				Labels:    prompb.FromLabels(ts.exemplarLabels, nil),
				Value:     12414,
				Timestamp: ts.timestamp,
			}}
		}
	}
	return ret
}

func toV2(batch []timeSeries) *writev2.Request {
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
			CreatedTimestamp: ts.createdTimestamp,
		}
		if ts.histogram != nil {
			ret.Timeseries[i].Histograms = []*writev2.Histogram{writev2.FromIntHistogram(ts.timestamp, ts.histogram)}
		} else {
			ret.Timeseries[i].Samples = []*writev2.Sample{{Value: ts.value, Timestamp: ts.timestamp}}
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
