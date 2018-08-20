# avalanche

Avalanche serves a text-based [Prometheus metrics](https://prometheus.io/docs/instrumenting/exposition_formats/) endpoint for load testing [Prometheus](https://prometheus.io/), [Cortex](https://github.com/weaveworks/cortex), [Thanos](https://github.com/improbable-eng/thanos), and possibly other [OpenMetrics](https://github.com/OpenObservability/OpenMetrics) consumers. Metric names and unique series change over time to simulate series churn.

Avalanche has command line flags to configure how many metrics and series are exposed and how often the series change.

```bash
usage: avalanche [<flags>]

avalanche - metrics test server

Flags:
  --help                 Show context-sensitive help (also try --help-long and --help-man).
  --metric-count=500     Number of metrics to serve.
  --label-count=10       Number of labels per-metric.
  --series-count=10      Number of series per-metric.
  --metricname-length=5  Modify length of metric names.
  --labelname-length=5   Modify length of label names.
  --value-interval=30    Change series values every {interval} seconds.
  --series-interval=60   Change series_id label values every {interval} seconds.
  --metric-interval=120  Change __name__ label values every {interval} seconds.
  --port=9001            Port to serve at
  --version              Show application version.
```

## build and run
```
go build -o=./avalanche ./cmd
./avalanche --help
```