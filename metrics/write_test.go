package metrics

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/prometheus/prometheus/prompb"
)

func TestShuffleTimestamps(t *testing.T) {
	rand.Seed(1)

	now := time.Now().UnixMilli()
	window := 5 * time.Minute
	windowMillis := window.Milliseconds()

	tss := []prompb.TimeSeries{
		{Samples: []prompb.Sample{{Timestamp: now}}},
		{Samples: []prompb.Sample{{Timestamp: now}}},
		{Samples: []prompb.Sample{{Timestamp: now}}},
	}

	originalTimestamps := make([]int64, len(tss))
	fmt.Println("Original Timestamps:")
	for i, ts := range tss {
		originalTimestamps[i] = ts.Samples[0].Timestamp
		fmt.Println(time.UnixMilli(ts.Samples[0].Timestamp))
	}

	shuffledTSS := shuffleTimestamps(tss, window)

	fmt.Println("Shuffled Timestamps:")
	for _, ts := range shuffledTSS {
		fmt.Println(time.UnixMilli(ts.Samples[0].Timestamp))
	}

	fmt.Println("Time Differences:")
	for i, ts := range shuffledTSS {
		originalTime := time.UnixMilli(originalTimestamps[i])
		shuffledTime := time.UnixMilli(ts.Samples[0].Timestamp)
		diff := originalTime.Sub(shuffledTime)
		fmt.Printf("Original: %v, Shuffled: %v, Difference: %v\n", originalTime, shuffledTime, diff)
	}

	for _, ts := range shuffledTSS {
		if ts.Samples[0].Timestamp < now-windowMillis || ts.Samples[0].Timestamp > now+windowMillis {
			t.Errorf("Timestamp out of range: got %v, want between %v and %v", ts.Samples[0].Timestamp, now-windowMillis/2, now+windowMillis/2)
		}
	}

	outOfOrder := false
	for i := 1; i < len(shuffledTSS); i++ {
		if shuffledTSS[i].Samples[0].Timestamp < shuffledTSS[i-1].Samples[0].Timestamp {
			outOfOrder = true
			break
		}
	}

	if !outOfOrder {
		t.Error("Timestamps are not out of order")
	}
}
