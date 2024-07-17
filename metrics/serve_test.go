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
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

// Helper function to count the series in the registry
func countSeries(t *testing.T, registry *prometheus.Registry) int {
	metricsFamilies, err := registry.Gather()
	assert.NoError(t, err)

	seriesCount := 0
	for _, mf := range metricsFamilies {
		for range mf.Metric {
			seriesCount++
		}
	}

	return seriesCount
}

func TestRunMetricsSeriesCountChangeDoubleHalve(t *testing.T) {
	const (
		initialSeriesCount   = 5
		metricCount          = 1
		labelCount           = 1
		maxSeriesCount       = 10
		minSeriesCount       = 1
		seriesChangeRate     = 1
		metricLength         = 1
		labelLength          = 1
		valueInterval        = 100
		seriesInterval       = 100
		metricInterval       = 100
		seriesChangeInterval = 3
		operationMode        = "double-halve"
		constLabel           = "constLabel=test"
	)

	stop := make(chan struct{})
	defer close(stop)

	promRegistry = prometheus.NewRegistry()

	_, err := RunMetrics(metricCount, labelCount, initialSeriesCount, seriesChangeRate, maxSeriesCount, minSeriesCount, metricLength, labelLength, valueInterval, seriesInterval, metricInterval, seriesChangeInterval, operationMode, []string{constLabel}, stop)
	assert.NoError(t, err)
	time.Sleep(2 * time.Second)
	for i := 0; i < 4; i++ {
		time.Sleep(time.Duration(seriesChangeInterval) * time.Second)
		if i%2 == 0 { // Expecting halved series count
			currentCount := countSeries(t, promRegistry)
			expectedCount := initialSeriesCount
			assert.Equal(t, expectedCount, currentCount, "Halved series count should be %d but got %d", expectedCount, currentCount)
		} else { // Expecting doubled series count
			currentCount := countSeries(t, promRegistry)
			expectedCount := initialSeriesCount * 2
			assert.Equal(t, expectedCount, currentCount, "Doubled series count should be %d but got %d", expectedCount, currentCount)
		}
	}
}
func TestRunMetricsGradualChange(t *testing.T) {
	const (
		metricCount          = 1
		labelCount           = 1
		seriesCount          = 100
		maxSeriesCount       = 30
		minSeriesCount       = 10
		seriesChangeRate     = 10
		metricLength         = 1
		labelLength          = 1
		valueInterval        = 100
		seriesInterval       = 100
		metricInterval       = 100
		seriesChangeInterval = 3
		operationMode        = "gradual-change"
		constLabel           = "constLabel=test"
	)

	stop := make(chan struct{})
	defer close(stop)

	promRegistry = prometheus.NewRegistry()

	_, err := RunMetrics(metricCount, labelCount, seriesCount, seriesChangeRate, maxSeriesCount, minSeriesCount, metricLength, labelLength, valueInterval, seriesInterval, metricInterval, seriesChangeInterval, operationMode, []string{constLabel}, stop)
	assert.NoError(t, err)

	time.Sleep(2 * time.Second)
	currentCount := countSeries(t, promRegistry)
	expectedInitialCount := currentCount
	assert.Equal(t, expectedInitialCount, currentCount, "Initial series count should be minSeriesCount %d but got %d", expectedInitialCount, currentCount)

	assert.Eventually(t, func() bool {
		graduallyIncreasedCount := countSeries(t, promRegistry)
		fmt.Println("seriesCount: ", graduallyIncreasedCount)
		if graduallyIncreasedCount > maxSeriesCount {
			t.Fatalf("Gradually increased series count should be less than maxSeriesCount %d but got %d", maxSeriesCount, graduallyIncreasedCount)
		}
		if currentCount > graduallyIncreasedCount {
			t.Fatalf("Gradually increased series count should be greater than initial series count %d but got %d", currentCount, graduallyIncreasedCount)
		} else {
			currentCount = graduallyIncreasedCount
		}

		return graduallyIncreasedCount == maxSeriesCount
	}, 15*time.Second, seriesChangeInterval*time.Second, "Did not receive update notification for series count gradual increase in time")

	assert.Eventually(t, func() bool {
		graduallyIncreasedCount := countSeries(t, promRegistry)
		fmt.Println("seriesCount: ", graduallyIncreasedCount)
		if graduallyIncreasedCount < minSeriesCount {
			t.Fatalf("Gradually increased series count should be less than maxSeriesCount %d but got %d", maxSeriesCount, graduallyIncreasedCount)
		}

		return graduallyIncreasedCount == minSeriesCount
	}, 15*time.Second, seriesChangeInterval*time.Second, "Did not receive update notification for series count gradual increase in time")
}

// if min is bigger than maxSeriesCount, fail in GradualChange
func TestRunMetricsWithInvalidSeriesCounts(t *testing.T) {
	const (
		metricCount          = 1
		labelCount           = 1
		seriesCount          = 100
		maxSeriesCount       = 10
		minSeriesCount       = 100
		seriesChangeRate     = 10
		metricLength         = 1
		labelLength          = 1
		valueInterval        = 100
		seriesInterval       = 100
		metricInterval       = 100
		seriesChangeInterval = 3
		operationMode        = "gradual-change"
		constLabel           = "constLabel=test"
	)

	stop := make(chan struct{})
	defer close(stop)

	promRegistry = prometheus.NewRegistry()

	_, err := RunMetrics(metricCount, labelCount, seriesCount, seriesChangeRate, maxSeriesCount, minSeriesCount, metricLength, labelLength, valueInterval, seriesInterval, metricInterval, seriesChangeInterval, operationMode, []string{constLabel}, stop)
	assert.Error(t, err)
}
