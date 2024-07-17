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
		metricCount           = 1
		labelCount            = 1
		seriesCount           = 100
		maxSeriesCount        = 30
		minSeriesCount        = 10
		seriesChangeRate      = 10
		metricLength          = 1
		labelLength           = 1
		valueInterval         = 100
		seriesInterval        = 100
		metricInterval        = 100
		seriesChangeInterval  = 3
		operationMode         = "gradual-change"
		constLabel            = "constLabel=test"
		updateNotifyTimeout   = 5 * time.Second
		waitTimeBetweenChecks = 40 * time.Second
	)

	stop := make(chan struct{})
	defer close(stop)

	promRegistry = prometheus.NewRegistry()

	_, err := RunMetrics(metricCount, labelCount, seriesCount, seriesChangeRate, maxSeriesCount, minSeriesCount, metricLength, labelLength, valueInterval, seriesInterval, metricInterval, seriesChangeInterval, operationMode, []string{constLabel}, stop)
	assert.NoError(t, err)

	time.Sleep(time.Duration(seriesChangeInterval) * time.Second)
	initialCount := countSeries(t, promRegistry)
	expectedInitialCount := minSeriesCount
	assert.Equal(t, expectedInitialCount, initialCount, "Initial series count should be minSeriesCount %d but got %d", expectedInitialCount, initialCount)

	assert.Eventually(t, func() bool {
		graduallyIncreasedCount := countSeries(t, promRegistry)
		if graduallyIncreasedCount > maxSeriesCount {
			t.Fatalf("Gradually increased series count should be less than maxSeriesCount %d but got %d", maxSeriesCount, graduallyIncreasedCount)
		}
		result := assert.Equal(t, maxSeriesCount, graduallyIncreasedCount, "Gradually increased series count should be max %d but got %d", maxSeriesCount, graduallyIncreasedCount)
		return result
	}, waitTimeBetweenChecks, seriesChangeInterval*time.Second, "Did not receive update notification for series count gradual increase in time")

}
