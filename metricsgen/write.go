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

package metricsgen

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/prometheus/client_golang/exp/api/remote"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/prompb"
	"gopkg.in/alecthomas/kingpin.v2"
)

// ConfigWrite for the remote write requests.
type ConfigWrite struct {
	URL             *url.URL
	RequestInterval time.Duration
	BatchSize,
	RequestCount int
	UpdateNotify      chan struct{}
	PprofURLs         []*url.URL
	Tenant            string
	TLSClientInsecure bool
	TLSClientCertFile string
	TLSClientKeyFile  string
	TLSCACertFile     string
	TenantHeader      string
	OutOfOrder        bool
	Concurrency       int
	WriteV2           bool
}

func NewWriteConfigFromFlags(flagReg func(name, help string) *kingpin.FlagClause) *ConfigWrite {
	cfg := &ConfigWrite{}
	flagReg("remote-url", "URL to send samples via remote_write API. By default, path is set to api/v1/write").
		URLVar(&cfg.URL)
	flagReg("remote-concurrency-limit", "how many concurrent writes can happen at any given time").Default("20").
		IntVar(&cfg.Concurrency)
	flagReg("remote-batch-size", "how many samples to send with each remote_write API request.").Default("2000").
		IntVar(&cfg.BatchSize)
	flagReg("remote-requests-count", "How many requests to send in total to the remote_write API. Set to -1 to run indefinitely.").Default("100").
		IntVar(&cfg.RequestCount)
	flagReg("remote-write-interval", "delay between each remote write request.").Default("100ms").
		DurationVar(&cfg.RequestInterval)
	flagReg("remote-tenant", "Tenant ID to include in remote_write send").Default("0").
		StringVar(&cfg.Tenant)
	flagReg("tls-client-insecure", "Skip certificate check on tls connection").Default("false").
		BoolVar(&cfg.TLSClientInsecure)
	flagReg("tls-client-cert-file", "Client certificate file for TLS connections.").Default("").
		StringVar(&cfg.TLSClientCertFile)
	flagReg("tls-client-key-file", "Client key file for TLS connections.").Default("").
		StringVar(&cfg.TLSClientKeyFile)
	flagReg("tls-ca-cert-file", "CA certificate file to verify server certificate.").Default("").
		StringVar(&cfg.TLSCACertFile)
	flagReg("remote-tenant-header", "Tenant ID to include in remote_write send. The default, is the default tenant header expected by Cortex.").Default("X-Scope-OrgID").
		StringVar(&cfg.TenantHeader)
	// TODO(bwplotka): Make this a non-bool flag (e.g. out-of-order-min-time).
	flagReg("remote-out-of-order", "Enable out-of-order timestamps in remote write requests").Default("true").
		BoolVar(&cfg.OutOfOrder)
	flagReg("remote-write-v2", "Send remote write v2 format requests.").Default("false").
		BoolVar(&cfg.WriteV2)

	return cfg
}

func (c *ConfigWrite) Validate() error {
	if c.URL != nil {
		if c.URL.Host == "" || c.URL.Scheme == "" {
			return fmt.Errorf("remote host and scheme can't be empty")
		}

		if c.BatchSize <= 0 {
			return fmt.Errorf("remote send batch sizemust be greater than 0, got %v", c.BatchSize)
		}

		if c.RequestInterval <= 0 {
			return fmt.Errorf("--remote-write-interval must be greater than 0, got %v", c.RequestInterval)
		}
	}
	if (c.TLSClientCertFile == "") != (c.TLSClientKeyFile == "") {
		return fmt.Errorf("--tls-client-cert-file and --tls-client-key-file must both be set or both be unset")
	}
	return nil
}

func buildTLSConfig(insecure bool, certFile, keyFile, caFile string) (*tls.Config, error) {
	if (certFile == "") != (keyFile == "") {
		return nil, fmt.Errorf("--tls-client-cert-file and --tls-client-key-file must both be set or both be unset")
	}

	if insecure && caFile != "" {
		log.Printf("warning: --tls-ca-cert-file is ignored when --tls-client-insecure is set")
	}

	cfg := &tls.Config{InsecureSkipVerify: insecure} //nolint:gosec

	if certFile != "" {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS client cert/key: %w", err)
		}
		cfg.Certificates = []tls.Certificate{cert}
	}

	if caFile != "" {
		caCert, err := os.ReadFile(caFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA cert file: %w", err)
		}
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA cert file %q: no valid PEM certificates found", caFile)
		}
		cfg.RootCAs = caCertPool
	}

	return cfg, nil
}

// Writer for remote write requests.
type Writer struct {
	logger    *slog.Logger
	timeout   time.Duration
	config    *ConfigWrite
	gatherer  prometheus.Gatherer
	remoteAPI *remote.API
}

// RunRemoteWriting initializes a http client and starts a Writer for remote writing metrics to a prometheus compatible remote endpoint.
func RunRemoteWriting(ctx context.Context, logger *slog.Logger, cfg *ConfigWrite, gatherer prometheus.Gatherer) error {
	tlsCfg, err := buildTLSConfig(cfg.TLSClientInsecure, cfg.TLSClientCertFile, cfg.TLSClientKeyFile, cfg.TLSCACertFile)
	if err != nil {
		return err
	}

	var rt http.RoundTripper = &http.Transport{
		TLSClientConfig: tlsCfg,
	}
	rt = &tenantRoundTripper{tenant: cfg.Tenant, tenantHeader: cfg.TenantHeader, rt: rt}
	rt = &userAgentRoundTripper{userAgent: "avalanche", rt: rt}
	httpClient := &http.Client{Transport: rt}

	remoteAPI, err := newRemoteAPI(cfg, logger, httpClient)
	if err != nil {
		return err
	}

	writer := Writer{
		logger:    logger,
		timeout:   time.Minute,
		config:    cfg,
		gatherer:  gatherer,
		remoteAPI: remoteAPI,
	}

	if cfg.WriteV2 {
		return writer.writeV2(ctx)
	}

	return writer.write(ctx)
}

// newRemoteAPI builds the remote write client. If the user-supplied URL has
// a path (other than a bare "/"), it is kept, subject to the client's path
// cleaning (trailing and duplicate slashes are dropped). Host-only URLs and
// a bare "/" get the client's default path, api/v1/write; posting samples
// to "/" is almost never intended.
// See https://github.com/prometheus-community/avalanche/issues/173.
func newRemoteAPI(cfg *ConfigWrite, logger *slog.Logger, httpClient *http.Client) (*remote.API, error) {
	opts := []remote.APIOption{
		remote.WithAPIHTTPClient(httpClient),
		remote.WithAPILogger(logger.With("component", "remote_write_api")),
	}
	if cfg.URL.Path != "" && cfg.URL.Path != "/" {
		// remote.NewAPI path.Join-s its path option onto the URL's path;
		// an empty path option keeps the URL's path as-is.
		opts = append(opts, remote.WithAPIPath(""))
	}
	return remote.NewAPI(cfg.URL.String(), opts...)
}

// Add the tenant ID header
func (rt *tenantRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req = cloneRequest(req)
	req.Header.Set(rt.tenantHeader, rt.tenant)
	return rt.rt.RoundTrip(req)
}

type tenantRoundTripper struct {
	tenant       string
	tenantHeader string
	rt           http.RoundTripper
}

type userAgentRoundTripper struct {
	userAgent string
	rt        http.RoundTripper
}

func (rt *userAgentRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req = cloneRequest(req)
	req.Header.Set("User-Agent", rt.userAgent)
	return rt.rt.RoundTrip(req)
}

// cloneRequest returns a clone of the provided *http.Request.
// The clone is a shallow copy of the struct and its Header map.
func cloneRequest(r *http.Request) *http.Request {
	// Shallow copy of the struct.
	r2 := new(http.Request)
	*r2 = *r
	// Deep copy of the Header.
	r2.Header = make(http.Header)
	for k, s := range r.Header {
		r2.Header[k] = s
	}
	return r2
}

func (w *Writer) write(ctx context.Context) error {
	select {
	// Wait for update first as write and collector.Run runs simultaneously.
	case <-w.config.UpdateNotify:
	case <-ctx.Done():
		return ctx.Err()
	}

	tss, err := collectMetrics(w.gatherer, w.config.OutOfOrder)
	if err != nil {
		return err
	}

	var (
		totalTime       time.Duration
		totalSamplesExp = len(tss) * w.config.RequestCount
		totalSamplesAct int
		mtx             sync.Mutex
		wgMetrics       sync.WaitGroup
		merr            []error
	)

	shouldRunForever := w.config.RequestCount == -1
	if shouldRunForever {
		log.Printf("Sending: %v timeseries infinitely, %v timeseries per request, %v delay between requests\n",
			len(tss), w.config.BatchSize, w.config.RequestInterval)
	} else {
		log.Printf("Sending: %v timeseries, %v times, %v timeseries per request, %v delay between requests\n",
			len(tss), w.config.RequestCount, w.config.BatchSize, w.config.RequestInterval)
	}

	ticker := time.NewTicker(w.config.RequestInterval)
	defer ticker.Stop()

	concurrencyLimitCh := make(chan struct{}, w.config.Concurrency)

	for i := 0; ; {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if !shouldRunForever {
			if i >= w.config.RequestCount {
				break
			}
			i++
		}

		<-ticker.C
		select {
		case <-w.config.UpdateNotify:
			log.Println("updating remote write metrics")
			tss, err = collectMetrics(w.gatherer, w.config.OutOfOrder)
			if err != nil {
				merr = append(merr, err)
			}
		default:
			tss = updateTimetamps(tss)
		}

		start := time.Now()
		for i := 0; i < len(tss); i += w.config.BatchSize {
			wgMetrics.Add(1)
			concurrencyLimitCh <- struct{}{}
			go func(i int) {
				defer func() {
					<-concurrencyLimitCh
				}()
				defer wgMetrics.Done()
				end := i + w.config.BatchSize
				if end > len(tss) {
					end = len(tss)
				}
				req := &prompb.WriteRequest{
					Timeseries: tss[i:end],
				}

				if _, err := w.remoteAPI.Write(ctx, remote.WriteV1MessageType, req); err != nil {
					merr = append(merr, err)
					w.logger.Error("error writing metrics", "error", err)
					return
				}

				mtx.Lock()
				totalSamplesAct += len(tss[i:end])
				mtx.Unlock()
			}(i)
		}
		wgMetrics.Wait()
		totalTime += time.Since(start)
		if countErrors(merr) > 20 {
			merr = append(merr, errors.New("too many errors"))
			return errors.Join(merr...)
		}
	}
	if w.config.RequestCount*len(tss) != totalSamplesAct {
		merr = append(merr, fmt.Errorf("total samples mismatch, exp:%v , act:%v", totalSamplesExp, totalSamplesAct))
	}
	w.logger.Info("metrics summary",
		"total_time", totalTime.Round(time.Second),
		"total_samples", totalSamplesAct,
		"samples_per_sec", int(float64(totalSamplesAct)/totalTime.Seconds()),
		"errors", countErrors(merr))
	return errors.Join(merr...)
}

func updateTimetamps(tss []prompb.TimeSeries) []prompb.TimeSeries {
	t := int64(model.Now())
	for i := range tss {
		tss[i].Samples[0].Timestamp = t
	}
	return tss
}

func collectMetrics(gatherer prometheus.Gatherer, outOfOrder bool) ([]prompb.TimeSeries, error) {
	metricFamilies, err := gatherer.Gather()
	if err != nil {
		return nil, err
	}
	tss := ToTimeSeriesSlice(metricFamilies)
	if outOfOrder {
		tss = shuffleTimestamps(tss)
	}
	return tss, nil
}

func shuffleTimestamps(tss []prompb.TimeSeries) []prompb.TimeSeries {
	now := time.Now().UnixMilli()
	offsets := []int64{0, -60 * 1000, -5 * 60 * 1000}

	for i := range tss {
		offset := offsets[i%len(offsets)]
		tss[i].Samples[0].Timestamp = now + offset
	}
	return tss
}

// ToTimeSeriesSlice converts a slice of metricFamilies containing samples into a slice of TimeSeries
func ToTimeSeriesSlice(metricFamilies []*dto.MetricFamily) []prompb.TimeSeries {
	tss := make([]prompb.TimeSeries, 0, len(metricFamilies)*10)
	timestamp := int64(model.Now()) // Not using metric.TimestampMs because it is (always?) nil. Is this right?

	skippedSamples := 0
	for _, metricFamily := range metricFamilies {
		for _, metric := range metricFamily.Metric {
			labels := prompbLabels(*metricFamily.Name, metric.Label)
			ts := prompb.TimeSeries{
				Labels: labels,
			}
			switch *metricFamily.Type {
			case dto.MetricType_COUNTER:
				ts.Samples = []prompb.Sample{{
					Value:     *metric.Counter.Value,
					Timestamp: timestamp,
				}}
				tss = append(tss, ts)
			case dto.MetricType_GAUGE:
				ts.Samples = []prompb.Sample{{
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
	return tss
}

func prompbLabels(name string, label []*dto.LabelPair) []prompb.Label {
	ret := make([]prompb.Label, 0, len(label)+1)
	ret = append(ret, prompb.Label{
		Name:  model.MetricNameLabel,
		Value: name,
	})
	for _, pair := range label {
		ret = append(ret, prompb.Label{
			Name:  *pair.Name,
			Value: *pair.Value,
		})
	}
	sort.Slice(ret, func(i, j int) bool {
		return ret[i].Name < ret[j].Name
	})
	return ret
}
