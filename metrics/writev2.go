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
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	writev2 "github.com/prometheus/client_golang/exp/api/remote/genproto/v2"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/model"

	"github.com/prometheus-community/avalanche/pkg/errors"
)

func (c *Client) writeV2(ctx context.Context) error {
	select {
	// Wait for update first as write and collector.Run runs simultaneously.
	case <-c.config.UpdateNotify:
	case <-ctx.Done():
		return ctx.Err()
	}

	tss, st, err := collectMetricsV2(c.gatherer, c.config.OutOfOrder)
	if err != nil {
		return err
	}

	var (
		totalTime       time.Duration
		totalSamplesExp = len(tss) * c.config.RequestCount
		totalSamplesAct int
		mtx             sync.Mutex
		wgMetrics       sync.WaitGroup
		merr            = &errors.MultiError{}
	)

	shouldRunForever := c.config.RequestCount == -1
	if shouldRunForever {
		log.Printf("Sending: %v timeseries infinitely, %v timeseries per request, %v delay between requests\n",
			len(tss), c.config.BatchSize, c.config.RequestInterval)
	} else {
		log.Printf("Sending: %v timeseries, %v times, %v timeseries per request, %v delay between requests\n",
			len(tss), c.config.RequestCount, c.config.BatchSize, c.config.RequestInterval)
	}

	ticker := time.NewTicker(c.config.RequestInterval)
	defer ticker.Stop()

	concurrencyLimitCh := make(chan struct{}, c.config.Concurrency)

	for i := 0; ; {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if !shouldRunForever {
			if i >= c.config.RequestCount {
				break
			}
			i++
		}

		<-ticker.C
		select {
		case <-c.config.UpdateNotify:
			log.Println("updating remote write metrics")
			tss, st, err = collectMetricsV2(c.gatherer, c.config.OutOfOrder)
			if err != nil {
				merr.Add(err)
			}
		default:
			tss = updateTimestampsV2(tss)
		}

		start := time.Now()
		for i := 0; i < len(tss); i += c.config.BatchSize {
			wgMetrics.Add(1)
			concurrencyLimitCh <- struct{}{}
			go func(i int) {
				defer func() {
					<-concurrencyLimitCh
				}()
				defer wgMetrics.Done()
				end := i + c.config.BatchSize
				if end > len(tss) {
					end = len(tss)
				}
				req := &writev2.Request{
					Timeseries: tss[i:end],
					Symbols:    st.Symbols(), // We pass full symbols table to each request for now
				}

				if _, err := c.remoteAPI.Write(ctx, req); err != nil {
					merr.Add(err)
					c.logger.Error("error writing metrics", "error", err)
					return
				}

				mtx.Lock()
				totalSamplesAct += len(tss[i:end])
				mtx.Unlock()
			}(i)
		}
		wgMetrics.Wait()
		totalTime += time.Since(start)
		if merr.Count() > 20 {
			merr.Add(fmt.Errorf("too many errors"))
			return merr.Err()
		}
	}
	if c.config.RequestCount*len(tss) != totalSamplesAct {
		merr.Add(fmt.Errorf("total samples mismatch, exp:%v , act:%v", totalSamplesExp, totalSamplesAct))
	}
	c.logger.Info("metrics summary",
		"total_time", totalTime.Round(time.Second),
		"total_samples", totalSamplesAct,
		"samples_per_sec", int(float64(totalSamplesAct)/totalTime.Seconds()),
		"errors", merr.Count())
	return merr.Err()
}

func updateTimestampsV2(tss []*writev2.TimeSeries) []*writev2.TimeSeries {
	now := time.Now().UnixMilli()
	for i := range tss {
		tss[i].Samples[0].Timestamp = now
	}
	return tss
}

func shuffleTimestampsV2(tss []*writev2.TimeSeries) []*writev2.TimeSeries {
	now := time.Now().UnixMilli()
	offsets := []int64{0, -60 * 1000, -5 * 60 * 1000}
	for i := range tss {
		offset := offsets[i%len(offsets)]
		tss[i].Samples[0].Timestamp = now + offset
	}
	return tss
}

func collectMetricsV2(gatherer prometheus.Gatherer, outOfOrder bool) ([]*writev2.TimeSeries, writev2.SymbolsTable, error) {
	metricFamilies, err := gatherer.Gather()
	if err != nil {
		return nil, writev2.SymbolsTable{}, err
	}
	tss, st := ToTimeSeriesSliceV2(metricFamilies)
	if outOfOrder {
		tss = shuffleTimestampsV2(tss)
	}
	return tss, st, nil
}

// ToTimeSeriesSliceV2 converts a slice of metricFamilies containing samples into a slice of writev2.TimeSeries.
func ToTimeSeriesSliceV2(metricFamilies []*dto.MetricFamily) ([]*writev2.TimeSeries, writev2.SymbolsTable) {
	st := writev2.NewSymbolTable()
	timestamp := int64(model.Now())
	tss := make([]*writev2.TimeSeries, 0, len(metricFamilies)*10)

	skippedSamples := 0
	for _, metricFamily := range metricFamilies {
		for _, metric := range metricFamily.Metric {
			labels := prompbLabels(*metricFamily.Name, metric.Label)
			labelRefs := make([]uint32, 0, len(labels))
			for _, label := range labels {
				labelRefs = append(labelRefs, st.Symbolize(label.Name))
				labelRefs = append(labelRefs, st.Symbolize(label.Value))
			}
			ts := &writev2.TimeSeries{
				LabelsRefs: labelRefs,
			}
			switch *metricFamily.Type {
			case dto.MetricType_COUNTER:
				ts.Samples = []*writev2.Sample{{
					Value:     *metric.Counter.Value,
					Timestamp: timestamp,
				}}
				tss = append(tss, ts)
			case dto.MetricType_GAUGE:
				ts.Samples = []*writev2.Sample{{
					Value:     *metric.Gauge.Value,
					Timestamp: timestamp,
				}}
				tss = append(tss, ts)
			default:
				skippedSamples++
			}
		}
	}
	if skippedSamples > 0 {
		log.Printf("WARN: Skipping %v samples; sending only %v samples, given only gauge and counters are currently implemented\n", skippedSamples, len(tss))
	}
	return tss, st
}
