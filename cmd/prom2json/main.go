// Copyright 2014 Prometheus Team
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
	"encoding/json"
	"fmt"
	"os"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/log"

	"github.com/prometheus/prom2json"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("Usage: %s METRICS_URL", os.Args[0])
	}

	mfChan := make(chan *dto.MetricFamily, 1024)

	go prom2json.FetchMetricFamilies(os.Args[1], mfChan)

	result := []*prom2json.Family{}
	for mf := range mfChan {
		result = append(result, prom2json.NewFamily(mf))
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
