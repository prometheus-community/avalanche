package metrics

import (
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
		initialSeriesCount    = 5
		metricCount           = 1
		labelCount            = 1
		seriesChangeRate      = 1
		metricLength          = 1
		labelLength           = 1
		valueInterval         = 1
		seriesInterval        = 1
		metricInterval        = 1
		seriesChangeInterval  = 3
		operationMode         = "double-halve"
		constLabel            = "constLabel=test"
		updateNotifyTimeout   = 3 * time.Second
		waitTimeBetweenChecks = 3 * time.Second
	)

	stop := make(chan struct{})
	defer close(stop)

	promRegistry = prometheus.NewRegistry()

	updateNotify, err := RunMetrics(metricCount, labelCount, initialSeriesCount, seriesChangeRate, metricLength, labelLength, valueInterval, seriesInterval, metricInterval, seriesChangeInterval, operationMode, []string{constLabel}, stop)
	assert.NoError(t, err)

	initialCount := countSeries(t, promRegistry)
	expectedInitialCount := initialSeriesCount
	assert.Equal(t, expectedInitialCount, initialCount, "Initial series count should be %d but got %d", expectedInitialCount, initialCount)

	// Test for doubling the series count
	select {
	case <-updateNotify:
		time.Sleep(waitTimeBetweenChecks)
		doubledCount := countSeries(t, promRegistry)
		expectedDoubledCount := initialSeriesCount * 2
		assert.Equal(t, expectedDoubledCount, doubledCount, "Doubled series count should be %d but got %d", expectedDoubledCount, doubledCount)
	case <-time.After(updateNotifyTimeout):
		t.Fatal("Did not receive update notification for series count doubling in time")
	}

	// Test for halving the series count
	select {
	case <-updateNotify:
		time.Sleep(waitTimeBetweenChecks)
		halvedCount := countSeries(t, promRegistry)
		expectedHalvedCount := initialSeriesCount
		assert.Equal(t, expectedHalvedCount, halvedCount, "Halved series count should be %d but got %d", expectedHalvedCount, halvedCount)
	case <-time.After(updateNotifyTimeout):
		t.Fatal("Did not receive update notification for series count halving in time")
	}
}
