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
	"flag"
	"fmt"
	"os"

	"github.com/prometheus/common/log"

	dto "github.com/prometheus/client_model/go"

	"github.com/prometheus/prom2json"
)

func main() {
	cert := flag.String("cert", "", "certificate file")
	key := flag.String("key", "", "key file")
	skipServerCertCheck := flag.Bool("accept-invalid-cert", false, "Accept any certificate during TLS handshake. Insecure, use only for testing.")
	flag.Parse()

	if len(flag.Args()) != 1 {
		log.Fatalf("Usage: %s METRICS_URL", os.Args[0])
	}
	if (*cert != "" && *key == "") || (*cert == "" && *key != "") {
		log.Fatalf("Usage: %s METRICS_URL\n with TLS client authentication: %s -cert=/path/to/certificate -key=/path/to/key METRICS_URL", os.Args[0], os.Args[0])
	}

	mfChan := make(chan *dto.MetricFamily, 1024)

	go func() {
		err := prom2json.FetchMetricFamilies(flag.Args()[0], mfChan, *cert, *key, *skipServerCertCheck)
		if err != nil {
			log.Fatal(err)
		}
	}()

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
