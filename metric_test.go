package prom2json

import (
	"math"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/assert"

	dto "github.com/prometheus/client_model/go"
	"regexp"
)

type testCase struct {
	name      string
	timestamp int64
	mPrefix   string
	mFamily   *dto.MetricFamily
	output    *Family
}

// Test metric with tags
var mWt1 = &dto.Metric{
	Label: []*dto.LabelPair{
		createLabelPair("tag1", "abc"),
		createLabelPair("tag2", "def"),
	},
	Counter: &dto.Counter{
		Value: floatPtr(1),
	},
}

// Test metric without tags
var mWOt1 = &dto.Metric{
	Label: []*dto.LabelPair{},
	Counter: &dto.Counter{
		Value: floatPtr(2),
	},
}

var mf1 = &dto.MetricFamily{
	Name: strPtr("counter1"),
	Type: metricTypePtr(dto.MetricType_COUNTER),
	Metric: []*dto.Metric{
		mWt1,
		mWOt1,
	},
}

var testCounter = testCase{
	name:      "test counter",
	timestamp: 123456789,
	mFamily: &dto.MetricFamily{
		Name: strPtr("counter1"),
		Type: metricTypePtr(dto.MetricType_COUNTER),
		Metric: []*dto.Metric{
			mWt1,
			mWOt1,
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
	output: &Family{
		Name: "counter1",
		Help: "",
		Type: "COUNTER",
		Metrics: []Metrics{
			Metric{
				Labels: map[string]string{
					"tag2": "def",
					"tag1": "abc",
				},
				Value: "1",
			},
			Metric{
				Labels: map[string]string{},
				Value:  "2",
			},
			Metric{
				Labels: map[string]string{
					"inf": "neg",
				},
				Value: "-Inf",
			},
			Metric{
				Labels: map[string]string{
					"inf": "pos",
				},
				Value: "+Inf",
			},
		},
	},
}

var testSum = testCase{
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
	output: &Family{
		Name: "summary1",
		Help: "",
		Type: "SUMMARY",
		Metrics: []Metrics{
			Summary{
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
}

var testHistogram = testCase{
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
	output: &Family{
		Name: "histogram1",
		Help: "",
		Type: "HISTOGRAM",
		Metrics: []Metrics{
			Histogram{
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
}

var tcs = []testCase{
	testCounter,
	testSum,
	testHistogram,
}

func TestConvertToMetricFamily(t *testing.T) {
	for _, tc := range tcs {
		output := NewFamily(tc.mFamily)
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

func TestFamily_AddLabel(t *testing.T) {
	x1 := NewFamily(mf1)
	x1.AddLabel("tagX", "valX")
	for i, m := range x1.Metrics {
		exp := map[string]string{"tagX": "valX"}
		lbs := mf1.Metric[i].GetLabel()
		for _, lb := range lbs {
			exp[lb.GetName()] = lb.GetValue()
		}
		got := m.GetLabels()
		assert.Equal(t, exp, got)
	}
}

func TestFamily_ToOpenTSDBv1(t *testing.T) {
	x1 := NewFamily(mf1)
	got := x1.ToOpenTSDBv1()
	p := []string{`put counter1 [0-9]+ 1.000000 tag1=abc tag2=def`, `put counter1 [0-9]+ 2.000000`}
	_, err := regexp.MatchString(p[0], got[0])
	assert.NoError(t, err, "Should match OpenTSDBv1 string")
	_, err = regexp.MatchString(p[1], got[1])
	assert.NoError(t, err, "Should match OpenTSDBv1 string")
}

func TestSanitizeTags(t *testing.T) {
	got, err := SanitizeTags("tag1", "abcdf")
	assert.NoError(t, err, "fine")
	assert.Equal(t, "tag1=abcdf", got)
	_, err = SanitizeTags("tag2", "abcdf.asda,asd")
	assert.Error(t, err, "fine")
	_, err = SanitizeTags("abcdf.asda,asd", "val3")
	assert.Error(t, err, "fine")
}

func TestLabelToString(t *testing.T) {
	_, err := LabelToString(map[string]string{})
	assert.Error(t, err, "Empty map")
	got, err := LabelToString(map[string]string{"tag1": "val1"})
	assert.NoError(t, err, "fine")
	assert.Equal(t, []string{"tag1=val1"}, got)
	got, err = LabelToString(map[string]string{"tag1": "val1", "tag2": "abcdf.asda,asd"})
	assert.NoError(t, err, "fine")
	assert.Equal(t, []string{"tag1=val1"}, got)
	_, err = LabelToString(map[string]string{"tag2": "abcdf.asda,asd"})
	assert.Error(t, err, "All k/v pairs fail")
}
