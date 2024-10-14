// Copyright 2022 The Prometheus Authors
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
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/nelkinda/health-go"
	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/prometheus-community/avalanche/metrics"
	"github.com/prometheus-community/avalanche/pkg/download"
)

func main() {
	kingpin.Version(version.Print("avalanche"))
	log.SetFlags(log.Ltime | log.Lshortfile) // Show file name and line in logs.
	kingpin.CommandLine.Help = "avalanche - metrics test server\n" +
		"\n" +
		"Capable of generating metrics to server on \\metrics or send via Remote Write.\n" +
		"\n" +
		"\nOptionally, on top of the --value-interval, --series-interval, --metric-interval logic, you can specify advanced --series-operation-mode:\n" +
		"  double-halve:\n" +
		"    Alternately doubles and halves the series count at regular intervals.\n" +
		"    Usage: ./avalanche --series-operation-mode=double-halve --series-change-interval=30 --series-count=500\n" +
		"    Description: This mode alternately doubles and halves the series count at regular intervals.\n" +
		"                 The series count is doubled on one tick and halved on the next, ensuring it never drops below 1.\n" +
		"\n" +
		"  gradual-change:\n" +
		"    Gradually changes the series count by a fixed rate at regular intervals.\n" +
		"    Usage: ./avalanche --series-operation-mode=gradual-change --series-change-interval=30 --series-change-rate=100 --max-series-count=2000 --min-series-count=200\n" +
		"    Description: This mode gradually increases the series count by seriesChangeRate on each tick up to maxSeriesCount,\n" +
		"                 then decreases it back to the minSeriesCount, and repeats this cycle indefinitely.\n" +
		"                 The series count is incremented by seriesChangeRate on each tick, ensuring it never drops below 1." +
		"\n" +
		"  spike:\n" +
		"    Periodically spikes the series count by a given multiplier.\n" +
		"    Usage: ./avalanche --series-operation-mode=spike --series-change-interval=180 --series-count=100 --spike-multiplier=1.5\n" +
		"    Description: This mode periodically increases the series count by a spike multiplier on one tick and\n" +
		"                 then returns it to the original count on the next tick. This pattern repeats indefinitely,\n" +
		"                 creating a spiking effect in the series count.\n"

	cfg := metrics.NewConfigFromFlags(kingpin.Flag)
	port := kingpin.Flag("port", "Port to serve at").Default("9001").Int()
	remoteURL := kingpin.Flag("remote-url", "URL to send samples via remote_write API.").URL()
	// TODO(bwplotka): Kill pprof feature, you can install OSS continuous profiling easily instead.
	remotePprofURLs := kingpin.Flag("remote-pprof-urls", "a list of urls to download pprofs during the remote write: --remote-pprof-urls=http://127.0.0.1:10902/debug/pprof/heap --remote-pprof-urls=http://127.0.0.1:10902/debug/pprof/profile").URLList()
	remotePprofInterval := kingpin.Flag("remote-pprof-interval", "how often to download pprof profiles. When not provided it will download a profile once before the end of the test.").Duration()
	remoteBatchSize := kingpin.Flag("remote-batch-size", "how many samples to send with each remote_write API request.").Default("2000").Int()
	remoteRequestCount := kingpin.Flag("remote-requests-count", "How many requests to send in total to the remote_write API. Set to -1 to run indefinitely.").Default("100").Int()
	remoteReqsInterval := kingpin.Flag("remote-write-interval", "delay between each remote write request.").Default("100ms").Duration()
	remoteTenant := kingpin.Flag("remote-tenant", "Tenant ID to include in remote_write send").Default("0").String()
	tlsClientInsecure := kingpin.Flag("tls-client-insecure", "Skip certificate check on tls connection").Default("false").Bool()
	remoteTenantHeader := kingpin.Flag("remote-tenant-header", "Tenant ID to include in remote_write send. The default, is the default tenant header expected by Cortex.").Default("X-Scope-OrgID").String()
	// TODO(bwplotka): Make this a non-bool flag (e.g. out-of-order-min-time).
	outOfOrder := kingpin.Flag("remote-out-of-order", "Enable out-of-order timestamps in remote write requests").Default("true").Bool()

	kingpin.Parse()
	if err := cfg.Validate(); err != nil {
		kingpin.FatalUsage("configuration error: %v", err)
	}
	log.Println("initializing avalanche...")

	collector := metrics.NewCollector(*cfg)
	reg := prometheus.NewRegistry()
	reg.MustRegister(collector)

	var g run.Group
	g.Add(run.SignalHandler(context.Background(), os.Interrupt, syscall.SIGTERM))
	g.Add(collector.Run, collector.Stop)

	// One-off remote write send mode.
	if *remoteURL != nil {
		if (**remoteURL).Host == "" || (**remoteURL).Scheme == "" {
			log.Fatal("remote host and scheme can't be empty")
		}
		if *remoteBatchSize <= 0 {
			log.Fatal("remote send batch size should be more than zero")
		}

		config := &metrics.ConfigWrite{
			URL:             **remoteURL,
			RequestInterval: *remoteReqsInterval,
			BatchSize:       *remoteBatchSize,
			RequestCount:    *remoteRequestCount,
			UpdateNotify:    collector.UpdateNotifyCh(),
			Tenant:          *remoteTenant,
			TLSClientConfig: tls.Config{
				InsecureSkipVerify: *tlsClientInsecure,
			},
			TenantHeader: *remoteTenantHeader,
			OutOfOrder:   *outOfOrder,
		}

		// Collect Pprof during the write only if not collecting within a regular interval.
		if *remotePprofInterval == 0 {
			config.PprofURLs = *remotePprofURLs
		}

		var (
			wg   sync.WaitGroup
			done = make(chan struct{})
		)
		if *remotePprofInterval > 0 {
			if len(*remotePprofURLs) == 0 {
				log.Fatal("remote profiling interval specified without any remote pprof urls")
			}
			suffix := rand.Intn(1000)
			go func() {
				ticker := time.NewTicker(*remotePprofInterval)
				var dur time.Duration
			loop:
				for {
					<-ticker.C
					select {
					case <-done: // Prevents a panic when calling wg.Add(1) after calling wg.Wait().
						break loop
					default:
					}
					dur += *remotePprofInterval
					wg.Add(1)
					download.URLs(*remotePprofURLs, strconv.Itoa(suffix)+"-"+dur.String())
					wg.Done()
				}
			}()
		}

		ctx, cancel := context.WithCancel(context.Background())
		g.Add(func() error {
			if err := metrics.SendRemoteWrite(ctx, config, reg); err != nil {
				return err
			}
			if *remotePprofInterval > 0 {
				done <- struct{}{}
				wg.Wait()
			}
			return nil // One-off.
		}, func(error) { cancel() })
	}

	httpSrv := &http.Server{Addr: fmt.Sprintf(":%v", *port)}
	g.Add(func() error {
		fmt.Printf("Serving your metrics at :%v/metrics\n", *port)
		http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{
			EnableOpenMetrics: true,
		}))
		http.HandleFunc("/health", health.New(health.Health{}).Handler)
		return httpSrv.ListenAndServe()
	}, func(_ error) {
		_ = httpSrv.Shutdown(context.Background())
	})

	log.Println("starting avalanche...")
	if err := g.Run(); err != nil {
		//nolint:errcheck
		log.Fatalf("running avalanche failed %v", err)
	}
	log.Println("avalanche finished")
}
