package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"runtime"

	"github.com/matttproud/golang_protobuf_extensions/ext"
	"github.com/prometheus/client_golang/text"

	dto "github.com/prometheus/client_model/go"
)

const acceptHeader = `application/vnd.google.protobuf;proto=io.prometheus.client.MetricFamily;encoding=delimited;q=0.7,text/plain;version=0.0.4;q=0.3`

type metricFamily struct {
	Name    string        `json:"name"`
	Help    string        `json:"help"`
	Type    string        `json:"type"`
	Metrics []interface{} `json:"metrics,omitempty"` // Either metric or summary.
}

// metric is for all "single value" metrics.
type metric struct {
	Labels map[string]string `json:"labels,omitempty"`
	Value  float64           `json:"value"`
}

type summary struct {
	Labels    map[string]string  `json:"labels,omitempty"`
	Quantiles map[string]float64 `json:"quantiles,omitempty"`
	Count     uint64             `json:"count"`
	Sum       float64            `json:"sum"`
}

func newMetricFamily(dtoMF *dto.MetricFamily) *metricFamily {
	mf := &metricFamily{
		Name:    dtoMF.GetName(),
		Help:    dtoMF.GetHelp(),
		Type:    dtoMF.GetType().String(),
		Metrics: make([]interface{}, len(dtoMF.Metric)),
	}
	isSummary := dtoMF.GetType() == dto.MetricType_SUMMARY
	for i, m := range dtoMF.Metric {
		if isSummary {
			mf.Metrics[i] = summary{
				Labels:    makeLabels(m),
				Quantiles: makeQuantiles(m),
				Count:     m.GetSummary().GetSampleCount(),
				Sum:       m.GetSummary().GetSampleSum(),
			}
		} else {
			mf.Metrics[i] = metric{
				Labels: makeLabels(m),
				Value:  getValue(m),
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

func makeQuantiles(m *dto.Metric) map[string]float64 {
	result := map[string]float64{}
	for _, q := range m.GetSummary().Quantile {
		result[fmt.Sprint(q.GetQuantile())] = q.GetValue()
	}
	return result
}

func fetchMetricFamilies(url string, ch chan<- *dto.MetricFamily) {
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
			if _, err = ext.ReadDelimited(resp.Body, mf); err != nil {
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
		var parser text.Parser
		metricFamilies, err := parser.TextToMetricFamilies(resp.Body)
		if err != nil {
			log.Fatalln("reading text format failed:", err)
		}
		for _, mf := range metricFamilies {
			ch <- mf
		}
	}
}

func main() {
	runtime.GOMAXPROCS(2)
	if len(os.Args) != 2 {
		log.Fatalf("Usage: %s METRICS_URL", os.Args[0])
	}

	mfChan := make(chan *dto.MetricFamily, 1024)

	go fetchMetricFamilies(os.Args[1], mfChan)

	result := []*metricFamily{}
	for mf := range mfChan {
		result = append(result, newMetricFamily(mf))
	}
	json, err := json.Marshal(result)
	if err != nil {
		log.Fatalln("error marshaling JSON:", err)
	}
	if _, err := os.Stdout.Write(json); err != nil {
		log.Fatalln("error writing to stdout:", err)
	}
	fmt.Println()
}
