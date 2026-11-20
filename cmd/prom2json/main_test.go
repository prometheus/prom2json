// Copyright The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	dto "github.com/prometheus/client_model/go"

	"github.com/prometheus/prom2json"
)

func TestReadFamiliesReturnsProducerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	tcs := []struct {
		name    string
		produce func(chan<- *dto.MetricFamily) error
	}{
		{
			name: "http error",
			produce: func(ch chan<- *dto.MetricFamily) error {
				return prom2json.FetchMetricFamilies(srv.URL, ch, http.DefaultTransport)
			},
		},
		{
			name: "malformed text",
			produce: func(ch chan<- *dto.MetricFamily) error {
				return prom2json.ParseReader(strings.NewReader("garbage{\n"), ch)
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			result, err := readFamilies(tc.produce)
			if err == nil {
				t.Fatalf("expected an error, got nil and %d families", len(result))
			}
		})
	}
}

func TestReadFamilies(t *testing.T) {
	in := "# TYPE a counter\na 1\n# TYPE b gauge\nb 2\n"
	result, err := readFamilies(func(ch chan<- *dto.MetricFamily) error {
		return prom2json.ParseReader(strings.NewReader(in), ch)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 families, got %d", len(result))
	}
}
