package metrics

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/prompb"

	dto "github.com/prometheus/client_model/go"
)

const maxErrMsgLen = 256

var userAgent = "avalanche"

type Client struct {
	url     *url.URL
	client  *http.Client
	timeout time.Duration
	batchSize,
	samplesCount,
	valueInterval,
	labelInterval,
	metricInterval int
	updateNotify chan struct{}
}

// SendRemoteWrite initializes a http client and
// sends metrics to a prometheus compatible remote endpoint.
func SendRemoteWrite(u url.URL, batchSize, samplesCount, valueInterval, labelInterval, metricInterval int, updateNotify chan struct{}) error {
	var rt http.RoundTripper = &http.Transport{}
	rt = &cortexTenantRoundTripper{tenant: "0", rt: rt}
	httpClient := &http.Client{Transport: rt}

	c := Client{
		url:            &u,
		client:         httpClient,
		timeout:        time.Minute,
		batchSize:      batchSize,
		samplesCount:   samplesCount,
		valueInterval:  valueInterval,
		labelInterval:  labelInterval,
		metricInterval: metricInterval,
		updateNotify:   updateNotify,
	}
	return c.write()
}

// Add the tenant ID header required by Cortex
func (rt *cortexTenantRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req = cloneRequest(req)
	req.Header.Set("X-Scope-OrgID", rt.tenant)
	return rt.rt.RoundTrip(req)
}

type cortexTenantRoundTripper struct {
	tenant string
	rt     http.RoundTripper
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

func (c *Client) write() error {

	tss, err := collectMetrics()
	if err != nil {
		return err
	}

	var (
		totalTime    time.Duration
		totalSamples int
	)

	fmt.Printf("Sending:  %v timeseries, %v samples, %v timeseries per request\n", len(tss), c.samplesCount, c.batchSize)
	for ii := 0; ii < c.samplesCount; ii++ {
		select {
		case <-c.updateNotify:
			fmt.Println("updating remote write metrics")
			tss, err = collectMetrics()
			if err != nil {
				return err
			}
		default:
		}

		start := time.Now()
		for i := 0; i < len(tss); i += c.batchSize {
			end := i + c.batchSize
			if end > len(tss) {
				end = len(tss)
			}
			req := &prompb.WriteRequest{
				Timeseries: tss[i:end],
			}
			err = c.Store(context.TODO(), req)
			if err != nil {
				return err
			}
			totalSamples += len(tss[i:end])
		}
		totalTime += time.Since(start)
	}
	fmt.Printf("Time: %v ; samples/sec: %v\n", totalTime.Round(time.Second), int(float64(totalSamples)/totalTime.Seconds()))
	return nil
}

func collectMetrics() ([]prompb.TimeSeries, error) {
	metricsMux.Lock()
	metricFamilies, err := promRegistry.Gather()
	if err != nil {
		return nil, err
	}
	metricsMux.Unlock()
	return ToTimeSeriesSlice(metricFamilies), nil
}

// ToTimeSeriesSlice converts a slice of metricFamilies containing samples into a slice of TimeSeries
func ToTimeSeriesSlice(metricFamilies []*dto.MetricFamily) []prompb.TimeSeries {
	tss := make([]prompb.TimeSeries, 0, len(metricFamilies)*10)
	timestamp := int64(model.Now()) // Not using metric.TimestampMs because it is (always?) nil. Is this right?

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
			case dto.MetricType_GAUGE:
				ts.Samples = []prompb.Sample{{
					Value:     *metric.Gauge.Value,
					Timestamp: timestamp,
				}}
			}
			tss = append(tss, ts)
		}
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
	sort.Slice(ret, func(i int, j int) bool {
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
	httpReq, err := http.NewRequest("POST", c.url.String(), bytes.NewReader(compressed))
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
	}
	if httpResp.StatusCode/100 == 5 {
		return err
	}
	return err
}
