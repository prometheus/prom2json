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
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/alecthomas/kingpin/v2"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/version"

	"github.com/prometheus/prom2json"
)

var usage = `The path or URL to metrics to convert, if omitted, defaults to read from STDIN.

Examples:

	$ prom2json http://my-prometheus-server:9000/metrics

	$ curl http://my-prometheus-server:9000/metrics | prom2json
	
`

func main() {
	cert := kingpin.Flag("cert", "client certificate file").PlaceHolder("FILE").String()
	key := kingpin.Flag("key", "client certificate's key file").PlaceHolder("FILE").String()
	skipServerCertCheck := kingpin.Flag("accept-invalid-cert", "Accept any certificate during TLS handshake. Insecure, use only for testing.").Bool()
	escapingScheme := kingpin.Flag("escaping", "Sets an escaping scheme in content negotiation. Use 'allow-utf-8' for full UTF-8 character support.").
		PlaceHolder("SCHEME").
		Enum(
			"allow-utf-8",
			"underscores",
			"dots",
			"values",
		)

	kingpin.CommandLine.UsageWriter(os.Stderr)
	kingpin.Version(version.Print("prom2json"))
	kingpin.HelpFlag.Short('h')

	var input io.Reader
	var err error
	arg := kingpin.Arg("METRICS_PATH | METRICS_URL", usage).String()

	kingpin.Parse()

	if *arg == "" {
		// Use stdin on empty argument.
		input = os.Stdin
	} else if url, urlErr := url.Parse(*arg); urlErr != nil || url.Scheme == "" {
		// `url, err := url.Parse("/some/path.txt")` results in: `err == nil && url.Scheme == ""`
		// Open file since arg appears not to be a valid URL (parsing error occurred or the scheme is missing).
		if input, err = os.Open(*arg); err != nil {
			fmt.Fprintln(os.Stderr, "error opening file:", err)
			os.Exit(1)
		}
	} else {
		// Validate Client SSL arguments since arg appears to be a valid URL.
		if (*cert != "" && *key == "") || (*cert == "" && *key != "") {
			fmt.Fprintf(os.Stderr, "%s\n with TLS client authentication: %s --cert /path/to/certificate --key /path/to/key METRICS_URL", usage, os.Args[0])
			os.Exit(1)
		}
	}

	var produce func(chan<- *dto.MetricFamily) error
	// Missing input means we are reading from an URL.
	if input != nil {
		produce = func(ch chan<- *dto.MetricFamily) error {
			if err := prom2json.ParseReader(input, ch); err != nil {
				return fmt.Errorf("error reading metrics: %w", err)
			}
			return nil
		}
	} else {
		transport, err := makeTransport(*cert, *key, *skipServerCertCheck)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		produce = func(ch chan<- *dto.MetricFamily) error {
			return prom2json.FetchMetricFamiliesWithEscapingScheme(*arg, ch, transport, *escapingScheme)
		}
	}

	result, err := readFamilies(produce)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	jsonText, err := json.Marshal(result)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error marshaling JSON:", err)
		os.Exit(1)
	}
	if _, err := os.Stdout.Write(jsonText); err != nil {
		fmt.Fprintln(os.Stderr, "error writing to stdout:", err)
		os.Exit(1)
	}
	fmt.Println()
}

// readFamilies runs produce in its own goroutine and collects the metric
// families it sends. produce closes the channel before it returns, even on
// error, so the error has to be waited for explicitly. Otherwise the caller
// could see a closed channel and print an empty result before the error
// arrives.
func readFamilies(produce func(chan<- *dto.MetricFamily) error) ([]*prom2json.Family, error) {
	mfChan := make(chan *dto.MetricFamily, 1024)
	errChan := make(chan error, 1)
	go func() {
		errChan <- produce(mfChan)
	}()

	result := []*prom2json.Family{}
	for mf := range mfChan {
		result = append(result, prom2json.NewFamily(mf))
	}
	if err := <-errChan; err != nil {
		return nil, err
	}
	return result, nil
}

func makeTransport(
	certificate string, key string,
	skipServerCertCheck bool,
) (*http.Transport, error) {
	// Start with the DefaultTransport for sane defaults.
	transport := http.DefaultTransport.(*http.Transport).Clone()
	// Conservatively disable HTTP keep-alives as this program will only
	// ever need a single HTTP request.
	transport.DisableKeepAlives = true
	// Timeout early if the server doesn't even return the headers.
	transport.ResponseHeaderTimeout = time.Minute
	tlsConfig := &tls.Config{InsecureSkipVerify: skipServerCertCheck}
	if certificate != "" && key != "" {
		cert, err := tls.LoadX509KeyPair(certificate, key)
		if err != nil {
			return nil, err
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}
	transport.TLSClientConfig = tlsConfig
	return transport, nil
}
