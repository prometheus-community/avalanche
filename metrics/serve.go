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

package metrics

import (
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	health "github.com/nelkinda/health-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	promRegistry = prometheus.NewRegistry() // local Registry so we don't get Go metrics, etc.
	valGenerator = rand.New(rand.NewSource(time.Now().UnixNano()))
	metrics      = make([]*prometheus.GaugeVec, 0)
	metricsMux   = &sync.Mutex{}
)

func registerMetrics(metricCount, metricLength, metricCycle int, labelKeys []string) {
	metrics = make([]*prometheus.GaugeVec, metricCount)
	for idx := 0; idx < metricCount; idx++ {
		gauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: fmt.Sprintf("avalanche_metric_%s_%v_%v", strings.Repeat("m", metricLength), metricCycle, idx),
			Help: "A tasty metric morsel",
		}, append([]string{"series_id", "cycle_id"}, labelKeys...))
		promRegistry.MustRegister(gauge)
		metrics[idx] = gauge
	}
}

func unregisterMetrics() {
	for _, metric := range metrics {
		promRegistry.Unregister(metric)
	}
}

func seriesLabels(seriesID, cycleID int, labelKeys, labelValues []string) prometheus.Labels {
	labels := prometheus.Labels{
		"series_id": fmt.Sprintf("%v", seriesID),
		"cycle_id":  fmt.Sprintf("%v", cycleID),
	}

	for idx, key := range labelKeys {
		labels[key] = labelValues[idx]
	}

	return labels
}

func deleteValues(labelKeys, labelValues []string, seriesCount, seriesCycle int) {
	for _, metric := range metrics {
		for idx := 0; idx < seriesCount; idx++ {
			labels := seriesLabels(idx, seriesCycle, labelKeys, labelValues)
			metric.Delete(labels)
		}
	}
}

func cycleValues(labelKeys, labelValues []string, seriesCount, seriesCycle int) {
	for _, metric := range metrics {
		for idx := 0; idx < seriesCount; idx++ {
			labels := seriesLabels(idx, seriesCycle, labelKeys, labelValues)
			metric.With(labels).Set(float64(valGenerator.Intn(100)))
		}
	}
}

// RunMetrics creates a set of Prometheus test series that update over time
func RunMetrics(metricCount, labelCount, seriesCount, seriesChangeRate, maxSeriesCount, minSeriesCount, metricLength, labelLength, valueInterval, seriesInterval, metricInterval, seriesChangeInterval int, seriesOperationMode string, constLabels []string, stop chan struct{}) (chan struct{}, error) {
	labelKeys := make([]string, labelCount)
	for idx := 0; idx < labelCount; idx++ {
		labelKeys[idx] = fmt.Sprintf("label_key_%s_%v", strings.Repeat("k", labelLength), idx)
	}
	labelValues := make([]string, labelCount)
	for idx := 0; idx < labelCount; idx++ {
		labelValues[idx] = fmt.Sprintf("label_val_%s_%v", strings.Repeat("v", labelLength), idx)
	}
	for _, cLabel := range constLabels {
		split := strings.Split(cLabel, "=")
		if len(split) != 2 {
			return make(chan struct{}, 1), fmt.Errorf("Constant label argument must have format labelName=labelValue but got %s", cLabel)
		}
		labelKeys = append(labelKeys, split[0])
		labelValues = append(labelValues, split[1])
	}

	metricCycle := 0
	seriesCycle := 0
	valueTick := time.NewTicker(time.Duration(valueInterval) * time.Second)
	seriesTick := time.NewTicker(time.Duration(seriesInterval) * time.Second)
	metricTick := time.NewTicker(time.Duration(metricInterval) * time.Second)
	changeSeriesTick := time.NewTicker(time.Duration(seriesChangeInterval) * time.Second)
	updateNotify := make(chan struct{}, 1)

	var currentSeriesCount int

	switch seriesOperationMode {
	case "double-halve":
		currentSeriesCount = seriesCount
		registerMetrics(metricCount, metricLength, metricCycle, labelKeys)
		cycleValues(labelKeys, labelValues, currentSeriesCount, seriesCycle)
		seriesIncrease := true
		go func() {
			for tick := range changeSeriesTick.C {
				metricsMux.Lock()
				unregisterMetrics()
				registerMetrics(metricCount, metricLength, metricCycle, labelKeys)
				cycleValues(labelKeys, labelValues, currentSeriesCount, seriesCycle)
				metricsMux.Unlock()
				if seriesIncrease {
					currentSeriesCount *= 2
				} else {
					currentSeriesCount /= 2
					if currentSeriesCount < 1 {
						currentSeriesCount = 1
					}
				}

				fmt.Printf("%v: Adjusting series count. New count: %d\n", tick, currentSeriesCount)

				seriesIncrease = !seriesIncrease

				select {
				case updateNotify <- struct{}{}:
				default:
				}
			}
		}()

	case "gradual-change":
		if minSeriesCount >= maxSeriesCount {
			return nil, fmt.Errorf("error: minSeriesCount must be less than maxSeriesCount, got %d and %d", minSeriesCount, maxSeriesCount)
		}
		seriesIncrease := true
		currentSeriesCount = minSeriesCount
		registerMetrics(metricCount, metricLength, metricCycle, labelKeys)
		cycleValues(labelKeys, labelValues, currentSeriesCount, seriesCycle)
		go func() {
			for tick := range changeSeriesTick.C {
				metricsMux.Lock()
				unregisterMetrics()
				registerMetrics(metricCount, metricLength, metricCycle, labelKeys)
				cycleValues(labelKeys, labelValues, currentSeriesCount, seriesCycle)
				metricsMux.Unlock()
				if seriesIncrease {
					currentSeriesCount += seriesChangeRate
					if currentSeriesCount >= maxSeriesCount {
						currentSeriesCount = maxSeriesCount
						seriesIncrease = false
					}
				} else {
					currentSeriesCount -= seriesChangeRate
					if currentSeriesCount <= minSeriesCount {
						currentSeriesCount = minSeriesCount
						seriesIncrease = true
					}
				}
				fmt.Printf("%v: Adjusting series count. New count: %d\n", tick, currentSeriesCount)

				select {
				case updateNotify <- struct{}{}:
				default:
				}
			}
		}()
	default:
		currentSeriesCount = seriesCount
		registerMetrics(metricCount, metricLength, metricCycle, labelKeys)
		cycleValues(labelKeys, labelValues, seriesCount, seriesCycle)
		go func() {
			for tick := range metricTick.C {
				metricsMux.Lock()
				fmt.Printf("%v: refreshing metric cycle\n", tick)
				metricCycle++
				unregisterMetrics()
				registerMetrics(metricCount, metricLength, metricCycle, labelKeys)
				cycleValues(labelKeys, labelValues, currentSeriesCount, seriesCycle)
				metricsMux.Unlock()

				select {
				case updateNotify <- struct{}{}:
				default:
				}
			}
		}()
	}

	go func() {
		for tick := range valueTick.C {
			metricsMux.Lock()
			fmt.Printf("%v: refreshing metric values\n", tick)
			cycleValues(labelKeys, labelValues, currentSeriesCount, seriesCycle)
			metricsMux.Unlock()

			select {
			case updateNotify <- struct{}{}:
			default:
			}
		}
	}()

	go func() {
		for tick := range seriesTick.C {
			metricsMux.Lock()
			fmt.Printf("%v: refreshing series cycle\n", tick)
			deleteValues(labelKeys, labelValues, currentSeriesCount, seriesCycle)
			seriesCycle++
			cycleValues(labelKeys, labelValues, currentSeriesCount, seriesCycle)
			metricsMux.Unlock()

			select {
			case updateNotify <- struct{}{}:
			default:
			}
		}
	}()

	go func() {
		<-stop
		valueTick.Stop()
		seriesTick.Stop()
		metricTick.Stop()
		changeSeriesTick.Stop()
	}()

	return updateNotify, nil

}

// ServeMetrics serves a prometheus metrics endpoint with test series
func ServeMetrics(port int) error {
	http.Handle("/metrics", promhttp.HandlerFor(promRegistry, promhttp.HandlerOpts{}))
	h := health.New(health.Health{})
	http.HandleFunc("/health", h.Handler)
	err := http.ListenAndServe(fmt.Sprintf(":%v", port), nil)
	if err != nil {
		return err
	}

	return nil
}
