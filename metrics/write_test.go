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

	"github.com/prometheus/prometheus/prompb"
)

func TestShuffleTimestamps(t *testing.T) {
	now := time.Now().UnixMilli()

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

	shuffledTSS := shuffleTimestamps(tss)

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

	offsets := []int64{0, -60 * 1000, -5 * 60 * 1000}
	for _, ts := range shuffledTSS {
		timestampValid := false
		for _, offset := range offsets {
			expectedTimestamp := now + offset
			if ts.Samples[0].Timestamp == expectedTimestamp {
				timestampValid = true
				break
			}
		}
		if !timestampValid {
			t.Errorf("Timestamp %v is not in the expected offsets: %v", ts.Samples[0].Timestamp, offsets)
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
