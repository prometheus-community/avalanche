# avalanche

Avalanche is a load-testing binary capable of generating metrics that can be either:

* scraped via [Prometheus scrape formats](https://prometheus.io/docs/instrumenting/exposition_formats/) (including [OpenMetrics](https://github.com/OpenObservability/OpenMetrics)) endpoint.
* written via Prometheus Remote Write (v1 only for now) to a target endpoint.

This allows load testing services that can scrape (e.g. Prometheus, OpenTelemetry Collector and so), as well as, services accepting data via Prometheus remote_write API such as [Thanos](https://github.com/thanos-io/thanos), [Cortex](https://github.com/cortexproject/cortex), [M3DB](https://m3db.github.io/m3/integrations/prometheus/), [VictoriaMetrics](https://github.com/VictoriaMetrics/VictoriaMetrics/) and other services [listed here](https://prometheus.io/docs/operating/integrations/#remote-endpoints-and-storage).

Metric names and unique series change over time to simulate series churn.

Checkout the (old-ish) [blog post](https://blog.freshtracks.io/load-testing-prometheus-metric-ingestion-5b878711711c).

## Installing

### Locally

```bash
go install github.com/prometheus-community/avalanche/cmd/avalanche@latest
${GOPATH}/bin/avalanche --help
```

### Docker 

```bash
docker run quay.io/prometheuscommunity/avalanche:latest --help
```

NOTE: We recommend using pinned image to a certain version (see all tags [here](https://quay.io/repository/prometheuscommunity/avalanche?tab=tags&tag=latest))

## Using

See [example](example/kubernetes-deployment.yaml) k8s manifest for deploying avalanche as an always running scrape target.

### Configuration

See `--help` for all flags and their documentation.

Notably, from 0.6.0 version, `avalanche` allows specifying various counts per various metric types.

You can choose you own distribution, but usually it makes more sense to mimic realistic distribution used by your example targets. Feel free to use a [handy `mtypes` Go CLI](./cmd/mtypes) to gather type distributions from a target and generate avalanche flags from it.

On top of scrape target functionality, avalanche is capable of Remote Write client load simulation, following the same, configured metric distribution via `--remote*` flags.

### Endpoints

Two endpoints are available :
* `/metrics` - metrics endpoint
* `/health` - healthcheck endpoint
