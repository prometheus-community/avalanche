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
	"fmt"
	"log"
	"net/http"
	"os"
	"syscall"

	"github.com/nelkinda/health-go"
	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/prometheus-community/avalanche/metrics"
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
	remoteWriteConfig := metrics.NewWriteConfigFromFlags(kingpin.Flag)

	kingpin.Parse()
	if err := cfg.Validate(); err != nil {
		kingpin.FatalUsage("configuration error: %v", err)
	}
	if err := remoteWriteConfig.Validate(); err != nil {
		kingpin.FatalUsage("remote write config validation failed: %v", err)
	}

	collector := metrics.NewCollector(*cfg)
	reg := prometheus.NewRegistry()
	reg.MustRegister(collector)
	remoteWriteConfig.UpdateNotify = collector.UpdateNotifyCh()

	log.Println("initializing avalanche...")

	var g run.Group
	g.Add(run.SignalHandler(context.Background(), os.Interrupt, syscall.SIGTERM))
	g.Add(collector.Run, collector.Stop)

	// One-off remote write send mode.
	if remoteWriteConfig.URL != nil {
		ctx, cancel := context.WithCancel(context.Background())
		g.Add(func() error {
			if err := remoteWriteConfig.SendRemoteWrite(ctx, reg); err != nil {
				return err
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
