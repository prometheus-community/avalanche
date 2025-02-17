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
	"testing"
	"time"

	"github.com/prometheus/prometheus/prompb"
)

func TestShuffleTimestamps(t *testing.T) {
	now := time.Now().UnixMilli()
	minTime := 5 * time.Minute
	tss := []prompb.TimeSeries{
		{Samples: []prompb.Sample{{Timestamp: now}}},
		{Samples: []prompb.Sample{{Timestamp: now}}},
		{Samples: []prompb.Sample{{Timestamp: now}}},
	}

	shuffledTSS := shuffleTimestamps(minTime, tss)
	interval := minTime.Milliseconds() / int64(len(tss))

	expectedTimestamps := []int64{
		now,
		now - interval,
		now - 2*interval,
	}

	for i, ts := range shuffledTSS {
		if ts.Samples[0].Timestamp != expectedTimestamps[i] {
			t.Errorf("Expected timestamp %d, but got %d", expectedTimestamps[i], ts.Samples[0].Timestamp)
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
