// Copyright 2024 The Prometheus Authors
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
	"strconv"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to count the series in the registry
func countSeries(t *testing.T, registry *prometheus.Registry) (seriesCount int) {
	t.Helper()

	metricsFamilies, err := registry.Gather()
	assert.NoError(t, err)

	for _, mf := range metricsFamilies {
		for range mf.Metric {
			seriesCount++
		}
	}
	return seriesCount
}

// countSeriesTypes gives exact count of all types. For complex types that are represented by counters in Prometheus
// data model (and text exposition formats), we count all individual resulting series.
func countSeriesTypes(t *testing.T, registry *prometheus.Registry) (gauges, counters, histograms, nhistograms, summaries int) {
	t.Helper()

	metricsFamilies, err := registry.Gather()
	assert.NoError(t, err)

	for _, mf := range metricsFamilies {
		for _, m := range mf.Metric {
			switch mf.GetType() {
			case io_prometheus_client.MetricType_GAUGE:
				gauges++
			case io_prometheus_client.MetricType_COUNTER:
				counters++
			case io_prometheus_client.MetricType_HISTOGRAM:
				if bkts := len(m.GetHistogram().Bucket); bkts == 0 {
					nhistograms++
				} else {
					histograms += 2 // count and sum.
					histograms += len(m.GetHistogram().GetBucket())
					if m.GetHistogram().GetBucket()[bkts-1].GetUpperBound() != math.Inf(+1) {
						// In the proto model we don't put explicit +Inf bucket, unless there is an exemplar,
						// but it will appear as series in text format and Prometheus model. Account for that.
						histograms++
					}
				}
			case io_prometheus_client.MetricType_SUMMARY:
				summaries += 2 // count and sum.
				summaries += len(m.GetSummary().GetQuantile())
			default:
				t.Fatalf("unknown metric type found %v", mf.GetType())
			}
		}
	}
	return gauges, counters, histograms, nhistograms, summaries
}

func TestRunMetrics(t *testing.T) {
	testCfg := Config{
		GaugeMetricCount:           200,
		CounterMetricCount:         200,
		HistogramMetricCount:       10,
		HistogramBuckets:           7,
		NativeHistogramMetricCount: 10,
		SummaryMetricCount:         10,
		SummaryObjectives:          2,
		SeriesOperationMode:        disabledOpMode,

		MinSeriesCount: 0,
		MaxSeriesCount: 1000,
		LabelCount:     1,
		SeriesCount:    10,
		MetricLength:   1,
		LabelLength:    1,
		ConstLabels:    []string{"constLabel=test"},
	}
	assert.NoError(t, testCfg.Validate())

	reg := prometheus.NewRegistry()
	coll := NewCollector(testCfg)
	reg.MustRegister(coll)

	go coll.Run()
	t.Cleanup(func() {
		coll.Stop(nil)
	})

	time.Sleep(2 * time.Second)

	g, c, h, nh, s := countSeriesTypes(t, reg)
	assert.Equal(t, testCfg.GaugeMetricCount*testCfg.SeriesCount, g)
	assert.Equal(t, testCfg.CounterMetricCount*testCfg.SeriesCount, c)
	assert.Equal(t, (2+testCfg.HistogramBuckets+1)*testCfg.HistogramMetricCount*testCfg.SeriesCount, h)
	assert.Equal(t, testCfg.NativeHistogramMetricCount*testCfg.SeriesCount, nh)
	assert.Equal(t, (2+testCfg.SummaryObjectives)*testCfg.SummaryMetricCount*testCfg.SeriesCount, s)
}

func TestRunMetrics_ValueChange_SeriesCountSame(t *testing.T) {
	testCfg := Config{
		GaugeMetricCount:           200,
		CounterMetricCount:         200,
		HistogramMetricCount:       10,
		HistogramBuckets:           7,
		NativeHistogramMetricCount: 10,
		SummaryMetricCount:         10,
		SummaryObjectives:          2,
		SeriesOperationMode:        disabledOpMode,

		MinSeriesCount: 0,
		MaxSeriesCount: 1000,
		LabelCount:     1,
		SeriesCount:    10,
		MetricLength:   1,
		LabelLength:    1,
		ConstLabels:    []string{"constLabel=test"},

		ValueInterval: 1, // Change value every second.
	}
	assert.NoError(t, testCfg.Validate())

	reg := prometheus.NewRegistry()
	coll := NewCollector(testCfg)
	reg.MustRegister(coll)

	go coll.Run()
	t.Cleanup(func() {
		coll.Stop(nil)
	})

	// We can't assert value, or even it's change without mocking random generator,
	// but let's at least assert series count does not change.
	for i := 0; i < 5; i++ {
		time.Sleep(2 * time.Second)

		g, c, h, nh, s := countSeriesTypes(t, reg)
		assert.Equal(t, testCfg.GaugeMetricCount*testCfg.SeriesCount, g)
		assert.Equal(t, testCfg.CounterMetricCount*testCfg.SeriesCount, c)
		assert.Equal(t, (2+testCfg.HistogramBuckets+1)*testCfg.HistogramMetricCount*testCfg.SeriesCount, h)
		assert.Equal(t, testCfg.NativeHistogramMetricCount*testCfg.SeriesCount, nh)
		assert.Equal(t, (2+testCfg.SummaryObjectives)*testCfg.SummaryMetricCount*testCfg.SeriesCount, s)
	}
}

func currentCycleID(t *testing.T, registry *prometheus.Registry) (cycleID int) {
	t.Helper()

	metricsFamilies, err := registry.Gather()
	assert.NoError(t, err)

	cycleID = -1
	for _, mf := range metricsFamilies {
		for _, m := range mf.Metric {
			for _, l := range m.GetLabel() {
				if l.GetName() == "cycle_id" {
					gotCycleID, err := strconv.Atoi(l.GetValue())
					require.NoError(t, err)

					if cycleID == -1 {
						cycleID = gotCycleID
						continue
					}
					if cycleID != gotCycleID {
						t.Fatalf("expected cycle ID to be the same across all metrics, previous metric had cycle_id=%v; now found %v", cycleID, m.GetLabel())
					}
				}
			}
		}
	}
	return cycleID
}

func TestRunMetrics_SeriesChurn(t *testing.T) {
	testCfg := Config{
		GaugeMetricCount:           200,
		CounterMetricCount:         200,
		HistogramMetricCount:       10,
		HistogramBuckets:           7,
		NativeHistogramMetricCount: 10,
		SummaryMetricCount:         10,
		SummaryObjectives:          2,
		SeriesOperationMode:        disabledOpMode,

		MinSeriesCount: 0,
		MaxSeriesCount: 1000,
		LabelCount:     1,
		SeriesCount:    10,
		MetricLength:   1,
		LabelLength:    1,
		ConstLabels:    []string{"constLabel=test"},

		SeriesInterval: 1, // Churn series every second.
		// Change value every second too, there was a regression when both value and series cycle.
		ValueInterval: 1,
	}
	assert.NoError(t, testCfg.Validate())

	reg := prometheus.NewRegistry()
	coll := NewCollector(testCfg)
	reg.MustRegister(coll)

	go coll.Run()
	t.Cleanup(func() {
		coll.Stop(nil)
	})

	cycleID := -1
	// No matter how much time we wait, we should see always same series count, just
	// different cycle_id.
	for i := 0; i < 5; i++ {
		time.Sleep(2 * time.Second)

		g, c, h, nh, s := countSeriesTypes(t, reg)
		assert.Equal(t, testCfg.GaugeMetricCount*testCfg.SeriesCount, g)
		assert.Equal(t, testCfg.CounterMetricCount*testCfg.SeriesCount, c)
		assert.Equal(t, (2+testCfg.HistogramBuckets+1)*testCfg.HistogramMetricCount*testCfg.SeriesCount, h)
		assert.Equal(t, testCfg.NativeHistogramMetricCount*testCfg.SeriesCount, nh)
		assert.Equal(t, (2+testCfg.SummaryObjectives)*testCfg.SummaryMetricCount*testCfg.SeriesCount, s)

		gotCycleID := currentCycleID(t, reg)
		require.Greater(t, gotCycleID, cycleID)
		cycleID = gotCycleID
	}
}

func TestRunMetricsSeriesCountChangeDoubleHalve(t *testing.T) {
	testCfg := Config{
		GaugeMetricCount:     1,
		LabelCount:           1,
		SeriesCount:          5, // Initial.
		MaxSeriesCount:       10,
		MinSeriesCount:       1,
		SpikeMultiplier:      1.5,
		SeriesChangeRate:     1,
		MetricLength:         1,
		LabelLength:          1,
		ValueInterval:        100,
		SeriesInterval:       100,
		MetricInterval:       100,
		SeriesChangeInterval: 3,
		SeriesOperationMode:  doubleHalveOpMode,
		ConstLabels:          []string{"constLabel=test"},
	}
	assert.NoError(t, testCfg.Validate())

	reg := prometheus.NewRegistry()
	coll := NewCollector(testCfg)
	reg.MustRegister(coll)

	go coll.Run()
	t.Cleanup(func() {
		coll.Stop(nil)
	})

	time.Sleep(2 * time.Second)
	for i := 0; i < 4; i++ {
		time.Sleep(time.Duration(testCfg.SeriesChangeInterval) * time.Second)
		if i%2 == 0 { // Expecting halved series count
			currentCount := countSeries(t, reg)
			expectedCount := testCfg.SeriesCount
			assert.Equal(t, expectedCount, currentCount, "Halved series count should be %d but got %d", expectedCount, currentCount)
		} else { // Expecting doubled series count
			currentCount := countSeries(t, reg)
			expectedCount := testCfg.SeriesCount * 2
			assert.Equal(t, expectedCount, currentCount, "Doubled series count should be %d but got %d", expectedCount, currentCount)
		}
	}
}

func TestRunMetricsGradualChange(t *testing.T) {
	testCfg := Config{
		GaugeMetricCount:     1,
		LabelCount:           1,
		SeriesCount:          100, // Initial.
		MaxSeriesCount:       30,
		MinSeriesCount:       10,
		SpikeMultiplier:      1.5,
		SeriesChangeRate:     10,
		MetricLength:         1,
		LabelLength:          1,
		ValueInterval:        100,
		SeriesInterval:       100,
		MetricInterval:       100,
		SeriesChangeInterval: 3,
		SeriesOperationMode:  gradualChangeOpMode,
		ConstLabels:          []string{"constLabel=test"},
	}
	assert.NoError(t, testCfg.Validate())

	reg := prometheus.NewRegistry()
	coll := NewCollector(testCfg)
	reg.MustRegister(coll)

	go coll.Run()
	t.Cleanup(func() {
		coll.Stop(nil)
	})

	time.Sleep(2 * time.Second)
	currentCount := countSeries(t, reg)
	fmt.Println("seriesCount: ", currentCount)
	assert.Equal(t, testCfg.MinSeriesCount, currentCount, "Initial series count should be minSeriesCount %d but got %d", testCfg.MinSeriesCount, currentCount)

	assert.Eventually(t, func() bool {
		graduallyIncreasedCount := countSeries(t, reg)
		fmt.Println("seriesCount: ", graduallyIncreasedCount)
		if graduallyIncreasedCount > testCfg.MaxSeriesCount {
			t.Fatalf("Gradually increased series count should be less than maxSeriesCount %d but got %d", testCfg.MaxSeriesCount, graduallyIncreasedCount)
		}
		if currentCount > graduallyIncreasedCount {
			t.Fatalf("Gradually increased series count should be greater than initial series count %d but got %d", currentCount, graduallyIncreasedCount)
		} else {
			currentCount = graduallyIncreasedCount
		}

		return graduallyIncreasedCount == testCfg.MaxSeriesCount
	}, 15*time.Second, time.Duration(testCfg.SeriesChangeInterval)*time.Second, "Did not receive update notification for series count gradual increase in time")

	assert.Eventually(t, func() bool {
		graduallyIncreasedCount := countSeries(t, reg)
		fmt.Println("seriesCount: ", graduallyIncreasedCount)
		if graduallyIncreasedCount < testCfg.MinSeriesCount {
			t.Fatalf("Gradually increased series count should be less than maxSeriesCount %d but got %d", testCfg.MaxSeriesCount, graduallyIncreasedCount)
		}

		return graduallyIncreasedCount == testCfg.MinSeriesCount
	}, 15*time.Second, time.Duration(testCfg.SeriesChangeInterval)*time.Second, "Did not receive update notification for series count gradual increase in time")
}

func TestRunMetricsWithInvalidSeriesCounts(t *testing.T) {
	testCfg := Config{
		GaugeMetricCount:     1,
		LabelCount:           1,
		SeriesCount:          100,
		MaxSeriesCount:       10,
		MinSeriesCount:       100,
		SpikeMultiplier:      1.5,
		SeriesChangeRate:     10,
		MetricLength:         1,
		LabelLength:          1,
		ValueInterval:        100,
		SeriesInterval:       100,
		MetricInterval:       100,
		SeriesChangeInterval: 3,
		SeriesOperationMode:  gradualChangeOpMode,
		ConstLabels:          []string{"constLabel=test"},
	}
	assert.Error(t, testCfg.Validate())
}

func TestRunMetricsSpikeChange(t *testing.T) {
	testCfg := Config{
		GaugeMetricCount:     1,
		LabelCount:           1,
		SeriesCount:          100,
		MaxSeriesCount:       30,
		MinSeriesCount:       10,
		SpikeMultiplier:      1.5,
		SeriesChangeRate:     10,
		MetricLength:         1,
		LabelLength:          1,
		ValueInterval:        100,
		SeriesInterval:       100,
		MetricInterval:       100,
		SeriesChangeInterval: 10,
		SeriesOperationMode:  spikeOpMode,
		ConstLabels:          []string{"constLabel=test"},
	}
	assert.NoError(t, testCfg.Validate())

	reg := prometheus.NewRegistry()
	coll := NewCollector(testCfg)
	reg.MustRegister(coll)

	go coll.Run()
	t.Cleanup(func() {
		coll.Stop(nil)
	})

	time.Sleep(2 * time.Second)
	for i := 0; i < 4; i++ {
		time.Sleep(time.Duration(testCfg.SeriesChangeInterval) * time.Second)
		if i%2 == 0 {
			currentCount := countSeries(t, reg)
			expectedCount := testCfg.SeriesCount
			assert.Equal(t, expectedCount, currentCount, fmt.Sprintf("Halved series count should be %d but got %d", expectedCount, currentCount))
		} else {
			currentCount := countSeries(t, reg)
			expectedCount := int(float64(testCfg.SeriesCount) * testCfg.SpikeMultiplier)
			assert.Equal(t, expectedCount, currentCount, fmt.Sprintf("Multiplied the series count by %.1f, should be %d but got %d", testCfg.SpikeMultiplier, expectedCount, currentCount))
		}
	}
}

func TestCollectorLabels(t *testing.T) {
	testCfg := Config{
		GaugeMetricCount:     1,
		LabelCount:           2,
		SeriesCount:          100,
		MaxSeriesCount:       30,
		MinSeriesCount:       10,
		SpikeMultiplier:      1.5,
		MetricLength:         1,
		LabelLength:          1,
		SeriesOperationMode:  spikeOpMode,
		ConstLabels:          []string{"constLabel=test"},
	}

	assert.NoError(t, testCfg.Validate())

	reg := prometheus.NewRegistry()
	col := NewCollector(testCfg)
	reg.MustRegister(col)

	go col.Run()
	t.Cleanup(func() {
		col.Stop(nil)
	})
	
	time.Sleep((2 * time.Second))
	metricsFamilies, err := reg.Gather()
	assert.NotEmpty(t, metricsFamilies)
	assert.NoError(t,err)

	for _, mf := range metricsFamilies {
		for _, m := range mf.Metric {
			labels := m.GetLabel()
			labelMap := make(map[string]string)
			for _, l := range labels {
				labelMap[l.GetName()] = l.GetValue()
			}
			assert.Equal(t, "test", labelMap["constLabel"])
			assert.Contains(t, labelMap, "label_key_k_0")
			assert.Contains(t, labelMap, "label_key_k_1")
			assert.Contains(t, labelMap, "series_id")
			assert.Contains(t, labelMap, "cycle_id")
		}
	}
}