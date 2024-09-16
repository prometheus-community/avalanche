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
	"math"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"gopkg.in/alecthomas/kingpin.v2"
)

type Collector struct {
	cfg    Config
	valGen *rand.Rand

	gauges           []*prometheus.GaugeVec
	counters         []*prometheus.CounterVec
	histograms       []*prometheus.HistogramVec
	nativeHistograms []*prometheus.HistogramVec
	summaries        []*prometheus.SummaryVec

	updateNotifyCh   chan struct{}
	stopCh           chan struct{}
	valueTick        *time.Ticker
	seriesTick       *time.Ticker
	metricTick       *time.Ticker
	changeSeriesTick *time.Ticker

	mu sync.Mutex
}

// NewCollector returns Prometheus collector that can be registered in registry
// that handles metric creation and changes, based on the given configuration.
func NewCollector(cfg Config) *Collector {
	if cfg.GaugeMetricCount == 0 {
		cfg.GaugeMetricCount = cfg.MetricCount // Handle deprecated field.
	}

	c := &Collector{
		cfg:            cfg,
		valGen:         rand.New(rand.NewSource(time.Now().UnixNano())),
		updateNotifyCh: make(chan struct{}, 1),
		stopCh:         make(chan struct{}),
	}

	if cfg.ValueInterval > 0 {
		c.valueTick = time.NewTicker(time.Duration(cfg.ValueInterval) * time.Second)
	}
	if cfg.SeriesInterval > 0 {
		c.seriesTick = time.NewTicker(time.Duration(cfg.SeriesInterval) * time.Second)
	}
	if cfg.MetricInterval > 0 {
		c.metricTick = time.NewTicker(time.Duration(cfg.MetricInterval) * time.Second)
	}
	if cfg.SeriesChangeInterval > 0 {
		c.changeSeriesTick = time.NewTicker(time.Duration(cfg.SeriesChangeInterval) * time.Second)
	}
	return c
}

func (c *Collector) UpdateNotifyCh() chan struct{} {
	return c.updateNotifyCh
}

type Config struct {
	MetricCount, GaugeMetricCount, CounterMetricCount, HistogramMetricCount, NativeHistogramMetricCount, SummaryMetricCount int
	HistogramBuckets                                                                                                        int

	LabelCount, SeriesCount        int
	MaxSeriesCount, MinSeriesCount int
	MetricLength, LabelLength      int

	ValueInterval, SeriesInterval, MetricInterval, SeriesChangeInterval, SeriesChangeRate int

	SpikeMultiplier     float64
	SeriesOperationMode string
	ConstLabels         []string
}

func NewConfigFromFlags(flagReg func(name, help string) *kingpin.FlagClause) *Config {
	cfg := &Config{}
	flagReg("metric-count", "Number of gauge metrics to serve. DEPRECATED use --gauge-metric-count instead").Default("0").
		IntVar(&cfg.MetricCount)
	// 500 in total by default, just a healthy distribution of types.
	flagReg("gauge-metric-count", "Number of gauge metrics to serve.").Default("200").
		IntVar(&cfg.GaugeMetricCount)
	flagReg("counter-metric-count", "Number of counter metrics to serve.").Default("200").
		IntVar(&cfg.CounterMetricCount)
	flagReg("histogram-metric-count", "Number of explicit (classic) histogram metrics to serve.").Default("10").
		IntVar(&cfg.HistogramMetricCount)
	flagReg("histogram-metric-bucket-count", "Number of explicit buckets (classic) histogram metrics.").Default("8").
		IntVar(&cfg.HistogramMetricCount)
	flagReg("native-histogram-metric-count", "Number of native (exponential) histogram metrics to serve.").Default("0").
		IntVar(&cfg.HistogramBuckets)
	flagReg("summary-metric-count", "Number of summary metrics to serve.").Default("0").
		IntVar(&cfg.SummaryMetricCount)

	flagReg("label-count", "Number of labels per-metric.").Default("10").
		IntVar(&cfg.LabelCount)
	flagReg("series-count", "Number of series per-metric. This excludes the extra series (e.g. _bucket) that will be added for complex types like classic histograms and summaries.").Default("100").
		IntVar(&cfg.SeriesCount)
	flagReg("max-series-count", "Maximum number of series to serve. Applies to 'gradual-change' mode.").Default("1000").
		IntVar(&cfg.MaxSeriesCount)
	flagReg("min-series-count", "Minimum number of series to serve. Applies to 'gradual-change' mode.").Default("100").
		IntVar(&cfg.MinSeriesCount)
	flagReg("spike-multiplier", "Multiplier for the spike mode.").Default("1.5").
		Float64Var(&cfg.SpikeMultiplier)
	flagReg("metricname-length", "Modify length of metric names.").Default("5").
		IntVar(&cfg.MetricLength)
	flagReg("labelname-length", "Modify length of label names.").Default("5").
		IntVar(&cfg.LabelLength)
	flagReg("const-label", "Constant label to add to every metric. Format is labelName=labelValue. Flag can be specified multiple times.").
		StringsVar(&cfg.ConstLabels)

	flagReg("value-interval", "Change series values every {interval} seconds. 0 means no change.").Default("30").
		IntVar(&cfg.ValueInterval)
	flagReg("series-interval", "Change series_id label values every {interval} seconds. 0 means no change.").Default("60").
		IntVar(&cfg.SeriesInterval)
	flagReg("metric-interval", "Change __name__ label values every {interval} seconds. 0 means no change.").Default("120").
		IntVar(&cfg.MetricInterval)
	flagReg("series-change-interval", "Change the number of series every {interval} seconds. Applies to 'gradual-change', 'double-halve' and 'spike' modes. 0 means no change.").Default("30").
		IntVar(&cfg.SeriesChangeInterval)
	flagReg("series-change-rate", "The rate at which the number of active series changes over time. Applies to 'gradual-change' mode.").Default("100").
		IntVar(&cfg.SeriesChangeRate)

	flagReg("series-operation-mode", "Mode of operation: 'gradual-change', 'double-halve', 'spike'").Default("default").
		StringVar(&cfg.SeriesOperationMode)
	return cfg
}

func (c Config) Validate() error {
	if c.MaxSeriesCount <= c.MinSeriesCount {
		return fmt.Errorf("--max-series-count (%d) must be greater than --min-series-count (%d)", c.MaxSeriesCount, c.MinSeriesCount)
	}
	if c.MinSeriesCount < 0 {
		return fmt.Errorf("--min-series-count must be 0 or higher, got %d", c.MinSeriesCount)
	}
	if c.MetricCount > 0 && c.GaugeMetricCount > 0 {
		return fmt.Errorf("--metric-count (set to %v) is deprecated and it means the same as --gauge-metric-count (set to %v); both can't be used in the same time", c.MetricCount, c.GaugeMetricCount)
	}
	for _, cLabel := range c.ConstLabels {
		split := strings.Split(cLabel, "=")
		if len(split) != 2 {
			return fmt.Errorf("constant label argument must have format labelName=labelValue but got %s", cLabel)
		}
	}
	if c.SeriesOperationMode == "gradual-change" && c.SeriesChangeRate <= 0 {
		return fmt.Errorf("--series-change-rate must be greater than 0, got %d", c.SeriesChangeRate)
	}
	if c.SeriesOperationMode == "spike" && c.SpikeMultiplier < 1 {
		return fmt.Errorf("--spike-multiplier must be greater than or equal to 1, got %f", c.SpikeMultiplier)
	}
	return nil
}

// Describe is used when registering metrics. It's noop avoiding us to have an easier dynamicity.
// No descriptors allow this collector to be "unchecked", but more efficient with what we try to do here.
func (c *Collector) Describe(chan<- *prometheus.Desc) {}

func (c *Collector) Collect(metricCh chan<- prometheus.Metric) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, m := range c.gauges {
		m.Collect(metricCh)
	}
	for _, m := range c.counters {
		m.Collect(metricCh)
	}
	for _, m := range c.histograms {
		m.Collect(metricCh)
	}
	for _, m := range c.nativeHistograms {
		m.Collect(metricCh)
	}
	for _, m := range c.summaries {
		m.Collect(metricCh)
	}
}

func help(mName string) string {
	return fmt.Sprintf("Metric %v is generated by https://github.com/prometheus-community/avalanche project allowing you to load test your Prometheus or Prometheus-compatible systems. It's not too long, not too short to simulate, often chunky descriptions on user metrics. It also contains metric name, so help is slighly different across metrics.", mName)
}

func (c *Collector) recreateMetrics(metricCycle int, labelKeys []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for id := range c.gauges {
		mName := fmt.Sprintf("avalanche_gauge_metric_%s_%v_%v", strings.Repeat("m", c.cfg.MetricLength), metricCycle, id)
		gauge := prometheus.NewGaugeVec(
			prometheus.GaugeOpts{Name: mName, Help: help(mName)},
			append([]string{"series_id", "cycle_id"}, labelKeys...),
		)
		c.gauges[id] = gauge
	}
	for id := range c.counters {
		mName := fmt.Sprintf("avalanche_counter_metric_%s_%v_%v_total", strings.Repeat("m", c.cfg.MetricLength), metricCycle, id)
		counter := prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: mName, Help: help(mName)},
			append([]string{"series_id", "cycle_id"}, labelKeys...),
		)
		c.counters[id] = counter
	}

	bkts := make([]float64, c.cfg.HistogramBuckets)
	for i := range bkts {
		bkts[i] = 0.0001 * math.Pow10(i)
	}
	for id := range c.histograms {
		mName := fmt.Sprintf("avalanche_histogram_metric_%s_%v_%v", strings.Repeat("m", c.cfg.MetricLength), metricCycle, id)
		histogram := prometheus.NewHistogramVec(
			prometheus.HistogramOpts{Name: mName, Help: help(mName), Buckets: bkts},
			append([]string{"series_id", "cycle_id"}, labelKeys...),
		)
		c.histograms[id] = histogram
	}

	for id := range c.nativeHistograms {
		mName := fmt.Sprintf("avalanche_native_histogram_metric_%s_%v_%v", strings.Repeat("m", c.cfg.MetricLength), metricCycle, id)
		histogram := prometheus.NewHistogramVec(
			prometheus.HistogramOpts{Name: mName, Help: help(mName), NativeHistogramBucketFactor: 1.1},
			append([]string{"series_id", "cycle_id"}, labelKeys...),
		)
		c.nativeHistograms[id] = histogram
	}

	for id := range c.summaries {
		mName := fmt.Sprintf("avalanche_summary_metric_%s_%v_%v", strings.Repeat("m", c.cfg.MetricLength), metricCycle, id)
		summary := prometheus.NewSummaryVec(
			prometheus.SummaryOpts{Name: mName, Help: help(mName)},
			append([]string{"series_id", "cycle_id"}, labelKeys...),
		)
		c.summaries[id] = summary
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

type seriesDeleter interface {
	Delete(labels prometheus.Labels) bool
}

func deleteValues[T seriesDeleter](metrics []T, labelKeys, labelValues []string, seriesCount, seriesCycle int) {
	for _, metric := range metrics {
		for idx := 0; idx < seriesCount; idx++ {
			labels := seriesLabels(idx, seriesCycle, labelKeys, labelValues)
			metric.Delete(labels)
		}
	}
}

func (c *Collector) cycleValues(labelKeys, labelValues []string, seriesCount, seriesCycle int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, metric := range c.gauges {
		for idx := 0; idx < seriesCount; idx++ {
			labels := seriesLabels(idx, seriesCycle, labelKeys, labelValues)
			metric.With(labels).Set(float64(c.valGen.Intn(100)))
		}
	}
	for _, metric := range c.counters {
		for idx := 0; idx < seriesCount; idx++ {
			labels := seriesLabels(idx, seriesCycle, labelKeys, labelValues)
			metric.With(labels).Add(float64(c.valGen.Intn(100)))
		}
	}
	for _, metric := range c.histograms {
		for idx := 0; idx < seriesCount; idx++ {
			labels := seriesLabels(idx, seriesCycle, labelKeys, labelValues)
			metric.With(labels).Observe(float64(c.valGen.Intn(100)))
		}
	}
	for _, metric := range c.nativeHistograms {
		for idx := 0; idx < seriesCount; idx++ {
			labels := seriesLabels(idx, seriesCycle, labelKeys, labelValues)
			metric.With(labels).Observe(float64(c.valGen.Intn(100)))
		}
	}
	for _, metric := range c.summaries {
		for idx := 0; idx < seriesCount; idx++ {
			labels := seriesLabels(idx, seriesCycle, labelKeys, labelValues)
			metric.With(labels).Observe(float64(c.valGen.Intn(100)))
		}
	}
}

func (c *Collector) handleValueTicks(labelKeys, labelValues *[]string, currentSeriesCount, seriesCycle *int) {
	if c.valueTick == nil {
		return
	}
	for tick := range c.valueTick.C {
		fmt.Printf("%v: refreshing metric values\n", tick)
		c.cycleValues(*labelKeys, *labelValues, *currentSeriesCount, *seriesCycle)

		select {
		case c.updateNotifyCh <- struct{}{}:
		default:
		}
	}
}

func (c *Collector) handleSeriesTicks(labelKeys, labelValues *[]string, currentSeriesCount, seriesCycle *int) {
	if c.seriesTick == nil {
		return
	}
	for tick := range c.seriesTick.C {
		c.mu.Lock()
		fmt.Printf("%v: refreshing series cycle\n", tick)
		deleteValues(c.gauges, *labelKeys, *labelValues, *currentSeriesCount, *seriesCycle)
		deleteValues(c.counters, *labelKeys, *labelValues, *currentSeriesCount, *seriesCycle)
		deleteValues(c.histograms, *labelKeys, *labelValues, *currentSeriesCount, *seriesCycle)
		deleteValues(c.nativeHistograms, *labelKeys, *labelValues, *currentSeriesCount, *seriesCycle)
		deleteValues(c.summaries, *labelKeys, *labelValues, *currentSeriesCount, *seriesCycle)
		(*seriesCycle)++
		c.mu.Unlock()
		c.cycleValues(*labelKeys, *labelValues, *currentSeriesCount, *seriesCycle)

		select {
		case c.updateNotifyCh <- struct{}{}:
		default:
		}
	}
}

func (c *Collector) handleMetricTicks(metricCycle *int, labelKeys *[]string) {
	if c.metricTick == nil {
		return
	}

	for tick := range c.metricTick.C {
		fmt.Printf("%v: refreshing metric cycle\n", tick)
		(*metricCycle)++
		c.recreateMetrics(*metricCycle, *labelKeys)
		select {
		case c.updateNotifyCh <- struct{}{}:
		default:
		}
	}
}

func changeSeriesGradual(seriesChangeRate, maxSeriesCount, minSeriesCount, currentSeriesCount *int, seriesIncrease *bool) {
	fmt.Printf("Current series count: %d\n", *currentSeriesCount)
	if *seriesIncrease {
		*currentSeriesCount += *seriesChangeRate
		if *currentSeriesCount >= *maxSeriesCount {
			*currentSeriesCount = *maxSeriesCount
			*seriesIncrease = false
		}
	} else {
		*currentSeriesCount -= *seriesChangeRate
		if *currentSeriesCount < *minSeriesCount {
			*currentSeriesCount = *minSeriesCount
			*seriesIncrease = true
		}
	}
}

func changeSeriesDoubleHalve(currentSeriesCount *int, seriesIncrease *bool) {
	if *seriesIncrease {
		*currentSeriesCount *= 2
	} else {
		*currentSeriesCount /= 2
		if *currentSeriesCount < 1 {
			*currentSeriesCount = 1
		}
	}
	*seriesIncrease = !*seriesIncrease
}

func (c *Collector) handleDoubleHalveMode(metricCycle, seriesCycle int, labelKeys, labelValues []string, currentSeriesCount *int) {
	if c.changeSeriesTick == nil {
		return
	}

	seriesIncrease := true
	for tick := range c.changeSeriesTick.C {
		c.recreateMetrics(metricCycle, labelKeys)
		c.cycleValues(labelKeys, labelValues, *currentSeriesCount, seriesCycle)

		changeSeriesDoubleHalve(currentSeriesCount, &seriesIncrease)

		fmt.Printf("%v: Adjusting series count. New count: %d\n", tick, *currentSeriesCount)

		select {
		case c.updateNotifyCh <- struct{}{}:
		default:
		}
	}
}

func (c *Collector) handleGradualChangeMode(metricCycle, seriesCycle int, labelKeys, labelValues []string, seriesCount *int) {
	if c.changeSeriesTick == nil {
		return
	}

	*seriesCount = c.cfg.MinSeriesCount
	seriesIncrease := true
	for tick := range c.changeSeriesTick.C {
		c.recreateMetrics(metricCycle, labelKeys)
		c.cycleValues(labelKeys, labelValues, *seriesCount, seriesCycle)

		changeSeriesGradual(&c.cfg.SeriesChangeRate, &c.cfg.MaxSeriesCount, &c.cfg.MinSeriesCount, seriesCount, &seriesIncrease)

		fmt.Printf("%v: Adjusting series count. New count: %d\n", tick, *seriesCount)

		select {
		case c.updateNotifyCh <- struct{}{}:
		default:
		}
	}
}

func (c *Collector) handleSpikeMode(metricCycle, seriesCycle int, labelKeys, labelValues []string, currentSeriesCount *int, spikeMultiplier float64) {
	if c.changeSeriesTick == nil {
		return
	}

	initialSeriesCount := *currentSeriesCount
	for tick := range c.changeSeriesTick.C {
		c.recreateMetrics(metricCycle, labelKeys)
		c.cycleValues(labelKeys, labelValues, *currentSeriesCount, seriesCycle)

		if *currentSeriesCount > initialSeriesCount {
			*currentSeriesCount = initialSeriesCount
		} else {
			*currentSeriesCount = int(float64(initialSeriesCount) * spikeMultiplier)
		}

		fmt.Printf("%v: Adjusting series count. New count: %d\n", tick, *currentSeriesCount)

		select {
		case c.updateNotifyCh <- struct{}{}:
		default:
		}
	}
}

// Run creates a set of Prometheus test series that update over time.
// NOTE: Only one execution of RunMetrics is currently expected.
func (c *Collector) Run() error {
	labelKeys := make([]string, c.cfg.LabelCount)
	for idx := 0; idx < c.cfg.LabelCount; idx++ {
		labelKeys[idx] = fmt.Sprintf("label_key_%s_%v", strings.Repeat("k", c.cfg.LabelLength), idx)
	}
	labelValues := make([]string, c.cfg.LabelCount)
	for idx := 0; idx < c.cfg.LabelCount; idx++ {
		labelValues[idx] = fmt.Sprintf("label_val_%s_%v", strings.Repeat("v", c.cfg.LabelLength), idx)
	}
	for _, cLabel := range c.cfg.ConstLabels {
		split := strings.Split(cLabel, "=")
		labelKeys = append(labelKeys, split[0])
		labelValues = append(labelValues, split[1])
	}

	metricCycle := 0
	seriesCycle := 0
	currentSeriesCount := c.cfg.SeriesCount

	c.mu.Lock()
	c.gauges = make([]*prometheus.GaugeVec, c.cfg.GaugeMetricCount)
	c.counters = make([]*prometheus.CounterVec, c.cfg.CounterMetricCount)
	c.histograms = make([]*prometheus.HistogramVec, c.cfg.HistogramMetricCount)
	c.nativeHistograms = make([]*prometheus.HistogramVec, c.cfg.NativeHistogramMetricCount)
	c.summaries = make([]*prometheus.SummaryVec, c.cfg.SummaryMetricCount)
	c.mu.Unlock()

	switch c.cfg.SeriesOperationMode {
	case "double-halve":
		c.recreateMetrics(metricCycle, labelKeys)
		c.cycleValues(labelKeys, labelValues, currentSeriesCount, seriesCycle)
		fmt.Printf("Starting double-halve mode; starting series: %d, change series interval: %d seconds\n", currentSeriesCount, c.cfg.SeriesChangeInterval)
		go c.handleDoubleHalveMode(metricCycle, seriesCycle, labelKeys, labelValues, &currentSeriesCount)
		go c.handleValueTicks(&labelKeys, &labelValues, &currentSeriesCount, &seriesCycle)
		go c.handleSeriesTicks(&labelKeys, &labelValues, &currentSeriesCount, &seriesCycle)
		go c.handleMetricTicks(&metricCycle, &labelKeys)

	case "gradual-change":
		fmt.Printf("Starting gradual-change mode; min series: %d, max series: %d, series change rate: %d, change series interval: %d seconds\n", c.cfg.MinSeriesCount, c.cfg.MaxSeriesCount, c.cfg.SeriesChangeRate, c.cfg.SeriesChangeInterval)

		c.recreateMetrics(metricCycle, labelKeys)
		c.cycleValues(labelKeys, labelValues, c.cfg.MinSeriesCount, seriesCycle)
		go c.handleGradualChangeMode(metricCycle, seriesCycle, labelKeys, labelValues, &currentSeriesCount)
		go c.handleValueTicks(&labelKeys, &labelValues, &currentSeriesCount, &seriesCycle)
		go c.handleSeriesTicks(&labelKeys, &labelValues, &currentSeriesCount, &seriesCycle)
		go c.handleMetricTicks(&metricCycle, &labelKeys)

	case "spike":
		c.recreateMetrics(metricCycle, labelKeys)
		c.cycleValues(labelKeys, labelValues, currentSeriesCount, seriesCycle)
		fmt.Printf("Starting spike mode; initial series: %d, spike multiplier: %f, spike interval: %v\n", currentSeriesCount, c.cfg.SpikeMultiplier, c.cfg.SeriesChangeInterval)
		go c.handleSpikeMode(metricCycle, seriesCycle, labelKeys, labelValues, &currentSeriesCount, c.cfg.SpikeMultiplier)
		go c.handleValueTicks(&labelKeys, &labelValues, &currentSeriesCount, &seriesCycle)
		go c.handleSeriesTicks(&labelKeys, &labelValues, &currentSeriesCount, &seriesCycle)
		go c.handleMetricTicks(&metricCycle, &labelKeys)

	default:
		c.recreateMetrics(metricCycle, labelKeys)
		c.cycleValues(labelKeys, labelValues, currentSeriesCount, seriesCycle)
		go c.handleValueTicks(&labelKeys, &labelValues, &currentSeriesCount, &seriesCycle)
		go c.handleSeriesTicks(&labelKeys, &labelValues, &currentSeriesCount, &seriesCycle)
		go c.handleMetricTicks(&metricCycle, &labelKeys)
	}

	<-c.stopCh
	return nil
}

func (c *Collector) Stop(_ error) {
	if c.valueTick != nil {
		c.valueTick.Stop()
	}
	if c.seriesTick != nil {
		c.seriesTick.Stop()
	}
	if c.metricTick != nil {
		c.metricTick.Stop()
	}
	if c.changeSeriesTick != nil {
		c.changeSeriesTick.Stop()
	}
	close(c.stopCh)
}
