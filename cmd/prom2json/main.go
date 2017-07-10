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
	"runtime"

	"github.com/prometheus/log"

	dto "github.com/prometheus/client_model/go"
	"github.com/ChristianKniep/prom2json/lib"
)

func main() {
	runtime.GOMAXPROCS(2)
	if len(os.Args) != 2 {
		log.Fatalf("Usage: %s METRICS_URL", os.Args[0])
	}

	mfChan := make(chan *dto.MetricFamily, 1024)

	go metric.FetchMetricFamilies(os.Args[1], mfChan)

	result := []*metric.Family{}
	for mf := range mfChan {
		result = append(result, metric.NewFamily(mf))
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
