# avalanche

Avalanche serves a text-based [Prometheus metrics](https://prometheus.io/docs/instrumenting/exposition_formats/) endpoint for load testing [Prometheus](https://prometheus.io/) and possibly other [OpenMetrics](https://github.com/OpenObservability/OpenMetrics) consumers.

Avalanche also supports load testing for services accepting data via Prometheus remote_write API such as [Thanos](https://github.com/improbable-eng/thanos), [Cortex](https://github.com/weaveworks/cortex), [M3DB](https://m3db.github.io/m3/integrations/prometheus/), [VictoriaMetrics](https://github.com/VictoriaMetrics/VictoriaMetrics/) and other services [listed here](https://prometheus.io/docs/operating/integrations/#remote-endpoints-and-storage).

Metric names and unique series change over time to simulate series churn.

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
