# mtypes

Go CLI gathering statistics around the distribution of types, average number of buckets (and more) across your Prometheus metrics/series.

## Usage

The main usage allows to take resource (from stdin, file or HTTP /metrics endpoint) and calculate type statistics e.g.:

```bash
go install github.com/prometheus-community/avalanche/cmd/mtypes@latest # or locally: alias mtypes="go run ./cmd/mtypes"
$ mtypes -resource=http://localhost:9090/metrics
$ mtypes -resource=./cmd/mtypes/exampleprometheustarget.txt
$ cat ./cmd/mtypes/exampleprometheustarget.txt | mtypes
```

```bash 
Metric Type    Metric Families    Series (adjusted)    Series (adjusted) %        Average Buckets/Objectives
GAUGE          76                 93 (93)              31.958763 (16.909091)      -
COUNTER        96                 157 (157)            53.951890 (28.545455)      -
HISTOGRAM      8                  14 (186)             4.810997 (33.818182)       11.285714
SUMMARY        15                 27 (114)             9.278351 (20.727273)       2.222222
---            ---                ---                  ---                        ---
*              195                291 (550)            100.000000 (100.000000)    -
```

> NOTE: "Adjusted" series, means actual number of individual series stored in Prometheus. Classic histograms and summaries are stored as a set of counters. This is relevant as the cost of indexing new series is higher than storing complex values (this is why we slowly move to native histograms).

Additionally, you can pass `--avalanche-flags-for-adjusted-series=10000` to print Avalanche v0.6.0+ flags to configure, for avalanche to generate metric target with the given amount of adjusted series, while maintaining a similar distribution e.g.

```bash
cat ./cmd/mtypes/exampleprometheustarget.txt | mtypes --avalanche-flags-for-adjusted-series=1000 
Metric Type    Metric Families    Series (adjusted)    Series (adjusted) %        Average Buckets/Objectives
GAUGE          76                 93 (93)              31.958763 (16.909091)      -
COUNTER        96                 157 (157)            53.951890 (28.545455)      -
HISTOGRAM      8                  14 (186)             4.810997 (33.818182)       11.285714
SUMMARY        15                 27 (114)             9.278351 (20.727273)       2.222222
---            ---                ---                  ---                        ---
*              195                291 (550)            100.000000 (100.000000)    -

Avalanche flags for the similar distribution to get to the adjusted series goal of: 1000
--gauge-metric-count=16
--counter-metric-count=28
--histogram-metric-count=2
--histogram-metric-bucket-count=10
--native-histogram-metric-count=0
--summary-metric-count=5
--summary-metric-objective-count=2
--series-count=10
--value-interval=300 # Changes values every 5m.
--series-interval=3600 # 1h series churn.
--metric-interval=0
This should give the total adjusted series to: 900
```
