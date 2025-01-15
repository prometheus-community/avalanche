## unreleased

## 0.7.0 / 2025-01-14

* [CHANGE] (breaking) Removed the deprecated `--metric-count` flag (use `--gauge-metric-count` instead). #119
* [CHANGE] (breaking) Removed `remote-pprof-urls` feature. #104
* [BUGFIX] Fixed `--const-label` feature. #113
* [BUGFIX] Fixed remote write run forever mode. #104
* [CHANGE] Remote write errors are logged on an ERROR level as they go (instead of the long interval). #102
* [FEATURE] Added `--remote-concurrency-limit` flag (20 default) on how many concurrent remote writes can happen. #46

## 0.6.0 / 2024-10-14

* [CHANGE] (breaking) `--metric-interval` default value is now zero (not 120). #99
* [CHANGE] (breaking) Change `out-of-order` to `--remote-out-of-order` for consistency. #101
* [CHANGE] Install command for `avalanche` moved to `cmd/avalanche/` from `cmd/` #97
* [FEATURE] Add `mtypes` binary for metric type calculation against targets #97
* [FEATURE] `--remote-requests-count` value -1 now makes remote-write send requests indefinitely #90
* [FEATURE] Add support for all metric types; deprecated --metric-count flag; --*-interval flags set to 0 means no change; added OpenMetrics support #80 #101

## 0.5.0 / 2024-09-15

* [FEATURE] add two new operation modes for metrics generation and cycling; gradual-change between min and max and double-halve #64
* (other changes, not captured due to lost 0.4.0 tag)

## 0.4.0 / 2022-03-08

Initial release under the Prometheus-community org.
