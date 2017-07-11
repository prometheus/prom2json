package prom2json

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/matttproud/golang_protobuf_extensions/pbutil"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/log"

	"errors"
	dto "github.com/prometheus/client_model/go"
	"regexp"
)

const acceptHeader = `application/vnd.google.protobuf;proto=io.prometheus.client.MetricFamily;encoding=delimited;q=0.7,text/plain;version=0.0.4;q=0.3`

// Family mirrors the MetricFamily proto message.
type Family struct {
	//Time    time.Time
	Name    string    `json:"name"`
	Help    string    `json:"help"`
	Type    string    `json:"type"`
	Metrics []Metrics `json:"metrics,omitempty"` // Either metric or summary.
}

// Metrics ensures that all metric types implement common functions like GetLabels().
type Metrics interface {
	GetLabels() map[string]string
}

// Metric is for all "single value" metrics, i.e. Counter, Gauge, and Untyped.
type Metric struct {
	Labels map[string]string `json:"labels,omitempty"`
	Value  string            `json:"value"`
}

// GetLabels returns the Labels of the Metric type to implement the Metrics interface.
func (m Metric) GetLabels() map[string]string {
	return m.Labels
}

// Summary mirrors the Summary proto message.
type Summary struct {
	Labels    map[string]string `json:"labels,omitempty"`
	Quantiles map[string]string `json:"quantiles,omitempty"`
	Count     string            `json:"count"`
	Sum       string            `json:"sum"`
}

// GetLabels returns the Labels of the Summary type to implement the Metrics interface.
func (s Summary) GetLabels() map[string]string {
	return s.Labels
}

// Histogram mirrors the Histogram proto message.
type Histogram struct {
	Labels  map[string]string `json:"labels,omitempty"`
	Buckets map[string]string `json:"buckets,omitempty"`
	Count   string            `json:"count"`
	Sum     string            `json:"sum"`
}

// GetLabels returns the Labels of the Histogram type to implement the Metrics interface.
func (h Histogram) GetLabels() map[string]string {
	return h.Labels
}

// NewFamily consumes a MetricFamily and transforms it to the local Family type.
func NewFamily(dtoMF *dto.MetricFamily) *Family {
	mf := &Family{
		//Time:    time.Now(),
		Name:    dtoMF.GetName(),
		Help:    dtoMF.GetHelp(),
		Type:    dtoMF.GetType().String(),
		Metrics: make([]Metrics, len(dtoMF.Metric)),
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

// FetchMetricFamilies retrieves metrics from the provided URL, decodes them
// into MetricFamily proto messages, and sends them to the provided channel. It
// returns after all MetricFamilies have been sent.
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
	ParseResponse(resp, ch)
}

// ParseResponse consumes an http.Response and pushes it to the MetricFamily
// channel. It returns when all all MetricFamilies are parsed and put on the
// channel.
func ParseResponse(resp *http.Response, ch chan<- *dto.MetricFamily) {
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

// AddLabel allows to add key/value labels to an already existing Family.
func (f *Family) AddLabel(key, val string) {
	for i, item := range f.Metrics {
		switch item.(type) {
		case Metric:
			m := item.(Metric)
			m.Labels[key] = val
			f.Metrics[i] = m
		}
	}
}

// ToOpenTSDBv1 transforms the metrics(not yet Histograms/Summaries) to OpenTSDB line format (v1),
// returns an array of lines.
func (f *Family) ToOpenTSDBv1() []string {
	base := fmt.Sprintf("put %s %d", f.Name, time.Now().Unix())
	res := []string{}
	for _, item := range f.Metrics {
		switch item.(type) {
		case Metric:
			m := item.(Metric)
			val, err := strconv.ParseFloat(m.Value, 64)
			if err != nil {
				continue
			}
			lab, err := LabelToString(m.Labels)
			if err != nil {
				met := fmt.Sprintf("%s %f", base, val)
				res = append(res, met)
			} else {
				met := fmt.Sprintf("%s %f %s", base, val, strings.Join(lab, " "))
				res = append(res, met)
			}
		default:
			log.Printf("Type '%s' not yet implemented", reflect.TypeOf(item))
		}
	}
	return res
}

// LabelToString consumes a k/v map and returns a sanitized []string{} with key=val pairs.
// In case the map is empty or all k/v pairs fail the sanitization test, an error is return.
func LabelToString(inp map[string]string) (lab []string, err error) {
	if len(inp) == 0 {
		return nil, errors.New("amp is empty, therefore no string for you")
	}
	for k, v := range inp {
		tag, err := SanitizeTags(k, v)
		if err != nil {
			log.Printf(err.Error())
			continue
		}
		lab = append(lab, tag)
	}
	if len(lab) == 0 {
		return nil, errors.New("all k/v pairs failed the sanitization test")
	}
	return
}

// SanitizeTags checks whether the tag is compliant with the rules defined in
// http://opentsdb.net/docs/build/html/user_guide/writing.html#metrics-and-tags.
// For starters only `^[a-zA-Z0-9\-\./]+$` is allowed.
// TODO: Add Unicode letters
func SanitizeTags(k, v string) (tag string, err error) {
	// a to z, A to Z, 0 to 9, -, _, ., /
	r := regexp.MustCompile(`^[a-zA-Z0-9\-\./]+$`)
	if !r.MatchString(k) || !r.MatchString(v) {
		return "", fmt.Errorf(`Did not match '[a-zA-Z\-\./]+': %s=%s`, k, v)
	}
	tag = fmt.Sprintf("%s=%s", k, v)
	return
}
