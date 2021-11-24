package metrics

import (
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/nelkinda/health-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	promRegistry = prometheus.NewRegistry() // local Registry so we don't get Go metrics, etc.
	valGenerator = rand.New(rand.NewSource(time.Now().UnixNano()))
	metrics      = make([]*prometheus.GaugeVec, 0)
	metricsMux   = &sync.Mutex{}
)

func registerMetrics(metricCount int, metricLength int, metricCycle int, labelKeys []string) {
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

func seriesLabels(seriesID int, cycleID int, labelKeys []string, labelValues []string) prometheus.Labels {
	labels := prometheus.Labels{
		"series_id": fmt.Sprintf("%v", seriesID),
		"cycle_id":  fmt.Sprintf("%v", cycleID),
	}

	for idx, key := range labelKeys {
		labels[key] = labelValues[idx]
	}

	return labels
}

func deleteValues(labelKeys []string, labelValues []string, seriesCount int, seriesCycle int) {
	for _, metric := range metrics {
		for idx := 0; idx < seriesCount; idx++ {
			labels := seriesLabels(idx, seriesCycle, labelKeys, labelValues)
			metric.Delete(labels)
		}
	}
}

func cycleValues(labelKeys []string, labelValues []string, seriesCount int, seriesCycle int) {
	for _, metric := range metrics {
		for idx := 0; idx < seriesCount; idx++ {
			labels := seriesLabels(idx, seriesCycle, labelKeys, labelValues)
			metric.With(labels).Set(float64(valGenerator.Intn(100)))
		}
	}
}

func maybeStartTicker(interval int, dontStart bool) (*time.Ticker, error) {
	if dontStart {
		t := time.NewTicker(time.Duration(1))
		// Since a nil ticker can't actually be used, immediately stop it instead
		t.Stop()
		return t, nil
	}
	return time.NewTicker(time.Duration(interval) * time.Second), nil
}

// RunMetrics creates a set of Prometheus test series that update over time
func RunMetrics(metricCount int, labelCount int, seriesCount int, metricLength int, labelLength int, valueInterval int, seriesInterval int, metricInterval int, ignoredIntervals map[string]bool, constLabels []string, stop chan struct{}) (chan struct{}, error) {
	labelKeys := make([]string, labelCount, labelCount)
	for idx := 0; idx < labelCount; idx++ {
		labelKeys[idx] = fmt.Sprintf("label_key_%s_%v", strings.Repeat("k", labelLength), idx)
	}
	labelValues := make([]string, labelCount, labelCount)
	for idx := 0; idx < labelCount; idx++ {
		labelValues[idx] = fmt.Sprintf("label_val_%s_%v", strings.Repeat("v", labelLength), idx)
	}
	for _, cLabel := range constLabels {
		split := strings.Split(cLabel, "=")
		if len(split) != 2 {
			return make(chan struct{}, 1), errors.New(fmt.Sprintf("Constant label argument must have format labelName=labelValue but got %s", cLabel))
		}
		labelKeys = append(labelKeys, split[0])
		labelValues = append(labelValues, split[1])
	}

	metricCycle := 0
	seriesCycle := 0
	registerMetrics(metricCount, metricLength, metricCycle, labelKeys)
	cycleValues(labelKeys, labelValues, seriesCount, seriesCycle)

	valueTick, err := maybeStartTicker(valueInterval, ignoredIntervals["value"])
	if err != nil {
		return make(chan struct{}, 1), err
	}
	seriesTick, err := maybeStartTicker(seriesInterval, ignoredIntervals["series"])
	if err != nil {
		return make(chan struct{}, 1), err
	}
	metricTick, err := maybeStartTicker(metricInterval, ignoredIntervals["metric"])
	if err != nil {
		return make(chan struct{}, 1), err
	}
	updateNotify := make(chan struct{}, 1)

	go func() {
		for tick := range valueTick.C {
			fmt.Printf("%v: refreshing metric values\n", tick)
			metricsMux.Lock()
			cycleValues(labelKeys, labelValues, seriesCount, seriesCycle)
			metricsMux.Unlock()
			select {
			case updateNotify <- struct{}{}:
			default:
			}
		}
	}()

	go func() {
		for tick := range seriesTick.C {
			fmt.Printf("%v: refreshing series cycle\n", tick)
			metricsMux.Lock()
			deleteValues(labelKeys, labelValues, seriesCount, seriesCycle)
			seriesCycle++
			cycleValues(labelKeys, labelValues, seriesCount, seriesCycle)
			metricsMux.Unlock()
			select {
			case updateNotify <- struct{}{}:
			default:
			}
		}
	}()

	go func() {
		for tick := range metricTick.C {
			fmt.Printf("%v: refreshing metric cycle\n", tick)
			metricsMux.Lock()
			metricCycle++
			unregisterMetrics()
			registerMetrics(metricCount, metricLength, metricCycle, labelKeys)
			cycleValues(labelKeys, labelValues, seriesCount, seriesCycle)
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
