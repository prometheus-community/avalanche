# avalanche

Avalanche serves a text-based [Prometheus metrics](https://prometheus.io/docs/instrumenting/exposition_formats/) endpoint for load testing [Prometheus](https://prometheus.io/), [Cortex](https://github.com/weaveworks/cortex), [Thanos](https://github.com/improbable-eng/thanos), and possibly other [OpenMetrics](https://github.com/OpenObservability/OpenMetrics) consumers. Metric names and unique series change over time to simulate series churn.

Checkout the [blog post](https://blog.freshtracks.io/load-testing-prometheus-metric-ingestion-5b878711711c).

## configuration flags 
```bash 
avalanche --help
```

## run Docker image

```bash
docker run quay.io/freshtracks.io/avalanche --help
```

## build and run go binary
```bash
go get github.com/open-fresh/avalanche/cmd/...
avalanche --help
```
