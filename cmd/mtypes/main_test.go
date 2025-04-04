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

package main

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
)

func TestCalculateTargetStatistics(t *testing.T) {
	for _, tc := range []struct {
		testName       string
		testInput      string
		expectedResult map[dto.MetricType]stats
		expectedTotal  stats
	}{
		{
			testName:  "samplePromInput",
			testInput: samplePromTestInput,
			expectedResult: map[dto.MetricType]stats{
				dto.MetricType_COUNTER:   {families: 104, series: 166, adjustedSeries: 166},
				dto.MetricType_GAUGE:     {families: 77, series: 94, adjustedSeries: 94},
				dto.MetricType_HISTOGRAM: {families: 11, series: 17, adjustedSeries: 224, buckets: 190},
				dto.MetricType_SUMMARY:   {families: 15, series: 27, adjustedSeries: 114, objectives: 60},
			},
			expectedTotal: stats{
				families: 207, series: 304, adjustedSeries: 598,
			},
		},
		{
			testName:       "empty",
			testInput:      "",
			expectedResult: map[dto.MetricType]stats{},
			expectedTotal: stats{
				families: 0, series: 0, adjustedSeries: 0,
			},
		},
		{
			testName:  "justGauge",
			testInput: justOneGaugeInput,
			expectedResult: map[dto.MetricType]stats{
				dto.MetricType_GAUGE: {families: 1, series: 1, adjustedSeries: 1},
			},
			expectedTotal: stats{
				families: 1, series: 1, adjustedSeries: 1,
			},
		},
	} {
		t.Run(tc.testName, func(t *testing.T) {
			s, err := calculateTargetStatistics(strings.NewReader(tc.testInput))
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tc.expectedResult, s, cmp.AllowUnexported(stats{})); diff != "" {
				t.Fatal(diff)
			}
			if totalDiff := cmp.Diff(tc.expectedTotal, computeTotal(s), cmp.AllowUnexported(stats{})); totalDiff != "" {
				t.Fatal(totalDiff)
			}
		})
	}
}

const justOneGaugeInput = `
# HELP up scrape successful
# TYPE up gauge
up 1
`

const samplePromTestInput = `# HELP gcm_export_pending_requests Number of in-flight requests to GCM.
# TYPE gcm_export_pending_requests gauge
gcm_export_pending_requests 1
# HELP gcm_export_projects_per_batch Number of different projects in a batch that's being sent.
# TYPE gcm_export_projects_per_batch histogram
gcm_export_projects_per_batch_bucket{le="1"} 1.0832458e+07
gcm_export_projects_per_batch_bucket{le="2"} 1.0832458e+07
gcm_export_projects_per_batch_bucket{le="4"} 1.0832458e+07
gcm_export_projects_per_batch_bucket{le="8"} 1.0832458e+07
gcm_export_projects_per_batch_bucket{le="16"} 1.0832458e+07
gcm_export_projects_per_batch_bucket{le="32"} 1.0832458e+07
gcm_export_projects_per_batch_bucket{le="64"} 1.0832458e+07
gcm_export_projects_per_batch_bucket{le="128"} 1.0832458e+07
gcm_export_projects_per_batch_bucket{le="256"} 1.0832458e+07
gcm_export_projects_per_batch_bucket{le="512"} 1.0832458e+07
gcm_export_projects_per_batch_bucket{le="1024"} 1.0832458e+07
gcm_export_projects_per_batch_bucket{le="+Inf"} 1.0832458e+07
gcm_export_projects_per_batch_sum 1.0832458e+07
gcm_export_projects_per_batch_count 1.0832458e+07
# HELP gcm_export_samples_exported_total Number of samples exported at scrape time.
# TYPE gcm_export_samples_exported_total counter
gcm_export_samples_exported_total 1.966333233e+09
# HELP gcm_export_samples_per_rpc_batch Number of samples that ended up in a single RPC batch.
# TYPE gcm_export_samples_per_rpc_batch histogram
gcm_export_samples_per_rpc_batch_bucket{le="1"} 236541
gcm_export_samples_per_rpc_batch_bucket{le="2"} 304313
gcm_export_samples_per_rpc_batch_bucket{le="5"} 355002
gcm_export_samples_per_rpc_batch_bucket{le="10"} 483585
gcm_export_samples_per_rpc_batch_bucket{le="20"} 579284
gcm_export_samples_per_rpc_batch_bucket{le="50"} 1.027749e+06
gcm_export_samples_per_rpc_batch_bucket{le="100"} 1.704702e+06
gcm_export_samples_per_rpc_batch_bucket{le="150"} 2.355089e+06
gcm_export_samples_per_rpc_batch_bucket{le="200"} 1.0832458e+07
gcm_export_samples_per_rpc_batch_bucket{le="+Inf"} 1.0832458e+07
gcm_export_samples_per_rpc_batch_sum 1.83976418e+09
gcm_export_samples_per_rpc_batch_count 1.0832458e+07
# HELP gcm_export_samples_sent_total Number of exported samples sent to GCM.
# TYPE gcm_export_samples_sent_total counter
gcm_export_samples_sent_total 1.839764124e+09
# HELP gcm_export_send_iterations_total Number of processing iterations of the sample export send handler.
# TYPE gcm_export_send_iterations_total counter
gcm_export_send_iterations_total 1.2444615e+07
# HELP gcm_export_shard_process_pending_total Number of shard retrievals with an empty result.
# TYPE gcm_export_shard_process_pending_total counter
gcm_export_shard_process_pending_total 8.66546153e+08
# HELP gcm_export_shard_process_samples_taken Number of samples taken when processing a shard.
# TYPE gcm_export_shard_process_samples_taken histogram
gcm_export_shard_process_samples_taken_bucket{le="1"} 5.6291878e+07
gcm_export_shard_process_samples_taken_bucket{le="2"} 9.1249561e+07
gcm_export_shard_process_samples_taken_bucket{le="5"} 1.27173414e+08
gcm_export_shard_process_samples_taken_bucket{le="10"} 1.34384486e+08
gcm_export_shard_process_samples_taken_bucket{le="20"} 1.68076229e+08
gcm_export_shard_process_samples_taken_bucket{le="50"} 2.04738182e+08
gcm_export_shard_process_samples_taken_bucket{le="100"} 2.04762012e+08
gcm_export_shard_process_samples_taken_bucket{le="150"} 2.04762012e+08
gcm_export_shard_process_samples_taken_bucket{le="200"} 2.04762012e+08
gcm_export_shard_process_samples_taken_bucket{le="+Inf"} 2.04762012e+08
gcm_export_shard_process_samples_taken_sum 1.83976418e+09
gcm_export_shard_process_samples_taken_count 2.04762012e+08
# HELP gcm_export_shard_process_total Number of shard retrievals.
# TYPE gcm_export_shard_process_total counter
gcm_export_shard_process_total 2.488923e+09
# HELP gcm_pool_intern_total Time series memory intern operations.
# TYPE gcm_pool_intern_total counter
gcm_pool_intern_total 4.8525498e+07
# HELP gcm_pool_release_total Time series memory intern release operations.
# TYPE gcm_pool_release_total counter
gcm_pool_release_total 4.8514709e+07
# HELP gcm_prometheus_samples_discarded_total Samples that were discarded during data model conversion.
# TYPE gcm_prometheus_samples_discarded_total counter
gcm_prometheus_samples_discarded_total{reason="staleness-marker"} 9919
gcm_prometheus_samples_discarded_total{reason="zero-buckets-bounds"} 1.076142e+07
# HELP go_gc_duration_seconds A summary of the pause duration of garbage collection cycles.
# TYPE go_gc_duration_seconds summary
go_gc_duration_seconds{quantile="0"} 5.8641e-05
go_gc_duration_seconds{quantile="0.25"} 8.4045e-05
go_gc_duration_seconds{quantile="0.5"} 0.000119609
go_gc_duration_seconds{quantile="0.75"} 0.000149195
go_gc_duration_seconds{quantile="1"} 0.000312434
go_gc_duration_seconds_sum 11.324308382
go_gc_duration_seconds_count 92364
# HELP go_goroutines Number of goroutines that currently exist.
# TYPE go_goroutines gauge
go_goroutines 112
# HELP go_info Information about the Go environment.
# TYPE go_info gauge
go_info{version="go1.20.14"} 1
# HELP go_memstats_alloc_bytes Number of bytes allocated and still in use.
# TYPE go_memstats_alloc_bytes gauge
go_memstats_alloc_bytes 1.09818568e+08
# HELP go_memstats_alloc_bytes_total Total number of bytes allocated, even if freed.
# TYPE go_memstats_alloc_bytes_total counter
go_memstats_alloc_bytes_total 7.420978933248e+12
# HELP go_memstats_buck_hash_sys_bytes Number of bytes used by the profiling bucket hash table.
# TYPE go_memstats_buck_hash_sys_bytes gauge
go_memstats_buck_hash_sys_bytes 3.653156e+06
# HELP go_memstats_frees_total Total number of frees.
# TYPE go_memstats_frees_total counter
go_memstats_frees_total 1.19996693238e+11
# HELP go_memstats_gc_sys_bytes Number of bytes used for garbage collection system metadata.
# TYPE go_memstats_gc_sys_bytes gauge
go_memstats_gc_sys_bytes 1.6556264e+07
# HELP go_memstats_heap_alloc_bytes Number of heap bytes allocated and still in use.
# TYPE go_memstats_heap_alloc_bytes gauge
go_memstats_heap_alloc_bytes 1.09818568e+08
# HELP go_memstats_heap_idle_bytes Number of heap bytes waiting to be used.
# TYPE go_memstats_heap_idle_bytes gauge
go_memstats_heap_idle_bytes 1.8628608e+08
# HELP go_memstats_heap_inuse_bytes Number of heap bytes that are in use.
# TYPE go_memstats_heap_inuse_bytes gauge
go_memstats_heap_inuse_bytes 1.3860864e+08
# HELP go_memstats_heap_objects Number of allocated objects.
# TYPE go_memstats_heap_objects gauge
go_memstats_heap_objects 738856
# HELP go_memstats_heap_released_bytes Number of heap bytes released to OS.
# TYPE go_memstats_heap_released_bytes gauge
go_memstats_heap_released_bytes 1.42557184e+08
# HELP go_memstats_heap_sys_bytes Number of heap bytes obtained from system.
# TYPE go_memstats_heap_sys_bytes gauge
go_memstats_heap_sys_bytes 3.2489472e+08
# HELP go_memstats_last_gc_time_seconds Number of seconds since 1970 of last garbage collection.
# TYPE go_memstats_last_gc_time_seconds gauge
go_memstats_last_gc_time_seconds 1.7278073317025118e+09
# HELP go_memstats_lookups_total Total number of pointer lookups.
# TYPE go_memstats_lookups_total counter
go_memstats_lookups_total 0
# HELP go_memstats_mallocs_total Total number of mallocs.
# TYPE go_memstats_mallocs_total counter
go_memstats_mallocs_total 1.19997432094e+11
# HELP go_memstats_mcache_inuse_bytes Number of bytes in use by mcache structures.
# TYPE go_memstats_mcache_inuse_bytes gauge
go_memstats_mcache_inuse_bytes 4800
# HELP go_memstats_mcache_sys_bytes Number of bytes used for mcache structures obtained from system.
# TYPE go_memstats_mcache_sys_bytes gauge
go_memstats_mcache_sys_bytes 15600
# HELP go_memstats_mspan_inuse_bytes Number of bytes in use by mspan structures.
# TYPE go_memstats_mspan_inuse_bytes gauge
go_memstats_mspan_inuse_bytes 1.8024e+06
# HELP go_memstats_mspan_sys_bytes Number of bytes used for mspan structures obtained from system.
# TYPE go_memstats_mspan_sys_bytes gauge
go_memstats_mspan_sys_bytes 3.24768e+06
# HELP go_memstats_next_gc_bytes Number of heap bytes when next garbage collection will take place.
# TYPE go_memstats_next_gc_bytes gauge
go_memstats_next_gc_bytes 1.636618e+08
# HELP go_memstats_other_sys_bytes Number of bytes used for other system allocations.
# TYPE go_memstats_other_sys_bytes gauge
go_memstats_other_sys_bytes 1.202956e+06
# HELP go_memstats_stack_inuse_bytes Number of bytes in use by the stack allocator.
# TYPE go_memstats_stack_inuse_bytes gauge
go_memstats_stack_inuse_bytes 2.260992e+06
# HELP go_memstats_stack_sys_bytes Number of bytes obtained from system for stack allocator.
# TYPE go_memstats_stack_sys_bytes gauge
go_memstats_stack_sys_bytes 2.260992e+06
# HELP go_memstats_sys_bytes Number of bytes obtained from system.
# TYPE go_memstats_sys_bytes gauge
go_memstats_sys_bytes 3.51831368e+08
# HELP go_threads Number of OS threads created.
# TYPE go_threads gauge
go_threads 12
# HELP grpc_client_handled_total Total number of RPCs completed by the client, regardless of success or failure.
# TYPE grpc_client_handled_total counter
grpc_client_handled_total{grpc_code="Canceled",grpc_method="CreateTimeSeries",grpc_service="google.monitoring.v3.MetricService",grpc_type="unary"} 9
grpc_client_handled_total{grpc_code="DeadlineExceeded",grpc_method="CreateTimeSeries",grpc_service="google.monitoring.v3.MetricService",grpc_type="unary"} 82
grpc_client_handled_total{grpc_code="Internal",grpc_method="CreateTimeSeries",grpc_service="google.monitoring.v3.MetricService",grpc_type="unary"} 4
grpc_client_handled_total{grpc_code="OK",grpc_method="CreateTimeSeries",grpc_service="google.monitoring.v3.MetricService",grpc_type="unary"} 1.0831867e+07
grpc_client_handled_total{grpc_code="Unauthenticated",grpc_method="CreateTimeSeries",grpc_service="google.monitoring.v3.MetricService",grpc_type="unary"} 1
grpc_client_handled_total{grpc_code="Unavailable",grpc_method="CreateTimeSeries",grpc_service="google.monitoring.v3.MetricService",grpc_type="unary"} 494
# HELP grpc_client_handling_seconds Histogram of response latency (seconds) of the gRPC until it is finished by the application.
# TYPE grpc_client_handling_seconds histogram
grpc_client_handling_seconds_bucket{grpc_method="CreateTimeSeries",grpc_service="google.monitoring.v3.MetricService",grpc_type="unary",le="0.005"} 0
grpc_client_handling_seconds_bucket{grpc_method="CreateTimeSeries",grpc_service="google.monitoring.v3.MetricService",grpc_type="unary",le="0.01"} 0
grpc_client_handling_seconds_bucket{grpc_method="CreateTimeSeries",grpc_service="google.monitoring.v3.MetricService",grpc_type="unary",le="0.025"} 34059
grpc_client_handling_seconds_bucket{grpc_method="CreateTimeSeries",grpc_service="google.monitoring.v3.MetricService",grpc_type="unary",le="0.05"} 1.127825e+06
grpc_client_handling_seconds_bucket{grpc_method="CreateTimeSeries",grpc_service="google.monitoring.v3.MetricService",grpc_type="unary",le="0.1"} 9.058302e+06
grpc_client_handling_seconds_bucket{grpc_method="CreateTimeSeries",grpc_service="google.monitoring.v3.MetricService",grpc_type="unary",le="0.25"} 1.0721886e+07
grpc_client_handling_seconds_bucket{grpc_method="CreateTimeSeries",grpc_service="google.monitoring.v3.MetricService",grpc_type="unary",le="0.5"} 1.0759498e+07
grpc_client_handling_seconds_bucket{grpc_method="CreateTimeSeries",grpc_service="google.monitoring.v3.MetricService",grpc_type="unary",le="1"} 1.0774023e+07
grpc_client_handling_seconds_bucket{grpc_method="CreateTimeSeries",grpc_service="google.monitoring.v3.MetricService",grpc_type="unary",le="2.5"} 1.079026e+07
grpc_client_handling_seconds_bucket{grpc_method="CreateTimeSeries",grpc_service="google.monitoring.v3.MetricService",grpc_type="unary",le="5"} 1.0800098e+07
grpc_client_handling_seconds_bucket{grpc_method="CreateTimeSeries",grpc_service="google.monitoring.v3.MetricService",grpc_type="unary",le="10"} 1.0832159e+07
grpc_client_handling_seconds_bucket{grpc_method="CreateTimeSeries",grpc_service="google.monitoring.v3.MetricService",grpc_type="unary",le="15"} 1.0832261e+07
grpc_client_handling_seconds_bucket{grpc_method="CreateTimeSeries",grpc_service="google.monitoring.v3.MetricService",grpc_type="unary",le="20"} 1.0832299e+07
grpc_client_handling_seconds_bucket{grpc_method="CreateTimeSeries",grpc_service="google.monitoring.v3.MetricService",grpc_type="unary",le="30"} 1.0832376e+07
grpc_client_handling_seconds_bucket{grpc_method="CreateTimeSeries",grpc_service="google.monitoring.v3.MetricService",grpc_type="unary",le="40"} 1.0832457e+07
grpc_client_handling_seconds_bucket{grpc_method="CreateTimeSeries",grpc_service="google.monitoring.v3.MetricService",grpc_type="unary",le="50"} 1.0832457e+07
grpc_client_handling_seconds_bucket{grpc_method="CreateTimeSeries",grpc_service="google.monitoring.v3.MetricService",grpc_type="unary",le="60"} 1.0832457e+07
grpc_client_handling_seconds_bucket{grpc_method="CreateTimeSeries",grpc_service="google.monitoring.v3.MetricService",grpc_type="unary",le="+Inf"} 1.0832457e+07
grpc_client_handling_seconds_sum{grpc_method="CreateTimeSeries",grpc_service="google.monitoring.v3.MetricService",grpc_type="unary"} 1.2123103039707085e+06
grpc_client_handling_seconds_count{grpc_method="CreateTimeSeries",grpc_service="google.monitoring.v3.MetricService",grpc_type="unary"} 1.0832457e+07
# HELP grpc_client_msg_received_total Total number of RPC stream messages received by the client.
# TYPE grpc_client_msg_received_total counter
grpc_client_msg_received_total{grpc_method="CreateTimeSeries",grpc_service="google.monitoring.v3.MetricService",grpc_type="unary"} 590
# HELP grpc_client_msg_sent_total Total number of gRPC stream messages sent by the client.
# TYPE grpc_client_msg_sent_total counter
grpc_client_msg_sent_total{grpc_method="CreateTimeSeries",grpc_service="google.monitoring.v3.MetricService",grpc_type="unary"} 1.0832458e+07
# HELP grpc_client_started_total Total number of RPCs started on the client.
# TYPE grpc_client_started_total counter
grpc_client_started_total{grpc_method="CreateTimeSeries",grpc_service="google.monitoring.v3.MetricService",grpc_type="unary"} 1.0832458e+07
# HELP net_conntrack_dialer_conn_attempted_total Total number of connections attempted by the given dialer a given name.
# TYPE net_conntrack_dialer_conn_attempted_total counter
net_conntrack_dialer_conn_attempted_total{dialer_name="cadvisor"} 94
net_conntrack_dialer_conn_attempted_total{dialer_name="default"} 0
net_conntrack_dialer_conn_attempted_total{dialer_name="kube-state-metrics"} 2
net_conntrack_dialer_conn_attempted_total{dialer_name="pods"} 179445
# HELP net_conntrack_dialer_conn_closed_total Total number of connections closed which originated from the dialer of a given name.
# TYPE net_conntrack_dialer_conn_closed_total counter
net_conntrack_dialer_conn_closed_total{dialer_name="cadvisor"} 3
net_conntrack_dialer_conn_closed_total{dialer_name="default"} 0
net_conntrack_dialer_conn_closed_total{dialer_name="kube-state-metrics"} 0
net_conntrack_dialer_conn_closed_total{dialer_name="pods"} 179394
# HELP net_conntrack_dialer_conn_established_total Total number of connections successfully established by the given dialer a given name.
# TYPE net_conntrack_dialer_conn_established_total counter
net_conntrack_dialer_conn_established_total{dialer_name="cadvisor"} 4
net_conntrack_dialer_conn_established_total{dialer_name="default"} 0
net_conntrack_dialer_conn_established_total{dialer_name="kube-state-metrics"} 2
net_conntrack_dialer_conn_established_total{dialer_name="pods"} 179399
# HELP net_conntrack_dialer_conn_failed_total Total number of connections failed to dial by the dialer a given name.
# TYPE net_conntrack_dialer_conn_failed_total counter
net_conntrack_dialer_conn_failed_total{dialer_name="cadvisor",reason="refused"} 7
net_conntrack_dialer_conn_failed_total{dialer_name="cadvisor",reason="resolution"} 0
net_conntrack_dialer_conn_failed_total{dialer_name="cadvisor",reason="timeout"} 83
net_conntrack_dialer_conn_failed_total{dialer_name="cadvisor",reason="unknown"} 90
net_conntrack_dialer_conn_failed_total{dialer_name="default",reason="refused"} 0
net_conntrack_dialer_conn_failed_total{dialer_name="default",reason="resolution"} 0
net_conntrack_dialer_conn_failed_total{dialer_name="default",reason="timeout"} 0
net_conntrack_dialer_conn_failed_total{dialer_name="default",reason="unknown"} 0
net_conntrack_dialer_conn_failed_total{dialer_name="kube-state-metrics",reason="refused"} 0
net_conntrack_dialer_conn_failed_total{dialer_name="kube-state-metrics",reason="resolution"} 0
net_conntrack_dialer_conn_failed_total{dialer_name="kube-state-metrics",reason="timeout"} 0
net_conntrack_dialer_conn_failed_total{dialer_name="kube-state-metrics",reason="unknown"} 0
net_conntrack_dialer_conn_failed_total{dialer_name="pods",reason="refused"} 4
net_conntrack_dialer_conn_failed_total{dialer_name="pods",reason="resolution"} 0
net_conntrack_dialer_conn_failed_total{dialer_name="pods",reason="timeout"} 42
net_conntrack_dialer_conn_failed_total{dialer_name="pods",reason="unknown"} 46
# HELP net_conntrack_listener_conn_accepted_total Total number of connections opened to the listener of a given name.
# TYPE net_conntrack_listener_conn_accepted_total counter
net_conntrack_listener_conn_accepted_total{listener_name="http"} 8
# HELP net_conntrack_listener_conn_closed_total Total number of connections closed that were made to the listener of a given name.
# TYPE net_conntrack_listener_conn_closed_total counter
net_conntrack_listener_conn_closed_total{listener_name="http"} 3
# HELP process_cpu_seconds_total Total user and system CPU time spent in seconds.
# TYPE process_cpu_seconds_total counter
process_cpu_seconds_total 64026.65
# HELP process_max_fds Maximum number of open file descriptors.
# TYPE process_max_fds gauge
process_max_fds 1.048576e+06
# HELP process_open_fds Number of open file descriptors.
# TYPE process_open_fds gauge
process_open_fds 105
# HELP process_resident_memory_bytes Resident memory size in bytes.
# TYPE process_resident_memory_bytes gauge
process_resident_memory_bytes 2.81624576e+08
# HELP process_start_time_seconds Start time of the process since unix epoch in seconds.
# TYPE process_start_time_seconds gauge
process_start_time_seconds 1.72511698039e+09
# HELP process_virtual_memory_bytes Virtual memory size in bytes.
# TYPE process_virtual_memory_bytes gauge
process_virtual_memory_bytes 2.8450332672e+10
# HELP process_virtual_memory_max_bytes Maximum amount of virtual memory available in bytes.
# TYPE process_virtual_memory_max_bytes gauge
process_virtual_memory_max_bytes 1.8446744073709552e+19
# HELP prometheus_api_remote_read_queries The current number of remote read queries being executed or waiting.
# TYPE prometheus_api_remote_read_queries gauge
prometheus_api_remote_read_queries 0
# HELP prometheus_build_info A metric with a constant '1' value labeled by version, revision, branch, goversion from which prometheus was built, and the goos and goarch for the build.
# TYPE prometheus_build_info gauge
prometheus_build_info{branch="",goarch="amd64",goos="linux",goversion="go1.20.14",revision="d7b199739aa7e0d00e7ebd0792339dd4b167a269-modified",tags="builtinassets",version="2.45.3"} 1
# HELP prometheus_config_last_reload_success_timestamp_seconds Timestamp of the last successful configuration reload.
# TYPE prometheus_config_last_reload_success_timestamp_seconds gauge
prometheus_config_last_reload_success_timestamp_seconds 1.725116982549508e+09
# HELP prometheus_config_last_reload_successful Whether the last configuration reload attempt was successful.
# TYPE prometheus_config_last_reload_successful gauge
prometheus_config_last_reload_successful 1
# HELP prometheus_engine_queries The current number of queries being executed or waiting.
# TYPE prometheus_engine_queries gauge
prometheus_engine_queries 0
# HELP prometheus_engine_queries_concurrent_max The max number of concurrent queries.
# TYPE prometheus_engine_queries_concurrent_max gauge
prometheus_engine_queries_concurrent_max 20
# HELP prometheus_engine_query_duration_seconds Query timings
# TYPE prometheus_engine_query_duration_seconds summary
prometheus_engine_query_duration_seconds{slice="inner_eval",quantile="0.5"} NaN
prometheus_engine_query_duration_seconds{slice="inner_eval",quantile="0.9"} NaN
prometheus_engine_query_duration_seconds{slice="inner_eval",quantile="0.99"} NaN
prometheus_engine_query_duration_seconds_sum{slice="inner_eval"} 0
prometheus_engine_query_duration_seconds_count{slice="inner_eval"} 0
prometheus_engine_query_duration_seconds{slice="prepare_time",quantile="0.5"} NaN
prometheus_engine_query_duration_seconds{slice="prepare_time",quantile="0.9"} NaN
prometheus_engine_query_duration_seconds{slice="prepare_time",quantile="0.99"} NaN
prometheus_engine_query_duration_seconds_sum{slice="prepare_time"} 0
prometheus_engine_query_duration_seconds_count{slice="prepare_time"} 0
prometheus_engine_query_duration_seconds{slice="queue_time",quantile="0.5"} NaN
prometheus_engine_query_duration_seconds{slice="queue_time",quantile="0.9"} NaN
prometheus_engine_query_duration_seconds{slice="queue_time",quantile="0.99"} NaN
prometheus_engine_query_duration_seconds_sum{slice="queue_time"} 0
prometheus_engine_query_duration_seconds_count{slice="queue_time"} 0
prometheus_engine_query_duration_seconds{slice="result_sort",quantile="0.5"} NaN
prometheus_engine_query_duration_seconds{slice="result_sort",quantile="0.9"} NaN
prometheus_engine_query_duration_seconds{slice="result_sort",quantile="0.99"} NaN
prometheus_engine_query_duration_seconds_sum{slice="result_sort"} 0
prometheus_engine_query_duration_seconds_count{slice="result_sort"} 0
# HELP prometheus_engine_query_log_enabled State of the query log.
# TYPE prometheus_engine_query_log_enabled gauge
prometheus_engine_query_log_enabled 0
# HELP prometheus_engine_query_log_failures_total The number of query log failures.
# TYPE prometheus_engine_query_log_failures_total counter
prometheus_engine_query_log_failures_total 0
# HELP prometheus_engine_query_samples_total The total number of samples loaded by all queries.
# TYPE prometheus_engine_query_samples_total counter
prometheus_engine_query_samples_total 0
# HELP prometheus_http_request_duration_seconds Histogram of latencies for HTTP requests.
# TYPE prometheus_http_request_duration_seconds histogram
prometheus_http_request_duration_seconds_bucket{handler="/-/ready",le="0.1"} 2
prometheus_http_request_duration_seconds_bucket{handler="/-/ready",le="0.2"} 2
prometheus_http_request_duration_seconds_bucket{handler="/-/ready",le="0.4"} 2
prometheus_http_request_duration_seconds_bucket{handler="/-/ready",le="1"} 2
prometheus_http_request_duration_seconds_bucket{handler="/-/ready",le="3"} 2
prometheus_http_request_duration_seconds_bucket{handler="/-/ready",le="8"} 2
prometheus_http_request_duration_seconds_bucket{handler="/-/ready",le="20"} 2
prometheus_http_request_duration_seconds_bucket{handler="/-/ready",le="60"} 2
prometheus_http_request_duration_seconds_bucket{handler="/-/ready",le="120"} 2
prometheus_http_request_duration_seconds_bucket{handler="/-/ready",le="+Inf"} 2
prometheus_http_request_duration_seconds_sum{handler="/-/ready"} 4.7443999999999995e-05
prometheus_http_request_duration_seconds_count{handler="/-/ready"} 2
prometheus_http_request_duration_seconds_bucket{handler="/-/reload",le="0.1"} 1
prometheus_http_request_duration_seconds_bucket{handler="/-/reload",le="0.2"} 1
prometheus_http_request_duration_seconds_bucket{handler="/-/reload",le="0.4"} 1
prometheus_http_request_duration_seconds_bucket{handler="/-/reload",le="1"} 1
prometheus_http_request_duration_seconds_bucket{handler="/-/reload",le="3"} 1
prometheus_http_request_duration_seconds_bucket{handler="/-/reload",le="8"} 1
prometheus_http_request_duration_seconds_bucket{handler="/-/reload",le="20"} 1
prometheus_http_request_duration_seconds_bucket{handler="/-/reload",le="60"} 1
prometheus_http_request_duration_seconds_bucket{handler="/-/reload",le="120"} 1
prometheus_http_request_duration_seconds_bucket{handler="/-/reload",le="+Inf"} 1
prometheus_http_request_duration_seconds_sum{handler="/-/reload"} 0.002356799
prometheus_http_request_duration_seconds_count{handler="/-/reload"} 1
prometheus_http_request_duration_seconds_bucket{handler="/debug/*subpath",le="0.1"} 358716
prometheus_http_request_duration_seconds_bucket{handler="/debug/*subpath",le="0.2"} 358716
prometheus_http_request_duration_seconds_bucket{handler="/debug/*subpath",le="0.4"} 358716
prometheus_http_request_duration_seconds_bucket{handler="/debug/*subpath",le="1"} 358716
prometheus_http_request_duration_seconds_bucket{handler="/debug/*subpath",le="3"} 358716
prometheus_http_request_duration_seconds_bucket{handler="/debug/*subpath",le="8"} 358716
prometheus_http_request_duration_seconds_bucket{handler="/debug/*subpath",le="20"} 358716
prometheus_http_request_duration_seconds_bucket{handler="/debug/*subpath",le="60"} 448346
prometheus_http_request_duration_seconds_bucket{handler="/debug/*subpath",le="120"} 448346
prometheus_http_request_duration_seconds_bucket{handler="/debug/*subpath",le="+Inf"} 448346
prometheus_http_request_duration_seconds_sum{handler="/debug/*subpath"} 2.692262582005182e+06
prometheus_http_request_duration_seconds_count{handler="/debug/*subpath"} 448346
prometheus_http_request_duration_seconds_bucket{handler="/metrics",le="0.1"} 179357
prometheus_http_request_duration_seconds_bucket{handler="/metrics",le="0.2"} 179357
prometheus_http_request_duration_seconds_bucket{handler="/metrics",le="0.4"} 179357
prometheus_http_request_duration_seconds_bucket{handler="/metrics",le="1"} 179357
prometheus_http_request_duration_seconds_bucket{handler="/metrics",le="3"} 179357
prometheus_http_request_duration_seconds_bucket{handler="/metrics",le="8"} 179357
prometheus_http_request_duration_seconds_bucket{handler="/metrics",le="20"} 179357
prometheus_http_request_duration_seconds_bucket{handler="/metrics",le="60"} 179357
prometheus_http_request_duration_seconds_bucket{handler="/metrics",le="120"} 179357
prometheus_http_request_duration_seconds_bucket{handler="/metrics",le="+Inf"} 179357
prometheus_http_request_duration_seconds_sum{handler="/metrics"} 552.2173053479947
prometheus_http_request_duration_seconds_count{handler="/metrics"} 179357
# HELP prometheus_http_requests_total Counter of HTTP requests.
# TYPE prometheus_http_requests_total counter
prometheus_http_requests_total{code="200",handler="/-/ready"} 1
prometheus_http_requests_total{code="200",handler="/-/reload"} 1
prometheus_http_requests_total{code="200",handler="/debug/*subpath"} 448346
prometheus_http_requests_total{code="200",handler="/metrics"} 179357
prometheus_http_requests_total{code="503",handler="/-/ready"} 1
# HELP prometheus_http_response_size_bytes Histogram of response size for HTTP requests.
# TYPE prometheus_http_response_size_bytes histogram
prometheus_http_response_size_bytes_bucket{handler="/-/ready",le="100"} 2
prometheus_http_response_size_bytes_bucket{handler="/-/ready",le="1000"} 2
prometheus_http_response_size_bytes_bucket{handler="/-/ready",le="10000"} 2
prometheus_http_response_size_bytes_bucket{handler="/-/ready",le="100000"} 2
prometheus_http_response_size_bytes_bucket{handler="/-/ready",le="1e+06"} 2
prometheus_http_response_size_bytes_bucket{handler="/-/ready",le="1e+07"} 2
prometheus_http_response_size_bytes_bucket{handler="/-/ready",le="1e+08"} 2
prometheus_http_response_size_bytes_bucket{handler="/-/ready",le="1e+09"} 2
prometheus_http_response_size_bytes_bucket{handler="/-/ready",le="+Inf"} 2
prometheus_http_response_size_bytes_sum{handler="/-/ready"} 47
prometheus_http_response_size_bytes_count{handler="/-/ready"} 2
prometheus_http_response_size_bytes_bucket{handler="/-/reload",le="100"} 1
prometheus_http_response_size_bytes_bucket{handler="/-/reload",le="1000"} 1
prometheus_http_response_size_bytes_bucket{handler="/-/reload",le="10000"} 1
prometheus_http_response_size_bytes_bucket{handler="/-/reload",le="100000"} 1
prometheus_http_response_size_bytes_bucket{handler="/-/reload",le="1e+06"} 1
prometheus_http_response_size_bytes_bucket{handler="/-/reload",le="1e+07"} 1
prometheus_http_response_size_bytes_bucket{handler="/-/reload",le="1e+08"} 1
prometheus_http_response_size_bytes_bucket{handler="/-/reload",le="1e+09"} 1
prometheus_http_response_size_bytes_bucket{handler="/-/reload",le="+Inf"} 1
prometheus_http_response_size_bytes_sum{handler="/-/reload"} 0
prometheus_http_response_size_bytes_count{handler="/-/reload"} 1
prometheus_http_response_size_bytes_bucket{handler="/debug/*subpath",le="100"} 0
prometheus_http_response_size_bytes_bucket{handler="/debug/*subpath",le="1000"} 179358
prometheus_http_response_size_bytes_bucket{handler="/debug/*subpath",le="10000"} 269558
prometheus_http_response_size_bytes_bucket{handler="/debug/*subpath",le="100000"} 359969
prometheus_http_response_size_bytes_bucket{handler="/debug/*subpath",le="1e+06"} 448346
prometheus_http_response_size_bytes_bucket{handler="/debug/*subpath",le="1e+07"} 448346
prometheus_http_response_size_bytes_bucket{handler="/debug/*subpath",le="1e+08"} 448346
prometheus_http_response_size_bytes_bucket{handler="/debug/*subpath",le="1e+09"} 448346
prometheus_http_response_size_bytes_bucket{handler="/debug/*subpath",le="+Inf"} 448346
prometheus_http_response_size_bytes_sum{handler="/debug/*subpath"} 1.7640059511e+10
prometheus_http_response_size_bytes_count{handler="/debug/*subpath"} 448346
prometheus_http_response_size_bytes_bucket{handler="/metrics",le="100"} 0
prometheus_http_response_size_bytes_bucket{handler="/metrics",le="1000"} 0
prometheus_http_response_size_bytes_bucket{handler="/metrics",le="10000"} 191
prometheus_http_response_size_bytes_bucket{handler="/metrics",le="100000"} 179357
prometheus_http_response_size_bytes_bucket{handler="/metrics",le="1e+06"} 179357
prometheus_http_response_size_bytes_bucket{handler="/metrics",le="1e+07"} 179357
prometheus_http_response_size_bytes_bucket{handler="/metrics",le="1e+08"} 179357
prometheus_http_response_size_bytes_bucket{handler="/metrics",le="1e+09"} 179357
prometheus_http_response_size_bytes_bucket{handler="/metrics",le="+Inf"} 179357
prometheus_http_response_size_bytes_sum{handler="/metrics"} 1.895799365e+09
prometheus_http_response_size_bytes_count{handler="/metrics"} 179357
# HELP prometheus_notifications_alertmanagers_discovered The number of alertmanagers discovered and active.
# TYPE prometheus_notifications_alertmanagers_discovered gauge
prometheus_notifications_alertmanagers_discovered 0
# HELP prometheus_notifications_dropped_total Total number of alerts dropped due to errors when sending to Alertmanager.
# TYPE prometheus_notifications_dropped_total counter
prometheus_notifications_dropped_total 0
# HELP prometheus_notifications_queue_capacity The capacity of the alert notifications queue.
# TYPE prometheus_notifications_queue_capacity gauge
prometheus_notifications_queue_capacity 10000
# HELP prometheus_notifications_queue_length The number of alert notifications in the queue.
# TYPE prometheus_notifications_queue_length gauge
prometheus_notifications_queue_length 0
# HELP prometheus_ready Whether Prometheus startup was fully completed and the server is ready for normal operation.
# TYPE prometheus_ready gauge
prometheus_ready 1
# HELP prometheus_remote_storage_exemplars_in_total Exemplars in to remote storage, compare to exemplars out for queue managers.
# TYPE prometheus_remote_storage_exemplars_in_total counter
prometheus_remote_storage_exemplars_in_total 0
# HELP prometheus_remote_storage_highest_timestamp_in_seconds Highest timestamp that has come into the remote storage via the Appender interface, in seconds since epoch.
# TYPE prometheus_remote_storage_highest_timestamp_in_seconds gauge
prometheus_remote_storage_highest_timestamp_in_seconds 1.727807345e+09
# HELP prometheus_remote_storage_histograms_in_total HistogramSamples in to remote storage, compare to histograms out for queue managers.
# TYPE prometheus_remote_storage_histograms_in_total counter
prometheus_remote_storage_histograms_in_total 0
# HELP prometheus_remote_storage_samples_in_total Samples in to remote storage, compare to samples out for queue managers.
# TYPE prometheus_remote_storage_samples_in_total counter
prometheus_remote_storage_samples_in_total 1.966333233e+09
# HELP prometheus_remote_storage_string_interner_zero_reference_releases_total The number of times release has been called for strings that are not interned.
# TYPE prometheus_remote_storage_string_interner_zero_reference_releases_total counter
prometheus_remote_storage_string_interner_zero_reference_releases_total 0
# HELP prometheus_rule_evaluation_duration_seconds The duration for a rule to execute.
# TYPE prometheus_rule_evaluation_duration_seconds summary
prometheus_rule_evaluation_duration_seconds{quantile="0.5"} NaN
prometheus_rule_evaluation_duration_seconds{quantile="0.9"} NaN
prometheus_rule_evaluation_duration_seconds{quantile="0.99"} NaN
prometheus_rule_evaluation_duration_seconds_sum 0
prometheus_rule_evaluation_duration_seconds_count 0
# HELP prometheus_rule_group_duration_seconds The duration of rule group evaluations.
# TYPE prometheus_rule_group_duration_seconds summary
prometheus_rule_group_duration_seconds{quantile="0.01"} NaN
prometheus_rule_group_duration_seconds{quantile="0.05"} NaN
prometheus_rule_group_duration_seconds{quantile="0.5"} NaN
prometheus_rule_group_duration_seconds{quantile="0.9"} NaN
prometheus_rule_group_duration_seconds{quantile="0.99"} NaN
prometheus_rule_group_duration_seconds_sum 0
prometheus_rule_group_duration_seconds_count 0
# HELP prometheus_sd_azure_failures_total Number of Azure service discovery refresh failures.
# TYPE prometheus_sd_azure_failures_total counter
prometheus_sd_azure_failures_total 0
# HELP prometheus_sd_consul_rpc_duration_seconds The duration of a Consul RPC call in seconds.
# TYPE prometheus_sd_consul_rpc_duration_seconds summary
prometheus_sd_consul_rpc_duration_seconds{call="service",endpoint="catalog",quantile="0.5"} NaN
prometheus_sd_consul_rpc_duration_seconds{call="service",endpoint="catalog",quantile="0.9"} NaN
prometheus_sd_consul_rpc_duration_seconds{call="service",endpoint="catalog",quantile="0.99"} NaN
prometheus_sd_consul_rpc_duration_seconds_sum{call="service",endpoint="catalog"} 0
prometheus_sd_consul_rpc_duration_seconds_count{call="service",endpoint="catalog"} 0
prometheus_sd_consul_rpc_duration_seconds{call="services",endpoint="catalog",quantile="0.5"} NaN
prometheus_sd_consul_rpc_duration_seconds{call="services",endpoint="catalog",quantile="0.9"} NaN
prometheus_sd_consul_rpc_duration_seconds{call="services",endpoint="catalog",quantile="0.99"} NaN
prometheus_sd_consul_rpc_duration_seconds_sum{call="services",endpoint="catalog"} 0
prometheus_sd_consul_rpc_duration_seconds_count{call="services",endpoint="catalog"} 0
# HELP prometheus_sd_consul_rpc_failures_total The number of Consul RPC call failures.
# TYPE prometheus_sd_consul_rpc_failures_total counter
prometheus_sd_consul_rpc_failures_total 0
# HELP prometheus_sd_discovered_targets Current number of discovered targets.
# TYPE prometheus_sd_discovered_targets gauge
prometheus_sd_discovered_targets{config="cadvisor",name="scrape"} 2
prometheus_sd_discovered_targets{config="kube-state-metrics",name="scrape"} 9
prometheus_sd_discovered_targets{config="pods",name="scrape"} 82
# HELP prometheus_sd_dns_lookup_failures_total The number of DNS-SD lookup failures.
# TYPE prometheus_sd_dns_lookup_failures_total counter
prometheus_sd_dns_lookup_failures_total 0
# HELP prometheus_sd_dns_lookups_total The number of DNS-SD lookups.
# TYPE prometheus_sd_dns_lookups_total counter
prometheus_sd_dns_lookups_total 0
# HELP prometheus_sd_failed_configs Current number of service discovery configurations that failed to load.
# TYPE prometheus_sd_failed_configs gauge
prometheus_sd_failed_configs{name="notify"} 0
prometheus_sd_failed_configs{name="scrape"} 0
# HELP prometheus_sd_file_read_errors_total The number of File-SD read errors.
# TYPE prometheus_sd_file_read_errors_total counter
prometheus_sd_file_read_errors_total 0
# HELP prometheus_sd_file_scan_duration_seconds The duration of the File-SD scan in seconds.
# TYPE prometheus_sd_file_scan_duration_seconds summary
prometheus_sd_file_scan_duration_seconds{quantile="0.5"} NaN
prometheus_sd_file_scan_duration_seconds{quantile="0.9"} NaN
prometheus_sd_file_scan_duration_seconds{quantile="0.99"} NaN
prometheus_sd_file_scan_duration_seconds_sum 0
prometheus_sd_file_scan_duration_seconds_count 0
# HELP prometheus_sd_file_watcher_errors_total The number of File-SD errors caused by filesystem watch failures.
# TYPE prometheus_sd_file_watcher_errors_total counter
prometheus_sd_file_watcher_errors_total 0
# HELP prometheus_sd_http_failures_total Number of HTTP service discovery refresh failures.
# TYPE prometheus_sd_http_failures_total counter
prometheus_sd_http_failures_total 0
# HELP prometheus_sd_kubernetes_events_total The number of Kubernetes events handled.
# TYPE prometheus_sd_kubernetes_events_total counter
prometheus_sd_kubernetes_events_total{event="add",role="endpoints"} 0
prometheus_sd_kubernetes_events_total{event="add",role="endpointslice"} 0
prometheus_sd_kubernetes_events_total{event="add",role="ingress"} 0
prometheus_sd_kubernetes_events_total{event="add",role="node"} 5
prometheus_sd_kubernetes_events_total{event="add",role="pod"} 169
prometheus_sd_kubernetes_events_total{event="add",role="service"} 9
prometheus_sd_kubernetes_events_total{event="delete",role="endpoints"} 0
prometheus_sd_kubernetes_events_total{event="delete",role="endpointslice"} 0
prometheus_sd_kubernetes_events_total{event="delete",role="ingress"} 0
prometheus_sd_kubernetes_events_total{event="delete",role="node"} 3
prometheus_sd_kubernetes_events_total{event="delete",role="pod"} 128
prometheus_sd_kubernetes_events_total{event="delete",role="service"} 2
prometheus_sd_kubernetes_events_total{event="update",role="endpoints"} 0
prometheus_sd_kubernetes_events_total{event="update",role="endpointslice"} 0
prometheus_sd_kubernetes_events_total{event="update",role="ingress"} 0
prometheus_sd_kubernetes_events_total{event="update",role="node"} 35525
prometheus_sd_kubernetes_events_total{event="update",role="pod"} 1034
prometheus_sd_kubernetes_events_total{event="update",role="service"} 29
# HELP prometheus_sd_kubernetes_http_request_duration_seconds Summary of latencies for HTTP requests to the Kubernetes API by endpoint.
# TYPE prometheus_sd_kubernetes_http_request_duration_seconds summary
prometheus_sd_kubernetes_http_request_duration_seconds_sum{endpoint="/api/v1/nodes"} 0.017348603
prometheus_sd_kubernetes_http_request_duration_seconds_count{endpoint="/api/v1/nodes"} 4
prometheus_sd_kubernetes_http_request_duration_seconds_sum{endpoint="/api/v1/pods"} 0.038949225999999997
prometheus_sd_kubernetes_http_request_duration_seconds_count{endpoint="/api/v1/pods"} 4
prometheus_sd_kubernetes_http_request_duration_seconds_sum{endpoint="/api/v1/services"} 0.014277334000000001
prometheus_sd_kubernetes_http_request_duration_seconds_count{endpoint="/api/v1/services"} 4
# HELP prometheus_sd_kubernetes_http_request_total Total number of HTTP requests to the Kubernetes API by status code.
# TYPE prometheus_sd_kubernetes_http_request_total counter
prometheus_sd_kubernetes_http_request_total{status_code="200"} 17957
prometheus_sd_kubernetes_http_request_total{status_code="<error>"} 83
# HELP prometheus_sd_kubernetes_workqueue_depth Current depth of the work queue.
# TYPE prometheus_sd_kubernetes_workqueue_depth gauge
prometheus_sd_kubernetes_workqueue_depth{queue_name="node"} 0
prometheus_sd_kubernetes_workqueue_depth{queue_name="pod"} 0
prometheus_sd_kubernetes_workqueue_depth{queue_name="service"} 0
# HELP prometheus_sd_kubernetes_workqueue_items_total Total number of items added to the work queue.
# TYPE prometheus_sd_kubernetes_workqueue_items_total counter
prometheus_sd_kubernetes_workqueue_items_total{queue_name="node"} 35533
prometheus_sd_kubernetes_workqueue_items_total{queue_name="pod"} 1329
prometheus_sd_kubernetes_workqueue_items_total{queue_name="service"} 40
# HELP prometheus_sd_kubernetes_workqueue_latency_seconds How long an item stays in the work queue.
# TYPE prometheus_sd_kubernetes_workqueue_latency_seconds summary
prometheus_sd_kubernetes_workqueue_latency_seconds_sum{queue_name="node"} 0.49772388200000356
prometheus_sd_kubernetes_workqueue_latency_seconds_count{queue_name="node"} 35533
prometheus_sd_kubernetes_workqueue_latency_seconds_sum{queue_name="pod"} 4.155762530999996
prometheus_sd_kubernetes_workqueue_latency_seconds_count{queue_name="pod"} 1329
prometheus_sd_kubernetes_workqueue_latency_seconds_sum{queue_name="service"} 0.8281205150000001
prometheus_sd_kubernetes_workqueue_latency_seconds_count{queue_name="service"} 40
# HELP prometheus_sd_kubernetes_workqueue_longest_running_processor_seconds Duration of the longest running processor in the work queue.
# TYPE prometheus_sd_kubernetes_workqueue_longest_running_processor_seconds gauge
prometheus_sd_kubernetes_workqueue_longest_running_processor_seconds{queue_name="node"} 0
prometheus_sd_kubernetes_workqueue_longest_running_processor_seconds{queue_name="pod"} 0
prometheus_sd_kubernetes_workqueue_longest_running_processor_seconds{queue_name="service"} 0
# HELP prometheus_sd_kubernetes_workqueue_unfinished_work_seconds How long an item has remained unfinished in the work queue.
# TYPE prometheus_sd_kubernetes_workqueue_unfinished_work_seconds gauge
prometheus_sd_kubernetes_workqueue_unfinished_work_seconds{queue_name="node"} 0
prometheus_sd_kubernetes_workqueue_unfinished_work_seconds{queue_name="pod"} 0
prometheus_sd_kubernetes_workqueue_unfinished_work_seconds{queue_name="service"} 0
# HELP prometheus_sd_kubernetes_workqueue_work_duration_seconds How long processing an item from the work queue takes.
# TYPE prometheus_sd_kubernetes_workqueue_work_duration_seconds summary
prometheus_sd_kubernetes_workqueue_work_duration_seconds_sum{queue_name="node"} 5.840500786999983
prometheus_sd_kubernetes_workqueue_work_duration_seconds_count{queue_name="node"} 35533
prometheus_sd_kubernetes_workqueue_work_duration_seconds_sum{queue_name="pod"} 0.034607483000000085
prometheus_sd_kubernetes_workqueue_work_duration_seconds_count{queue_name="pod"} 1329
prometheus_sd_kubernetes_workqueue_work_duration_seconds_sum{queue_name="service"} 0.0010254919999999998
prometheus_sd_kubernetes_workqueue_work_duration_seconds_count{queue_name="service"} 40
# HELP prometheus_sd_kuma_fetch_duration_seconds The duration of a Kuma MADS fetch call.
# TYPE prometheus_sd_kuma_fetch_duration_seconds summary
prometheus_sd_kuma_fetch_duration_seconds{quantile="0.5"} NaN
prometheus_sd_kuma_fetch_duration_seconds{quantile="0.9"} NaN
prometheus_sd_kuma_fetch_duration_seconds{quantile="0.99"} NaN
prometheus_sd_kuma_fetch_duration_seconds_sum 0
prometheus_sd_kuma_fetch_duration_seconds_count 0
# HELP prometheus_sd_kuma_fetch_failures_total The number of Kuma MADS fetch call failures.
# TYPE prometheus_sd_kuma_fetch_failures_total counter
prometheus_sd_kuma_fetch_failures_total 0
# HELP prometheus_sd_kuma_fetch_skipped_updates_total The number of Kuma MADS fetch calls that result in no updates to the targets.
# TYPE prometheus_sd_kuma_fetch_skipped_updates_total counter
prometheus_sd_kuma_fetch_skipped_updates_total 0
# HELP prometheus_sd_linode_failures_total Number of Linode service discovery refresh failures.
# TYPE prometheus_sd_linode_failures_total counter
prometheus_sd_linode_failures_total 0
# HELP prometheus_sd_nomad_failures_total Number of nomad service discovery refresh failures.
# TYPE prometheus_sd_nomad_failures_total counter
prometheus_sd_nomad_failures_total 0
# HELP prometheus_sd_received_updates_total Total number of update events received from the SD providers.
# TYPE prometheus_sd_received_updates_total counter
prometheus_sd_received_updates_total{name="scrape"} 36897
# HELP prometheus_sd_updates_total Total number of update events sent to the SD consumers.
# TYPE prometheus_sd_updates_total counter
prometheus_sd_updates_total{name="scrape"} 34137
# HELP prometheus_target_interval_length_seconds Actual intervals between scrapes.
# TYPE prometheus_target_interval_length_seconds summary
prometheus_target_interval_length_seconds{interval="15s",quantile="0.01"} 14.99914058
prometheus_target_interval_length_seconds{interval="15s",quantile="0.05"} 14.999310634
prometheus_target_interval_length_seconds{interval="15s",quantile="0.5"} 15.000008779
prometheus_target_interval_length_seconds{interval="15s",quantile="0.9"} 15.000545764
prometheus_target_interval_length_seconds{interval="15s",quantile="0.99"} 15.000857257
prometheus_target_interval_length_seconds_sum{interval="15s"} 2.4210266343189236e+07
prometheus_target_interval_length_seconds_count{interval="15s"} 1.614017e+06
# HELP prometheus_target_metadata_cache_bytes The number of bytes that are currently used for storing metric metadata in the cache
# TYPE prometheus_target_metadata_cache_bytes gauge
prometheus_target_metadata_cache_bytes{scrape_job="cadvisor"} 6898
prometheus_target_metadata_cache_bytes{scrape_job="kube-state-metrics"} 1933
prometheus_target_metadata_cache_bytes{scrape_job="pods"} 34437
# HELP prometheus_target_metadata_cache_entries Total number of metric metadata entries in the cache
# TYPE prometheus_target_metadata_cache_entries gauge
prometheus_target_metadata_cache_entries{scrape_job="cadvisor"} 138
prometheus_target_metadata_cache_entries{scrape_job="kube-state-metrics"} 39
prometheus_target_metadata_cache_entries{scrape_job="pods"} 583
# HELP prometheus_target_scrape_pool_exceeded_label_limits_total Total number of times scrape pools hit the label limits, during sync or config reload.
# TYPE prometheus_target_scrape_pool_exceeded_label_limits_total counter
prometheus_target_scrape_pool_exceeded_label_limits_total 0
# HELP prometheus_target_scrape_pool_exceeded_target_limit_total Total number of times scrape pools hit the target limit, during sync or config reload.
# TYPE prometheus_target_scrape_pool_exceeded_target_limit_total counter
prometheus_target_scrape_pool_exceeded_target_limit_total 0
# HELP prometheus_target_scrape_pool_reloads_failed_total Total number of failed scrape pool reloads.
# TYPE prometheus_target_scrape_pool_reloads_failed_total counter
prometheus_target_scrape_pool_reloads_failed_total 0
# HELP prometheus_target_scrape_pool_reloads_total Total number of scrape pool reloads.
# TYPE prometheus_target_scrape_pool_reloads_total counter
prometheus_target_scrape_pool_reloads_total 0
# HELP prometheus_target_scrape_pool_sync_total Total number of syncs that were executed on a scrape pool.
# TYPE prometheus_target_scrape_pool_sync_total counter
prometheus_target_scrape_pool_sync_total{scrape_job="cadvisor"} 34137
prometheus_target_scrape_pool_sync_total{scrape_job="kube-state-metrics"} 34137
prometheus_target_scrape_pool_sync_total{scrape_job="pods"} 34137
# HELP prometheus_target_scrape_pool_target_limit Maximum number of targets allowed in this scrape pool.
# TYPE prometheus_target_scrape_pool_target_limit gauge
prometheus_target_scrape_pool_target_limit{scrape_job="cadvisor"} 0
prometheus_target_scrape_pool_target_limit{scrape_job="kube-state-metrics"} 0
prometheus_target_scrape_pool_target_limit{scrape_job="pods"} 0
# HELP prometheus_target_scrape_pool_targets Current number of targets in this scrape pool.
# TYPE prometheus_target_scrape_pool_targets gauge
prometheus_target_scrape_pool_targets{scrape_job="cadvisor"} 2
prometheus_target_scrape_pool_targets{scrape_job="kube-state-metrics"} 2
prometheus_target_scrape_pool_targets{scrape_job="pods"} 5
# HELP prometheus_target_scrape_pools_failed_total Total number of scrape pool creations that failed.
# TYPE prometheus_target_scrape_pools_failed_total counter
prometheus_target_scrape_pools_failed_total 0
# HELP prometheus_target_scrape_pools_total Total number of scrape pool creation attempts.
# TYPE prometheus_target_scrape_pools_total counter
prometheus_target_scrape_pools_total 3
# HELP prometheus_target_scrapes_cache_flush_forced_total How many times a scrape cache was flushed due to getting big while scrapes are failing.
# TYPE prometheus_target_scrapes_cache_flush_forced_total counter
prometheus_target_scrapes_cache_flush_forced_total 0
# HELP prometheus_target_scrapes_exceeded_body_size_limit_total Total number of scrapes that hit the body size limit
# TYPE prometheus_target_scrapes_exceeded_body_size_limit_total counter
prometheus_target_scrapes_exceeded_body_size_limit_total 0
# HELP prometheus_target_scrapes_exceeded_native_histogram_bucket_limit_total Total number of scrapes that hit the native histogram bucket limit and were rejected.
# TYPE prometheus_target_scrapes_exceeded_native_histogram_bucket_limit_total counter
prometheus_target_scrapes_exceeded_native_histogram_bucket_limit_total 0
# HELP prometheus_target_scrapes_exceeded_sample_limit_total Total number of scrapes that hit the sample limit and were rejected.
# TYPE prometheus_target_scrapes_exceeded_sample_limit_total counter
prometheus_target_scrapes_exceeded_sample_limit_total 0
# HELP prometheus_target_scrapes_exemplar_out_of_order_total Total number of exemplar rejected due to not being out of the expected order.
# TYPE prometheus_target_scrapes_exemplar_out_of_order_total counter
prometheus_target_scrapes_exemplar_out_of_order_total 0
# HELP prometheus_target_scrapes_sample_duplicate_timestamp_total Total number of samples rejected due to duplicate timestamps but different values.
# TYPE prometheus_target_scrapes_sample_duplicate_timestamp_total counter
prometheus_target_scrapes_sample_duplicate_timestamp_total 0
# HELP prometheus_target_scrapes_sample_out_of_bounds_total Total number of samples rejected due to timestamp falling outside of the time bounds.
# TYPE prometheus_target_scrapes_sample_out_of_bounds_total counter
prometheus_target_scrapes_sample_out_of_bounds_total 0
# HELP prometheus_target_scrapes_sample_out_of_order_total Total number of samples rejected due to not being out of the expected order.
# TYPE prometheus_target_scrapes_sample_out_of_order_total counter
prometheus_target_scrapes_sample_out_of_order_total 0
# HELP prometheus_target_sync_failed_total Total number of target sync failures.
# TYPE prometheus_target_sync_failed_total counter
prometheus_target_sync_failed_total{scrape_job="cadvisor"} 0
prometheus_target_sync_failed_total{scrape_job="kube-state-metrics"} 0
prometheus_target_sync_failed_total{scrape_job="pods"} 0
# HELP prometheus_target_sync_length_seconds Actual interval to sync the scrape pool.
# TYPE prometheus_target_sync_length_seconds summary
prometheus_target_sync_length_seconds{scrape_job="cadvisor",quantile="0.01"} 0.00016778
prometheus_target_sync_length_seconds{scrape_job="cadvisor",quantile="0.05"} 0.00016778
prometheus_target_sync_length_seconds{scrape_job="cadvisor",quantile="0.5"} 0.000201532
prometheus_target_sync_length_seconds{scrape_job="cadvisor",quantile="0.9"} 0.000217346
prometheus_target_sync_length_seconds{scrape_job="cadvisor",quantile="0.99"} 0.000217346
prometheus_target_sync_length_seconds_sum{scrape_job="cadvisor"} 9.36278804700008
prometheus_target_sync_length_seconds_count{scrape_job="cadvisor"} 34137
prometheus_target_sync_length_seconds{scrape_job="kube-state-metrics",quantile="0.01"} 0.000148145
prometheus_target_sync_length_seconds{scrape_job="kube-state-metrics",quantile="0.05"} 0.000148145
prometheus_target_sync_length_seconds{scrape_job="kube-state-metrics",quantile="0.5"} 0.000175667
prometheus_target_sync_length_seconds{scrape_job="kube-state-metrics",quantile="0.9"} 0.000188701
prometheus_target_sync_length_seconds{scrape_job="kube-state-metrics",quantile="0.99"} 0.000188701
prometheus_target_sync_length_seconds_sum{scrape_job="kube-state-metrics"} 6.007913164999995
prometheus_target_sync_length_seconds_count{scrape_job="kube-state-metrics"} 34137
prometheus_target_sync_length_seconds{scrape_job="pods",quantile="0.01"} 0.000867282
prometheus_target_sync_length_seconds{scrape_job="pods",quantile="0.05"} 0.000867282
prometheus_target_sync_length_seconds{scrape_job="pods",quantile="0.5"} 0.000913952
prometheus_target_sync_length_seconds{scrape_job="pods",quantile="0.9"} 0.001163668
prometheus_target_sync_length_seconds{scrape_job="pods",quantile="0.99"} 0.001163668
prometheus_target_sync_length_seconds_sum{scrape_job="pods"} 44.38431514700025
prometheus_target_sync_length_seconds_count{scrape_job="pods"} 34137
# HELP prometheus_template_text_expansion_failures_total The total number of template text expansion failures.
# TYPE prometheus_template_text_expansion_failures_total counter
prometheus_template_text_expansion_failures_total 0
# HELP prometheus_template_text_expansions_total The total number of template text expansions.
# TYPE prometheus_template_text_expansions_total counter
prometheus_template_text_expansions_total 0
# HELP prometheus_treecache_watcher_goroutines The current number of watcher goroutines.
# TYPE prometheus_treecache_watcher_goroutines gauge
prometheus_treecache_watcher_goroutines 0
# HELP prometheus_treecache_zookeeper_failures_total The total number of ZooKeeper failures.
# TYPE prometheus_treecache_zookeeper_failures_total counter
prometheus_treecache_zookeeper_failures_total 0
# HELP prometheus_tsdb_blocks_loaded Number of currently loaded data blocks
# TYPE prometheus_tsdb_blocks_loaded gauge
prometheus_tsdb_blocks_loaded 16
# HELP prometheus_tsdb_checkpoint_creations_failed_total Total number of checkpoint creations that failed.
# TYPE prometheus_tsdb_checkpoint_creations_failed_total counter
prometheus_tsdb_checkpoint_creations_failed_total 0
# HELP prometheus_tsdb_checkpoint_creations_total Total number of checkpoint creations attempted.
# TYPE prometheus_tsdb_checkpoint_creations_total counter
prometheus_tsdb_checkpoint_creations_total 187
# HELP prometheus_tsdb_checkpoint_deletions_failed_total Total number of checkpoint deletions that failed.
# TYPE prometheus_tsdb_checkpoint_deletions_failed_total counter
prometheus_tsdb_checkpoint_deletions_failed_total 0
# HELP prometheus_tsdb_checkpoint_deletions_total Total number of checkpoint deletions attempted.
# TYPE prometheus_tsdb_checkpoint_deletions_total counter
prometheus_tsdb_checkpoint_deletions_total 187
# HELP prometheus_tsdb_clean_start -1: lockfile is disabled. 0: a lockfile from a previous execution was replaced. 1: lockfile creation was clean
# TYPE prometheus_tsdb_clean_start gauge
prometheus_tsdb_clean_start -1
# HELP prometheus_tsdb_compaction_chunk_range_seconds Final time range of chunks on their first compaction
# TYPE prometheus_tsdb_compaction_chunk_range_seconds histogram
prometheus_tsdb_compaction_chunk_range_seconds_bucket{le="100"} 673
prometheus_tsdb_compaction_chunk_range_seconds_bucket{le="400"} 673
prometheus_tsdb_compaction_chunk_range_seconds_bucket{le="1600"} 673
prometheus_tsdb_compaction_chunk_range_seconds_bucket{le="6400"} 673
prometheus_tsdb_compaction_chunk_range_seconds_bucket{le="25600"} 952
prometheus_tsdb_compaction_chunk_range_seconds_bucket{le="102400"} 2954
prometheus_tsdb_compaction_chunk_range_seconds_bucket{le="409600"} 11240
prometheus_tsdb_compaction_chunk_range_seconds_bucket{le="1.6384e+06"} 34940
prometheus_tsdb_compaction_chunk_range_seconds_bucket{le="6.5536e+06"} 1.3837075e+07
prometheus_tsdb_compaction_chunk_range_seconds_bucket{le="2.62144e+07"} 1.3837077e+07
prometheus_tsdb_compaction_chunk_range_seconds_bucket{le="+Inf"} 1.3837077e+07
prometheus_tsdb_compaction_chunk_range_seconds_sum 2.9219718662064e+13
prometheus_tsdb_compaction_chunk_range_seconds_count 1.3837077e+07
# HELP prometheus_tsdb_compaction_chunk_samples Final number of samples on their first compaction
# TYPE prometheus_tsdb_compaction_chunk_samples histogram
prometheus_tsdb_compaction_chunk_samples_bucket{le="4"} 1813
prometheus_tsdb_compaction_chunk_samples_bucket{le="6"} 2625
prometheus_tsdb_compaction_chunk_samples_bucket{le="9"} 5359
prometheus_tsdb_compaction_chunk_samples_bucket{le="13.5"} 7578
prometheus_tsdb_compaction_chunk_samples_bucket{le="20.25"} 10695
prometheus_tsdb_compaction_chunk_samples_bucket{le="30.375"} 14153
prometheus_tsdb_compaction_chunk_samples_bucket{le="45.5625"} 20641
prometheus_tsdb_compaction_chunk_samples_bucket{le="68.34375"} 26828
prometheus_tsdb_compaction_chunk_samples_bucket{le="102.515625"} 37088
prometheus_tsdb_compaction_chunk_samples_bucket{le="153.7734375"} 1.3192758e+07
prometheus_tsdb_compaction_chunk_samples_bucket{le="230.66015625"} 1.3830353e+07
prometheus_tsdb_compaction_chunk_samples_bucket{le="345.990234375"} 1.3837077e+07
prometheus_tsdb_compaction_chunk_samples_bucket{le="+Inf"} 1.3837077e+07
prometheus_tsdb_compaction_chunk_samples_sum 1.852852608e+09
prometheus_tsdb_compaction_chunk_samples_count 1.3837077e+07
# HELP prometheus_tsdb_compaction_chunk_size_bytes Final size of chunks on their first compaction
# TYPE prometheus_tsdb_compaction_chunk_size_bytes histogram
prometheus_tsdb_compaction_chunk_size_bytes_bucket{le="32"} 5907
prometheus_tsdb_compaction_chunk_size_bytes_bucket{le="48"} 3.717611e+06
prometheus_tsdb_compaction_chunk_size_bytes_bucket{le="72"} 3.972949e+06
prometheus_tsdb_compaction_chunk_size_bytes_bucket{le="108"} 4.043949e+06
prometheus_tsdb_compaction_chunk_size_bytes_bucket{le="162"} 4.106797e+06
prometheus_tsdb_compaction_chunk_size_bytes_bucket{le="243"} 4.42655e+06
prometheus_tsdb_compaction_chunk_size_bytes_bucket{le="364.5"} 1.075848e+07
prometheus_tsdb_compaction_chunk_size_bytes_bucket{le="546.75"} 1.2225892e+07
prometheus_tsdb_compaction_chunk_size_bytes_bucket{le="820.125"} 1.3311939e+07
prometheus_tsdb_compaction_chunk_size_bytes_bucket{le="1230.1875"} 1.3795122e+07
prometheus_tsdb_compaction_chunk_size_bytes_bucket{le="1845.28125"} 1.3836776e+07
prometheus_tsdb_compaction_chunk_size_bytes_bucket{le="2767.921875"} 1.3837077e+07
prometheus_tsdb_compaction_chunk_size_bytes_bucket{le="+Inf"} 1.3837077e+07
prometheus_tsdb_compaction_chunk_size_bytes_sum 4.281044268e+09
prometheus_tsdb_compaction_chunk_size_bytes_count 1.3837077e+07
# HELP prometheus_tsdb_compaction_duration_seconds Duration of compaction runs
# TYPE prometheus_tsdb_compaction_duration_seconds histogram
prometheus_tsdb_compaction_duration_seconds_bucket{le="1"} 540
prometheus_tsdb_compaction_duration_seconds_bucket{le="2"} 540
prometheus_tsdb_compaction_duration_seconds_bucket{le="4"} 554
prometheus_tsdb_compaction_duration_seconds_bucket{le="8"} 559
prometheus_tsdb_compaction_duration_seconds_bucket{le="16"} 559
prometheus_tsdb_compaction_duration_seconds_bucket{le="32"} 561
prometheus_tsdb_compaction_duration_seconds_bucket{le="64"} 561
prometheus_tsdb_compaction_duration_seconds_bucket{le="128"} 561
prometheus_tsdb_compaction_duration_seconds_bucket{le="256"} 561
prometheus_tsdb_compaction_duration_seconds_bucket{le="512"} 561
prometheus_tsdb_compaction_duration_seconds_bucket{le="1024"} 561
prometheus_tsdb_compaction_duration_seconds_bucket{le="2048"} 561
prometheus_tsdb_compaction_duration_seconds_bucket{le="4096"} 561
prometheus_tsdb_compaction_duration_seconds_bucket{le="8192"} 561
prometheus_tsdb_compaction_duration_seconds_bucket{le="+Inf"} 561
prometheus_tsdb_compaction_duration_seconds_sum 272.2973793669999
prometheus_tsdb_compaction_duration_seconds_count 561
# HELP prometheus_tsdb_compaction_populating_block Set to 1 when a block is currently being written to the disk.
# TYPE prometheus_tsdb_compaction_populating_block gauge
prometheus_tsdb_compaction_populating_block 0
# HELP prometheus_tsdb_compactions_failed_total Total number of compactions that failed for the partition.
# TYPE prometheus_tsdb_compactions_failed_total counter
prometheus_tsdb_compactions_failed_total 0
# HELP prometheus_tsdb_compactions_skipped_total Total number of skipped compactions due to disabled auto compaction.
# TYPE prometheus_tsdb_compactions_skipped_total counter
prometheus_tsdb_compactions_skipped_total 0
# HELP prometheus_tsdb_compactions_total Total number of compactions that were executed for the partition.
# TYPE prometheus_tsdb_compactions_total counter
prometheus_tsdb_compactions_total 561
# HELP prometheus_tsdb_compactions_triggered_total Total number of triggered compactions for the partition.
# TYPE prometheus_tsdb_compactions_triggered_total counter
prometheus_tsdb_compactions_triggered_total 44842
# HELP prometheus_tsdb_data_replay_duration_seconds Time taken to replay the data on disk.
# TYPE prometheus_tsdb_data_replay_duration_seconds gauge
prometheus_tsdb_data_replay_duration_seconds 0.767674068
# HELP prometheus_tsdb_exemplar_exemplars_appended_total Total number of appended exemplars.
# TYPE prometheus_tsdb_exemplar_exemplars_appended_total counter
prometheus_tsdb_exemplar_exemplars_appended_total 0
# HELP prometheus_tsdb_exemplar_exemplars_in_storage Number of exemplars currently in circular storage.
# TYPE prometheus_tsdb_exemplar_exemplars_in_storage gauge
prometheus_tsdb_exemplar_exemplars_in_storage 0
# HELP prometheus_tsdb_exemplar_last_exemplars_timestamp_seconds The timestamp of the oldest exemplar stored in circular storage. Useful to check for what timerange the current exemplar buffer limit allows. This usually means the last timestampfor all exemplars for a typical setup. This is not true though if one of the series timestamp is in future compared to rest series.
# TYPE prometheus_tsdb_exemplar_last_exemplars_timestamp_seconds gauge
prometheus_tsdb_exemplar_last_exemplars_timestamp_seconds 0
# HELP prometheus_tsdb_exemplar_max_exemplars Total number of exemplars the exemplar storage can store, resizeable.
# TYPE prometheus_tsdb_exemplar_max_exemplars gauge
prometheus_tsdb_exemplar_max_exemplars 0
# HELP prometheus_tsdb_exemplar_out_of_order_exemplars_total Total number of out of order exemplar ingestion failed attempts.
# TYPE prometheus_tsdb_exemplar_out_of_order_exemplars_total counter
prometheus_tsdb_exemplar_out_of_order_exemplars_total 0
# HELP prometheus_tsdb_exemplar_series_with_exemplars_in_storage Number of series with exemplars currently in circular storage.
# TYPE prometheus_tsdb_exemplar_series_with_exemplars_in_storage gauge
prometheus_tsdb_exemplar_series_with_exemplars_in_storage 0
# HELP prometheus_tsdb_head_active_appenders Number of currently active appender transactions
# TYPE prometheus_tsdb_head_active_appenders gauge
prometheus_tsdb_head_active_appenders 0
# HELP prometheus_tsdb_head_chunks Total number of chunks in the head block.
# TYPE prometheus_tsdb_head_chunks gauge
prometheus_tsdb_head_chunks 47276
# HELP prometheus_tsdb_head_chunks_created_total Total number of chunks created in the head
# TYPE prometheus_tsdb_head_chunks_created_total counter
prometheus_tsdb_head_chunks_created_total 1.3884353e+07
# HELP prometheus_tsdb_head_chunks_removed_total Total number of chunks removed in the head
# TYPE prometheus_tsdb_head_chunks_removed_total counter
prometheus_tsdb_head_chunks_removed_total 1.3837077e+07
# HELP prometheus_tsdb_head_chunks_storage_size_bytes Size of the chunks_head directory.
# TYPE prometheus_tsdb_head_chunks_storage_size_bytes gauge
prometheus_tsdb_head_chunks_storage_size_bytes 2.0828256e+07
# HELP prometheus_tsdb_head_gc_duration_seconds Runtime of garbage collection in the head block.
# TYPE prometheus_tsdb_head_gc_duration_seconds summary
prometheus_tsdb_head_gc_duration_seconds_sum 3.114924039999997
prometheus_tsdb_head_gc_duration_seconds_count 373
# HELP prometheus_tsdb_head_max_time Maximum timestamp of the head block. The unit is decided by the library consumer.
# TYPE prometheus_tsdb_head_max_time gauge
prometheus_tsdb_head_max_time 1.727807345546e+12
# HELP prometheus_tsdb_head_max_time_seconds Maximum timestamp of the head block.
# TYPE prometheus_tsdb_head_max_time_seconds gauge
prometheus_tsdb_head_max_time_seconds 1.727807345e+09
# HELP prometheus_tsdb_head_min_time Minimum time bound of the head block. The unit is decided by the library consumer.
# TYPE prometheus_tsdb_head_min_time gauge
prometheus_tsdb_head_min_time 1.727798400141e+12
# HELP prometheus_tsdb_head_min_time_seconds Minimum time bound of the head block.
# TYPE prometheus_tsdb_head_min_time_seconds gauge
prometheus_tsdb_head_min_time_seconds 1.7277984e+09
# HELP prometheus_tsdb_head_out_of_order_samples_appended_total Total number of appended out of order samples.
# TYPE prometheus_tsdb_head_out_of_order_samples_appended_total counter
prometheus_tsdb_head_out_of_order_samples_appended_total 0
# HELP prometheus_tsdb_head_samples_appended_total Total number of appended samples.
# TYPE prometheus_tsdb_head_samples_appended_total counter
prometheus_tsdb_head_samples_appended_total{type="float"} 1.856200861e+09
prometheus_tsdb_head_samples_appended_total{type="histogram"} 0
# HELP prometheus_tsdb_head_series Total number of series in the head block.
# TYPE prometheus_tsdb_head_series gauge
prometheus_tsdb_head_series 10789
# HELP prometheus_tsdb_head_series_created_total Total number of series created in the head
# TYPE prometheus_tsdb_head_series_created_total counter
prometheus_tsdb_head_series_created_total 42838
# HELP prometheus_tsdb_head_series_not_found_total Total number of requests for series that were not found.
# TYPE prometheus_tsdb_head_series_not_found_total counter
prometheus_tsdb_head_series_not_found_total 0
# HELP prometheus_tsdb_head_series_removed_total Total number of series removed in the head
# TYPE prometheus_tsdb_head_series_removed_total counter
prometheus_tsdb_head_series_removed_total 32049
# HELP prometheus_tsdb_head_truncations_failed_total Total number of head truncations that failed.
# TYPE prometheus_tsdb_head_truncations_failed_total counter
prometheus_tsdb_head_truncations_failed_total 0
# HELP prometheus_tsdb_head_truncations_total Total number of head truncations attempted.
# TYPE prometheus_tsdb_head_truncations_total counter
prometheus_tsdb_head_truncations_total 373
# HELP prometheus_tsdb_isolation_high_watermark The highest TSDB append ID that has been given out.
# TYPE prometheus_tsdb_isolation_high_watermark gauge
prometheus_tsdb_isolation_high_watermark 1.614044e+06
# HELP prometheus_tsdb_isolation_low_watermark The lowest TSDB append ID that is still referenced.
# TYPE prometheus_tsdb_isolation_low_watermark gauge
prometheus_tsdb_isolation_low_watermark 1.614044e+06
# HELP prometheus_tsdb_lowest_timestamp Lowest timestamp value stored in the database. The unit is decided by the library consumer.
# TYPE prometheus_tsdb_lowest_timestamp gauge
prometheus_tsdb_lowest_timestamp 1.711547243455e+12
# HELP prometheus_tsdb_lowest_timestamp_seconds Lowest timestamp value stored in the database.
# TYPE prometheus_tsdb_lowest_timestamp_seconds gauge
prometheus_tsdb_lowest_timestamp_seconds 1.711547243e+09
# HELP prometheus_tsdb_mmap_chunk_corruptions_total Total number of memory-mapped chunk corruptions.
# TYPE prometheus_tsdb_mmap_chunk_corruptions_total counter
prometheus_tsdb_mmap_chunk_corruptions_total 0
# HELP prometheus_tsdb_out_of_bound_samples_total Total number of out of bound samples ingestion failed attempts with out of order support disabled.
# TYPE prometheus_tsdb_out_of_bound_samples_total counter
prometheus_tsdb_out_of_bound_samples_total{type="float"} 0
# HELP prometheus_tsdb_out_of_order_samples_total Total number of out of order samples ingestion failed attempts due to out of order being disabled.
# TYPE prometheus_tsdb_out_of_order_samples_total counter
prometheus_tsdb_out_of_order_samples_total{type="float"} 0
prometheus_tsdb_out_of_order_samples_total{type="histogram"} 0
# HELP prometheus_tsdb_reloads_failures_total Number of times the database failed to reloadBlocks block data from disk.
# TYPE prometheus_tsdb_reloads_failures_total counter
prometheus_tsdb_reloads_failures_total 0
# HELP prometheus_tsdb_reloads_total Number of times the database reloaded block data from disk.
# TYPE prometheus_tsdb_reloads_total counter
prometheus_tsdb_reloads_total 45030
# HELP prometheus_tsdb_retention_limit_bytes Max number of bytes to be retained in the tsdb blocks, configured 0 means disabled
# TYPE prometheus_tsdb_retention_limit_bytes gauge
prometheus_tsdb_retention_limit_bytes 5.36870912e+11
# HELP prometheus_tsdb_size_retentions_total The number of times that blocks were deleted because the maximum number of bytes was exceeded.
# TYPE prometheus_tsdb_size_retentions_total counter
prometheus_tsdb_size_retentions_total 0
# HELP prometheus_tsdb_snapshot_replay_error_total Total number snapshot replays that failed.
# TYPE prometheus_tsdb_snapshot_replay_error_total counter
prometheus_tsdb_snapshot_replay_error_total 0
# HELP prometheus_tsdb_storage_blocks_bytes The number of bytes that are currently used for local storage by all blocks.
# TYPE prometheus_tsdb_storage_blocks_bytes gauge
prometheus_tsdb_storage_blocks_bytes 2.7078242758e+10
# HELP prometheus_tsdb_symbol_table_size_bytes Size of symbol table in memory for loaded blocks
# TYPE prometheus_tsdb_symbol_table_size_bytes gauge
prometheus_tsdb_symbol_table_size_bytes 6624
# HELP prometheus_tsdb_time_retentions_total The number of times that blocks were deleted because the maximum time limit was exceeded.
# TYPE prometheus_tsdb_time_retentions_total counter
prometheus_tsdb_time_retentions_total 0
# HELP prometheus_tsdb_tombstone_cleanup_seconds The time taken to recompact blocks to remove tombstones.
# TYPE prometheus_tsdb_tombstone_cleanup_seconds histogram
prometheus_tsdb_tombstone_cleanup_seconds_bucket{le="0.005"} 0
prometheus_tsdb_tombstone_cleanup_seconds_bucket{le="0.01"} 0
prometheus_tsdb_tombstone_cleanup_seconds_bucket{le="0.025"} 0
prometheus_tsdb_tombstone_cleanup_seconds_bucket{le="0.05"} 0
prometheus_tsdb_tombstone_cleanup_seconds_bucket{le="0.1"} 0
prometheus_tsdb_tombstone_cleanup_seconds_bucket{le="0.25"} 0
prometheus_tsdb_tombstone_cleanup_seconds_bucket{le="0.5"} 0
prometheus_tsdb_tombstone_cleanup_seconds_bucket{le="1"} 0
prometheus_tsdb_tombstone_cleanup_seconds_bucket{le="2.5"} 0
prometheus_tsdb_tombstone_cleanup_seconds_bucket{le="5"} 0
prometheus_tsdb_tombstone_cleanup_seconds_bucket{le="10"} 0
prometheus_tsdb_tombstone_cleanup_seconds_bucket{le="+Inf"} 0
prometheus_tsdb_tombstone_cleanup_seconds_sum 0
prometheus_tsdb_tombstone_cleanup_seconds_count 0
# HELP prometheus_tsdb_too_old_samples_total Total number of out of order samples ingestion failed attempts with out of support enabled, but sample outside of time window.
# TYPE prometheus_tsdb_too_old_samples_total counter
prometheus_tsdb_too_old_samples_total{type="float"} 0
# HELP prometheus_tsdb_vertical_compactions_total Total number of compactions done on overlapping blocks.
# TYPE prometheus_tsdb_vertical_compactions_total counter
prometheus_tsdb_vertical_compactions_total 0
# HELP prometheus_tsdb_wal_completed_pages_total Total number of completed pages.
# TYPE prometheus_tsdb_wal_completed_pages_total counter
prometheus_tsdb_wal_completed_pages_total 397233
# HELP prometheus_tsdb_wal_corruptions_total Total number of WAL corruptions.
# TYPE prometheus_tsdb_wal_corruptions_total counter
prometheus_tsdb_wal_corruptions_total 0
# HELP prometheus_tsdb_wal_fsync_duration_seconds Duration of write log fsync.
# TYPE prometheus_tsdb_wal_fsync_duration_seconds summary
prometheus_tsdb_wal_fsync_duration_seconds{quantile="0.5"} NaN
prometheus_tsdb_wal_fsync_duration_seconds{quantile="0.9"} NaN
prometheus_tsdb_wal_fsync_duration_seconds{quantile="0.99"} NaN
prometheus_tsdb_wal_fsync_duration_seconds_sum 0.805116427
prometheus_tsdb_wal_fsync_duration_seconds_count 373
# HELP prometheus_tsdb_wal_page_flushes_total Total number of page flushes.
# TYPE prometheus_tsdb_wal_page_flushes_total counter
prometheus_tsdb_wal_page_flushes_total 2.011145e+06
# HELP prometheus_tsdb_wal_segment_current Write log segment index that TSDB is currently writing to.
# TYPE prometheus_tsdb_wal_segment_current gauge
prometheus_tsdb_wal_segment_current 2277
# HELP prometheus_tsdb_wal_storage_size_bytes Size of the write log directory.
# TYPE prometheus_tsdb_wal_storage_size_bytes gauge
prometheus_tsdb_wal_storage_size_bytes 9.6264943e+07
# HELP prometheus_tsdb_wal_truncate_duration_seconds Duration of WAL truncation.
# TYPE prometheus_tsdb_wal_truncate_duration_seconds summary
prometheus_tsdb_wal_truncate_duration_seconds_sum 69.80804534300002
prometheus_tsdb_wal_truncate_duration_seconds_count 187
# HELP prometheus_tsdb_wal_truncations_failed_total Total number of write log truncations that failed.
# TYPE prometheus_tsdb_wal_truncations_failed_total counter
prometheus_tsdb_wal_truncations_failed_total 0
# HELP prometheus_tsdb_wal_truncations_total Total number of write log truncations attempted.
# TYPE prometheus_tsdb_wal_truncations_total counter
prometheus_tsdb_wal_truncations_total 187
# HELP prometheus_tsdb_wal_writes_failed_total Total number of write log writes that failed.
# TYPE prometheus_tsdb_wal_writes_failed_total counter
prometheus_tsdb_wal_writes_failed_total 0
# HELP prometheus_web_federation_errors_total Total number of errors that occurred while sending federation responses.
# TYPE prometheus_web_federation_errors_total counter
prometheus_web_federation_errors_total 0
# HELP prometheus_web_federation_warnings_total Total number of warnings that occurred while sending federation responses.
# TYPE prometheus_web_federation_warnings_total counter
prometheus_web_federation_warnings_total 0
# HELP promhttp_metric_handler_requests_in_flight Current number of scrapes being served.
# TYPE promhttp_metric_handler_requests_in_flight gauge
promhttp_metric_handler_requests_in_flight 1
# HELP promhttp_metric_handler_requests_total Total number of scrapes by HTTP status code.
# TYPE promhttp_metric_handler_requests_total counter
promhttp_metric_handler_requests_total{code="200"} 179357
promhttp_metric_handler_requests_total{code="500"} 0
promhttp_metric_handler_requests_total{code="503"} 0
`

func TestComputeAvalancheFlags(t *testing.T) {
	for _, tc := range []struct {
		testName               string
		avalancheFlagsForTotal int
		seriesCount            int
		statistics             map[dto.MetricType]stats
		total                  stats
		expectedSum            int
		expectedFlags          []string
	}{
		{
			testName:               "samplePromInput",
			avalancheFlagsForTotal: 1000,
			seriesCount:            10,
			statistics: map[dto.MetricType]stats{
				dto.MetricType_COUNTER:   {families: 104, series: 166, adjustedSeries: 166},
				dto.MetricType_GAUGE:     {families: 77, series: 94, adjustedSeries: 94},
				dto.MetricType_HISTOGRAM: {families: 11, series: 17, adjustedSeries: 224, buckets: 190},
				dto.MetricType_SUMMARY:   {families: 15, series: 27, adjustedSeries: 114, objectives: 60},
			},
			total: stats{
				families: 207, series: 304, adjustedSeries: 598,
			},
			expectedSum: 84,
			expectedFlags: []string{
				"--gauge-metric-count=15",
				"--counter-metric-count=27",
				"--histogram-metric-count=2",
				"--histogram-metric-bucket-count=10",
				"--native-histogram-metric-count=0",
				"--summary-metric-count=4",
				"--summary-metric-objective-count=2",
				"--series-count=10",
				"--value-interval=300",
				"--series-interval=3600",
				"--metric-interval=0",
			},
		},
		{
			testName:               "noInput",
			avalancheFlagsForTotal: 1000,
			seriesCount:            10,
			statistics:             map[dto.MetricType]stats{},
			total:                  stats{},
			expectedSum:            0,
			expectedFlags: []string{
				"--gauge-metric-count=0",
				"--counter-metric-count=0",
				"--histogram-metric-count=0",
				"--native-histogram-metric-count=0",
				"--summary-metric-count=0",
				"--series-count=10",
				"--value-interval=300",
				"--series-interval=3600",
				"--metric-interval=0",
			},
		},
	} {
		t.Run(tc.testName, func(t *testing.T) {
			avalancheFlags, adjustedSum := computeAvalancheFlags(tc.avalancheFlagsForTotal, tc.seriesCount, tc.total, tc.statistics)
			assert.Equal(t, tc.expectedSum, adjustedSum)
			assert.Equal(t, tc.expectedFlags, avalancheFlags)
		})
	}
}
