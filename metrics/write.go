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

package metrics

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sort"
	"sync"
	"time"

	"github.com/prometheus-community/avalanche/pkg/errors"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/prompb"
	"gopkg.in/alecthomas/kingpin.v2"
)

const maxErrMsgLen = 256

var userAgent = "avalanche"

// ConfigWrite for the remote write requests.
type ConfigWrite struct {
	URL             *url.URL
	RequestInterval time.Duration
	BatchSize,
	RequestCount int
	UpdateNotify    chan struct{}
	PprofURLs       []*url.URL
	Tenant          string
	TLSClientConfig tls.Config
	TenantHeader    string
	OutOfOrder      bool
	Concurrency     int
}

func NewWriteConfigFromFlags(flagReg func(name, help string) *kingpin.FlagClause) *ConfigWrite {
	cfg := &ConfigWrite{}
	flagReg("remote-url", "URL to send samples via remote_write API.").
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
		BoolVar(&cfg.TLSClientConfig.InsecureSkipVerify)
	flagReg("remote-tenant-header", "Tenant ID to include in remote_write send. The default, is the default tenant header expected by Cortex.").Default("X-Scope-OrgID").
		StringVar(&cfg.TenantHeader)
	// TODO(bwplotka): Make this a non-bool flag (e.g. out-of-order-min-time).
	flagReg("remote-out-of-order", "Enable out-of-order timestamps in remote write requests").Default("true").
		BoolVar(&cfg.OutOfOrder)

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
	return nil
}

// Client for the remote write requests.
type Client struct {
	client   *http.Client
	timeout  time.Duration
	config   *ConfigWrite
	gatherer prometheus.Gatherer
}

// SendRemoteWrite initializes a http client and
// sends metrics to a prometheus compatible remote endpoint.
func (c *ConfigWrite) SendRemoteWrite(ctx context.Context, gatherer prometheus.Gatherer) error {
	var rt http.RoundTripper = &http.Transport{
		TLSClientConfig: &c.TLSClientConfig,
	}
	rt = &tenantRoundTripper{tenant: c.Tenant, tenantHeader: c.TenantHeader, rt: rt}
	httpClient := &http.Client{Transport: rt}

	client := Client{
		client:   httpClient,
		timeout:  time.Minute,
		config:   c,
		gatherer: gatherer,
	}
	return client.write(ctx)
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

func (c *Client) write(ctx context.Context) error {
	select {
	// Wait for update first as write and collector.Run runs simultaneously.
	case <-c.config.UpdateNotify:
	case <-ctx.Done():
		return ctx.Err()
	}

	tss, err := collectMetrics(c.gatherer, c.config.OutOfOrder)
	if err != nil {
		return err
	}

	var (
		totalTime       time.Duration
		totalSamplesExp = len(tss) * c.config.RequestCount
		totalSamplesAct int
		mtx             sync.Mutex
		wgMetrics       sync.WaitGroup
		merr            = &errors.MultiError{}
	)

	shouldRunForever := c.config.RequestCount == -1
	if shouldRunForever {
		log.Printf("Sending: %v timeseries infinitely, %v timeseries per request, %v delay between requests\n",
			len(tss), c.config.BatchSize, c.config.RequestInterval)
	} else {
		log.Printf("Sending: %v timeseries, %v times, %v timeseries per request, %v delay between requests\n",
			len(tss), c.config.RequestCount, c.config.BatchSize, c.config.RequestInterval)
	}

	ticker := time.NewTicker(c.config.RequestInterval)
	defer ticker.Stop()

	concurrencyLimitCh := make(chan struct{}, c.config.Concurrency)

	for i := 0; ; {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if !shouldRunForever {
			if i >= c.config.RequestCount {
				break
			}
			i++
		}

		<-ticker.C
		select {
		case <-c.config.UpdateNotify:
			log.Println("updating remote write metrics")
			tss, err = collectMetrics(c.gatherer, c.config.OutOfOrder)
			if err != nil {
				merr.Add(err)
			}
		default:
			tss = updateTimetamps(tss)
		}

		start := time.Now()
		for i := 0; i < len(tss); i += c.config.BatchSize {
			wgMetrics.Add(1)
			concurrencyLimitCh <- struct{}{}
			go func(i int) {
				defer func() {
					<-concurrencyLimitCh
				}()
				defer wgMetrics.Done()
				end := i + c.config.BatchSize
				if end > len(tss) {
					end = len(tss)
				}
				req := &prompb.WriteRequest{
					Timeseries: tss[i:end],
				}
				if err := c.Store(context.TODO(), req); err != nil {
					merr.Add(err)
					return
				}
				mtx.Lock()
				totalSamplesAct += len(tss[i:end])
				mtx.Unlock()
			}(i)
		}
		wgMetrics.Wait()
		totalTime += time.Since(start)
		if merr.Count() > 20 {
			merr.Add(fmt.Errorf("too many errors"))
			return merr.Err()
		}
	}
	if c.config.RequestCount*len(tss) != totalSamplesAct {
		merr.Add(fmt.Errorf("total samples mismatch, exp:%v , act:%v", totalSamplesExp, totalSamplesAct))
	}
	log.Printf("Total request time: %v ; Total samples: %v; Samples/sec: %v\n", totalTime.Round(time.Second), totalSamplesAct, int(float64(totalSamplesAct)/totalTime.Seconds()))
	return merr.Err()
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

// Store sends a batch of samples to the HTTP endpoint.
func (c *Client) Store(ctx context.Context, req *prompb.WriteRequest) error {
	data, err := proto.Marshal(req)
	if err != nil {
		return err
	}

	compressed := snappy.Encode(nil, data)
	httpReq, err := http.NewRequest("POST", c.config.URL.String(), bytes.NewReader(compressed))
	if err != nil {
		// Errors from NewRequest are from unparseable URLs, so are not
		// recoverable.
		return err
	}
	httpReq.Header.Add("Content-Encoding", "snappy")
	httpReq.Header.Set("Content-Type", "application/x-protobuf")
	httpReq.Header.Set("User-Agent", userAgent)
	httpReq.Header.Set("X-Prometheus-Remote-Write-Version", "0.1.0")
	httpReq = httpReq.WithContext(ctx)

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	httpResp, err := c.client.Do(httpReq.WithContext(ctx))
	if err != nil {
		return err
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode/100 != 2 {
		scanner := bufio.NewScanner(io.LimitReader(httpResp.Body, maxErrMsgLen))
		line := ""
		if scanner.Scan() {
			line = scanner.Text()
		}
		err = fmt.Errorf("server returned HTTP status %s: %s", httpResp.Status, line)
		log.Println(err)
	}

	return err
}
