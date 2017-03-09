package metric

import (
    "fmt"
    "io"
    "mime"
    "net/http"

    "github.com/matttproud/golang_protobuf_extensions/pbutil"
    "github.com/prometheus/common/expfmt"
    "github.com/prometheus/log"

    dto "github.com/prometheus/client_model/go"
)
const acceptHeader = `application/vnd.google.protobuf;proto=io.prometheus.client.MetricFamily;encoding=delimited;q=0.7,text/plain;version=0.0.4;q=0.3`

// MetricFamily holds the family... :)
type Family struct {
	Name    string        `json:"name"`
	Help    string        `json:"help"`
	Type    string        `json:"type"`
	Metrics []interface{} `json:"metrics,omitempty"` // Either metric or summary.
}

// Metric is for all "single value" metrics.
type Metric struct {
	Labels map[string]string `json:"labels,omitempty"`
	Value  string            `json:"value"`
}

type Summary struct {
	Labels    map[string]string `json:"labels,omitempty"`
	Quantiles map[string]string `json:"quantiles,omitempty"`
	Count     string            `json:"count"`
	Sum       string            `json:"sum"`
}

type Histogram struct {
	Labels  map[string]string `json:"labels,omitempty"`
	Buckets map[string]string `json:"buckets,omitempty"`
	Count   string            `json:"count"`
	Sum     string            `json:"sum"`
}

func NewFamily(dtoMF *dto.MetricFamily) *Family {
	mf := &Family{
		Name:    dtoMF.GetName(),
		Help:    dtoMF.GetHelp(),
		Type:    dtoMF.GetType().String(),
		Metrics: make([]interface{}, len(dtoMF.Metric)),
	}
	for i, m := range dtoMF.Metric {
		if dtoMF.GetType() == dto.MetricType_SUMMARY {
			mf.Metrics[i] = Summary{
				Labels:    makeLabels(m),
				Quantiles: makeQuantiles(m),
				Count:     fmt.Sprint(m.GetSummary().GetSampleCount()),
				Sum:       fmt.Sprint(m.GetSummary().GetSampleSum()),
			}
		} else if dtoMF.GetType() == dto.MetricType_HISTOGRAM {
			mf.Metrics[i] = Histogram{
				Labels:  makeLabels(m),
				Buckets: makeBuckets(m),
				Count:   fmt.Sprint(m.GetHistogram().GetSampleCount()),
				Sum:     fmt.Sprint(m.GetSummary().GetSampleSum()),
			}
		} else {
			mf.Metrics[i] = Metric{
				Labels: makeLabels(m),
				Value:  fmt.Sprint(getValue(m)),
			}
		}
	}
	return mf
}

func getValue(m *dto.Metric) float64 {
	if m.Gauge != nil {
		return m.GetGauge().GetValue()
	}
	if m.Counter != nil {
		return m.GetCounter().GetValue()
	}
	if m.Untyped != nil {
		return m.GetUntyped().GetValue()
	}
	return 0.
}

func makeLabels(m *dto.Metric) map[string]string {
	result := map[string]string{}
	for _, lp := range m.Label {
		result[lp.GetName()] = lp.GetValue()
	}
	return result
}

func makeQuantiles(m *dto.Metric) map[string]string {
	result := map[string]string{}
	for _, q := range m.GetSummary().Quantile {
		result[fmt.Sprint(q.GetQuantile())] = fmt.Sprint(q.GetValue())
	}
	return result
}

func makeBuckets(m *dto.Metric) map[string]string {
	result := map[string]string{}
	for _, b := range m.GetHistogram().Bucket {
		result[fmt.Sprint(b.GetUpperBound())] = fmt.Sprint(b.GetCumulativeCount())
	}
	return result
}

// FetchMetricFamilies does stuff
func FetchMetricFamilies(url string, ch chan<- *dto.MetricFamily) {
	defer close(ch)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatalf("creating GET request for URL %q failed: %s", url, err)
	}
	req.Header.Add("Accept", acceptHeader)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("executing GET request for URL %q failed: %s", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("GET request for URL %q returned HTTP status %s", url, resp.Status)
	}

	mediatype, params, err := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if err == nil && mediatype == "application/vnd.google.protobuf" &&
		params["encoding"] == "delimited" &&
		params["proto"] == "io.prometheus.client.MetricFamily" {
		for {
			mf := &dto.MetricFamily{}
			if _, err = pbutil.ReadDelimited(resp.Body, mf); err != nil {
				if err == io.EOF {
					break
				}
				log.Fatalln("reading metric family protocol buffer failed:", err)
			}
			ch <- mf
		}
	} else {
		// We could do further content-type checks here, but the
		// fallback for now will anyway be the text format
		// version 0.0.4, so just go for it and see if it works.
		var parser expfmt.TextParser
		metricFamilies, err := parser.TextToMetricFamilies(resp.Body)
		if err != nil {
			log.Fatalln("reading text format failed:", err)
		}
		for _, mf := range metricFamilies {
			ch <- mf
		}
	}
}
