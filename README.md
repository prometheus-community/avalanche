# avalanche

Avalanche is a load-testing binary capable of generating metrics that can be either:

* scraped via [Prometheus scrape formats](https://prometheus.io/docs/instrumenting/exposition_formats/) (including [OpenMetrics](https://github.com/OpenObservability/OpenMetrics)) endpoint.
* written via Prometheus Remote Write (v1 only for now) to a target endpoint.

This allows load testing services that can scrape (e.g. Prometheus, OpenTelemetry Collector and so), as well as, services accepting data via Prometheus remote_write API such as [Thanos](https://github.com/thanos-io/thanos), [Cortex](https://github.com/cortexproject/cortex), [M3DB](https://m3db.github.io/m3/integrations/prometheus/), [VictoriaMetrics](https://github.com/VictoriaMetrics/VictoriaMetrics/) and other services [listed here](https://prometheus.io/docs/operating/integrations/#remote-endpoints-and-storage).

Metric names and unique series change over time to simulate series churn.

Checkout the [blog post](https://blog.freshtracks.io/load-testing-prometheus-metric-ingestion-5b878711711c).

## configuration flags

```bash
avalanche --help
```

## run Docker image

```bash
docker run quay.io/prometheuscommunity/avalanche:main --help
```

## Endpoints

Two endpoints are available :
* `/metrics` - metrics endpoint
* `/health` - healthcheck endpoint

## build and run go binary

```bash
go install github.com/prometheus-community/avalanche/cmd@latest
go/bin/cmd --help
```
