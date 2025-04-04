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

// Package main implements mtypes CLI, see README for details.
// Initially hosted and created by @bwplotka in https://github.com/bwplotka/prombenchy/pull/12.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"strings"
	"text/tabwriter"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

type stats struct {
	families, series, buckets, objectives int

	// adjustedSeries represents series that would result in "series" in Prometheus data model
	// (includes _bucket, _count, _sum, _quantile).
	adjustedSeries int
}

var metricType_NATIVE_HISTOGRAM dto.MetricType = 999 //nolint:revive

func main() {
	resource := flag.String("resource", "", "Path or URL to the resource (file, <url>/metrics) containing Prometheus metric format.")
	avalancheFlagsForTotal := flag.Int("avalanche-flags-for-adjusted-series", 0, "If more than zero, it additionally prints flags for the avalanche 0.6.0 command line to generate metrics for the similar type distribution; to get the total number of adjusted series to the given value.")
	flag.Parse()

	var input io.Reader = os.Stdin
	if *resource != "" {
		switch {
		case strings.HasPrefix(*resource, "https://"), strings.HasPrefix(*resource, "http://"):
			if _, err := url.Parse(*resource); err != nil {
				log.Fatalf("error parsing HTTP URL to the resource %v; got %v", *resource, err)
			}
			resp, err := http.Get(*resource)
			if err != nil {
				log.Fatalf("http get against %v failed", err)
			}
			defer resp.Body.Close()
			input = resp.Body
		default:
			// Open the input file.
			file, err := os.Open(*resource)
			if err != nil {
				log.Fatalf("Error opening file: %v", err) //nolint:gocritic
			}
			defer file.Close()
			input = file
		}
	}
	statistics, err := calculateTargetStatistics(input)
	if err != nil {
		log.Fatal(err)
	}
	total := computeTotal(statistics)
	writeStatistics(os.Stdout, total, statistics)
	if *avalancheFlagsForTotal > 0 {
		fmt.Fprintln(os.Stdout)
		fmt.Fprintln(os.Stdout, "Avalanche flags for the similar distribution to get to the adjusted series goal of:", *avalancheFlagsForTotal)
		seriesCount := 10
		flags, adjustedSum := computeAvalancheFlags(*avalancheFlagsForTotal, seriesCount, total, statistics)
		for _, f := range flags {
			fmt.Fprintln(os.Stdout, f)
		}
		fmt.Fprintln(os.Stdout, "This should give the total adjusted series to:", adjustedSum*10)
	}
}

func computeTotal(statistics map[dto.MetricType]stats) stats {
	var total stats
	for _, s := range statistics {
		total.families += s.families
		total.series += s.series
		total.adjustedSeries += s.adjustedSeries
	}
	return total
}

func computeAvalancheFlags(avalancheFlagsForTotal, seriesCount int, total stats, statistics map[dto.MetricType]stats) ([]string, int) {
	// adjustedGoal is tracking the # of adjusted series we want to generate with avalanche.
	adjustedGoal := float64(avalancheFlagsForTotal)
	adjustedGoal /= float64(seriesCount)
	// adjustedSum is tracking the total sum of series so far (at the end hopefully adjustedSum ~= adjustedGoal)
	adjustedSum := 0
	// Accumulate flags
	var flags []string
	for _, mtype := range allTypes {
		s := statistics[mtype]

		// adjustedSeriesRatio is tracking the ratio of this type in the input file.
		// We try to get similar ratio, but with different absolute counts, given the total sum of series we are aiming for.
		adjustedSeriesRatio := float64(s.adjustedSeries) / float64(total.adjustedSeries)

		// adjustedSeriesForType is tracking (per metric type), how many unique series of that
		// metric type avalanche needs to create according to the ratio we got from our input.
		var adjustedSeriesForType int
		if !math.IsNaN(adjustedSeriesRatio) {
			adjustedSeriesForType = int(adjustedGoal * adjustedSeriesRatio)
		}

		switch mtype {
		case dto.MetricType_GAUGE:
			flags = append(flags, fmt.Sprintf("--gauge-metric-count=%v", adjustedSeriesForType))
			adjustedSum += adjustedSeriesForType
		case dto.MetricType_COUNTER:
			flags = append(flags, fmt.Sprintf("--counter-metric-count=%v", adjustedSeriesForType))
			adjustedSum += adjustedSeriesForType
		case dto.MetricType_HISTOGRAM:
			var avgBkts int
			if s.series > 0 {
				avgBkts = s.buckets / s.series
				adjustedSeriesForType /= 2 + avgBkts
			}
			flags = append(flags, fmt.Sprintf("--histogram-metric-count=%v", adjustedSeriesForType))
			if s.series > 0 {
				flags = append(flags, fmt.Sprintf("--histogram-metric-bucket-count=%v", avgBkts-1)) // -1 is due to caveat of additional +Inf not added by avalanche.
			}
			adjustedSum += adjustedSeriesForType * (2 + avgBkts)
		case metricType_NATIVE_HISTOGRAM:
			flags = append(flags, fmt.Sprintf("--native-histogram-metric-count=%v", adjustedSeriesForType))
			adjustedSum += adjustedSeriesForType
		case dto.MetricType_SUMMARY:
			var avgObjs int
			if s.series > 0 {
				avgObjs = s.objectives / s.series
				adjustedSeriesForType /= 2 + avgObjs
			}
			flags = append(flags, fmt.Sprintf("--summary-metric-count=%v", adjustedSeriesForType))
			if s.series > 0 {
				flags = append(flags, fmt.Sprintf("--summary-metric-objective-count=%v", avgObjs))
			}
			adjustedSum += adjustedSeriesForType * (2 + avgObjs)
		default:
			if s.series > 0 {
				log.Fatalf("not supported %v metric in avalanche", mtype)
			}
		}
	}
	flags = append(flags, fmt.Sprintf("--series-count=%v", seriesCount))
	// Changes values every 5m.
	flags = append(flags, "--value-interval=300")
	// 1h series churn.
	flags = append(flags, "--series-interval=3600")
	flags = append(flags, "--metric-interval=0")

	return flags, adjustedSum
}

var allTypes = []dto.MetricType{dto.MetricType_GAUGE, dto.MetricType_COUNTER, dto.MetricType_HISTOGRAM, metricType_NATIVE_HISTOGRAM, dto.MetricType_GAUGE_HISTOGRAM, dto.MetricType_SUMMARY, dto.MetricType_UNTYPED}

func writeStatistics(writer io.Writer, total stats, statistics map[dto.MetricType]stats) {
	w := tabwriter.NewWriter(writer, 0, 0, 4, ' ', 0)
	fmt.Fprintln(w, "Metric Type\tMetric Families\tSeries (adjusted)\tSeries (adjusted) %\tAverage Buckets/Objectives")

	for _, mtype := range allTypes {
		s, ok := statistics[mtype]
		if !ok {
			continue
		}

		mtypeStr := mtype.String()
		if mtype == metricType_NATIVE_HISTOGRAM {
			mtypeStr = "HISTOGRAM (native)"
		}

		seriesRatio := 100 * float64(s.series) / float64(total.series)
		adjustedSeriesRatio := 100 * float64(s.adjustedSeries) / float64(total.adjustedSeries)
		switch {
		case s.buckets > 0:
			fmt.Fprintf(w, "%s\t%d\t%d (%d)\t%f (%f)\t%f\n", mtypeStr, s.families, s.series, s.adjustedSeries, seriesRatio, adjustedSeriesRatio, float64(s.buckets)/float64(s.series))
		case s.objectives > 0:
			fmt.Fprintf(w, "%s\t%d\t%d (%d)\t%f (%f)\t%f\n", mtypeStr, s.families, s.series, s.adjustedSeries, seriesRatio, adjustedSeriesRatio, float64(s.objectives)/float64(s.series))
		default:
			fmt.Fprintf(w, "%s\t%d\t%d (%d)\t%f (%f)\t-\n", mtypeStr, s.families, s.series, s.adjustedSeries, seriesRatio, adjustedSeriesRatio)
		}
	}
	fmt.Fprintf(w, "---\t---\t---\t---\t---\n")
	fmt.Fprintf(w, "*\t%d\t%d (%d)\t%f (%f)\t-\n", total.families, total.series, total.adjustedSeries, 100.0, 100.0)
	_ = w.Flush()
}

func calculateTargetStatistics(r io.Reader) (statistics map[dto.MetricType]stats, _ error) {
	// Parse the Prometheus Text format.
	parser := expfmt.NewDecoder(r, expfmt.NewFormat(expfmt.TypeProtoText))

	statistics = map[dto.MetricType]stats{}
	nativeS := statistics[metricType_NATIVE_HISTOGRAM]
	for {
		var mf dto.MetricFamily
		if err := parser.Decode(&mf); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("parsing %w", err)
		}

		s := statistics[mf.GetType()]

		var mfAccounted, mfAccountedNative bool
		switch mf.GetType() {
		case dto.MetricType_GAUGE_HISTOGRAM, dto.MetricType_HISTOGRAM:
			for _, m := range mf.GetMetric() {
				if m.GetHistogram().GetSchema() == 0 {
					// classic one.
					s.series++
					s.buckets += len(m.GetHistogram().GetBucket())
					s.adjustedSeries += 2 + len(m.GetHistogram().GetBucket())

					if !mfAccounted {
						s.families++
						mfAccounted = true
					}
				} else {
					// native one.
					nativeS.series++
					nativeS.buckets += len(m.GetHistogram().GetNegativeDelta())
					nativeS.buckets += len(m.GetHistogram().GetNegativeCount())
					nativeS.buckets += len(m.GetHistogram().GetPositiveDelta())
					nativeS.buckets += len(m.GetHistogram().GetPositiveCount())
					nativeS.adjustedSeries++

					if !mfAccountedNative {
						nativeS.families++
						mfAccountedNative = true
					}
				}
			}
		case dto.MetricType_SUMMARY:
			s.series += len(mf.GetMetric())
			s.families++
			for _, m := range mf.GetMetric() {
				s.objectives += len(m.GetSummary().GetQuantile())
				s.adjustedSeries += 2 + len(m.GetSummary().GetQuantile())
			}
		default:
			s.series += len(mf.GetMetric())
			s.families++
			s.adjustedSeries += len(mf.GetMetric())
		}
		statistics[mf.GetType()] = s
	}
	if nativeS.series > 0 {
		statistics[metricType_NATIVE_HISTOGRAM] = nativeS
	}
	return statistics, nil
}
