package main

import (
	"math"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
	dto "github.com/prometheus/client_model/go"
	p2m "github.com/qnib/prom2json/lib"

)

type testCase struct {
	name         string
	timestamp    int64
	mPrefix string
	mFamily *dto.MetricFamily
	output       *p2m.Family
}

var tcs = []testCase{
	testCase{
		name:      "test counter",
		timestamp: 123456789,
		mFamily: &dto.MetricFamily{
			Name: strPtr("counter1"),
			Type: metricTypePtr(dto.MetricType_COUNTER),
			Metric: []*dto.Metric{
				// Test metric with tags
				&dto.Metric{
					Label: []*dto.LabelPair{
						createLabelPair("tag1", "abc"),
						createLabelPair("tag2", "def"),
					},
					Counter: &dto.Counter{
						Value: floatPtr(1),
					},
				},
				// Test metric without tags
				&dto.Metric{
					Label: []*dto.LabelPair{},
					Counter: &dto.Counter{
						Value: floatPtr(2),
					},
				},
				// Test metric with -Inf
				&dto.Metric{
					Label: []*dto.LabelPair{
						createLabelPair("inf", "neg"),
					},
					Counter: &dto.Counter{
						Value: floatPtr(math.Inf(-1)),
					},
				},
				// Test metric with +Inf
				&dto.Metric{
					Label: []*dto.LabelPair{
						createLabelPair("inf", "pos"),
					},
					Counter: &dto.Counter{
						Value: floatPtr(math.Inf(1)),
					},
				},
			},
		},
		output: &p2m.Family{
			Name: "counter1",
			Help: "",
			Type: "COUNTER",
			Metrics: []interface{}{
				p2m.Metric{
					Labels: map[string]string{
						"tag2": "def",
						"tag1": "abc",
					},
					Value: "1",
				},
				p2m.Metric{
					Labels: map[string]string{},
					Value:  "2",
				},
				p2m.Metric{
					Labels: map[string]string{
						"inf": "neg",
					},
					Value: "-Inf",
				},
				p2m.Metric{
					Labels: map[string]string{
						"inf": "pos",
					},
					Value: "+Inf",
				},
			},
		},
	},
	testCase{
		name:      "test summaries",
		timestamp: 123456789,
		mFamily: &dto.MetricFamily{
			Name: strPtr("summary1"),
			Type: metricTypePtr(dto.MetricType_SUMMARY),
			Metric: []*dto.Metric{
				&dto.Metric{
					// Test summary with NaN
					Label: []*dto.LabelPair{
						createLabelPair("tag1", "abc"),
						createLabelPair("tag2", "def"),
					},
					Summary: &dto.Summary{
						SampleCount: uintPtr(1),
						SampleSum:   floatPtr(2),
						Quantile: []*dto.Quantile{
							createQuantile(0.5, 3),
							createQuantile(0.9, 4),
							createQuantile(0.99, math.NaN()),
						},
					},
				},
			},
		},
		output: &p2m.Family{
			Name: "summary1",
			Help: "",
			Type: "SUMMARY",
			Metrics: []interface{}{
				p2m.Summary{
					Labels: map[string]string{
						"tag1": "abc",
						"tag2": "def",
					},
					Quantiles: map[string]string{
						"0.5":  "3",
						"0.9":  "4",
						"0.99": "NaN",
					},
					Count: "1",
					Sum:   "2",
				},
			},
		},
	},
	testCase{
		name:      "test histograms",
		timestamp: 123456789,
		mFamily: &dto.MetricFamily{
			Name: strPtr("histogram1"),
			Type: metricTypePtr(dto.MetricType_HISTOGRAM),
			Metric: []*dto.Metric{
				&dto.Metric{
					// Test summary with NaN
					Label: []*dto.LabelPair{
						createLabelPair("tag1", "abc"),
						createLabelPair("tag2", "def"),
					},
					Histogram: &dto.Histogram{
						SampleCount: uintPtr(1),
						SampleSum:   floatPtr(2),
						Bucket: []*dto.Bucket{
							createBucket(250000, 3),
							createBucket(500000, 4),
							createBucket(1e+06, 5),
						},
					},
				},
			},
		},
		output: &p2m.Family{
			Name: "histogram1",
			Help: "",
			Type: "HISTOGRAM",
			Metrics: []interface{}{
				p2m.Histogram{
					Labels: map[string]string{
						"tag1": "abc",
						"tag2": "def",
					},
					Buckets: map[string]string{
						"250000": "3",
						"500000": "4",
						"1e+06":  "5",
					},
					Count: "1",
					Sum:   "0",
				},
			},
		},
	},
}


func TestConvertToMetricFamily(t *testing.T) {
	for _, tc := range tcs {
		output := p2m.NewFamily(tc.mFamily)
		if !reflect.DeepEqual(tc.output, output) {
			t.Errorf("test case %s: conversion to metricFamily format failed:\nexpected:\n%s\n\nactual:\n%s",
				tc.name, spew.Sdump(tc.output), spew.Sdump(output))
		}
	}
}

func strPtr(s string) *string {
	return &s
}

func floatPtr(f float64) *float64 {
	return &f
}

func metricTypePtr(mt dto.MetricType) *dto.MetricType {
	return &mt
}

func uintPtr(u uint64) *uint64 {
	return &u
}

func createLabelPair(name string, value string) *dto.LabelPair {
	return &dto.LabelPair{
		Name:  &name,
		Value: &value,
	}
}

func createQuantile(q float64, v float64) *dto.Quantile {
	return &dto.Quantile{
		Quantile: &q,
		Value:    &v,
	}
}

func createBucket(bound float64, count uint64) *dto.Bucket {
	return &dto.Bucket{
		UpperBound:      &bound,
		CumulativeCount: &count,
	}
}
