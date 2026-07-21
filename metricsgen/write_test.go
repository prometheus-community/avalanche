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
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/exp/api/remote"
	writev2 "github.com/prometheus/client_golang/exp/api/remote/genproto/v2"
	"github.com/prometheus/prometheus/prompb"
	"github.com/stretchr/testify/require"
)

func TestShuffleTimestamps(t *testing.T) {
	now := time.Now().UnixMilli()

	tss := []prompb.TimeSeries{
		{Samples: []prompb.Sample{{Timestamp: now}}},
		{Samples: []prompb.Sample{{Timestamp: now}}},
		{Samples: []prompb.Sample{{Timestamp: now}}},
	}

	shuffledTSS := shuffleTimestamps(tss)

	offsets := []int64{0, -60 * 1000, -5 * 60 * 1000}
	for _, ts := range shuffledTSS {
		timestampValid := false
		for _, offset := range offsets {
			expectedTimestamp := now + offset
			if ts.Samples[0].Timestamp == expectedTimestamp {
				timestampValid = true
				break
			}
		}
		if !timestampValid {
			t.Errorf("Timestamp %v is not in the expected offsets: %v", ts.Samples[0].Timestamp, offsets)
		}
	}

	outOfOrder := false
	for i := 1; i < len(shuffledTSS); i++ {
		if shuffledTSS[i].Samples[0].Timestamp < shuffledTSS[i-1].Samples[0].Timestamp {
			outOfOrder = true
			break
		}
	}

	if !outOfOrder {
		t.Error("Timestamps are not out of order")
	}
}

func TestNewRemoteAPIPath(t *testing.T) {
	for _, tc := range []struct {
		name      string
		urlPath   string
		wantPath  string
		wantQuery string
	}{
		{name: "no path appends default", urlPath: "", wantPath: "/api/v1/write"},
		{name: "root path appends default", urlPath: "/", wantPath: "/api/v1/write"},
		{name: "custom path is respected", urlPath: "/api/v1/receive", wantPath: "/api/v1/receive"},
		{name: "trailing slash is cleaned", urlPath: "/api/v1/receive/", wantPath: "/api/v1/receive"},
		{name: "prefix path is respected", urlPath: "/prometheus", wantPath: "/prometheus"},
		{name: "query string preserved", urlPath: "/api/v1/receive?tenant=a", wantPath: "/api/v1/receive", wantQuery: "tenant=a"},
		{name: "escaped path segment preserved", urlPath: "/tenant%2Freceive", wantPath: "/tenant%2Freceive"},
		{name: "double slash is cleaned", urlPath: "//foo", wantPath: "/foo"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var (
				mu       sync.Mutex
				gotPath  string
				gotQuery string
			)
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				gotPath = r.URL.EscapedPath()
				gotQuery = r.URL.RawQuery
				mu.Unlock()
				w.Header().Set("X-Prometheus-Remote-Write-Samples-Written", "0")
				w.Header().Set("X-Prometheus-Remote-Write-Histograms-Written", "0")
				w.Header().Set("X-Prometheus-Remote-Write-Exemplars-Written", "0")
				w.WriteHeader(http.StatusNoContent)
			}))
			t.Cleanup(srv.Close)

			u, err := url.Parse(srv.URL + tc.urlPath)
			require.NoError(t, err)

			api, err := newRemoteAPI(
				&ConfigWrite{URL: u},
				slog.New(slog.NewTextHandler(io.Discard, nil)),
				srv.Client(),
			)
			require.NoError(t, err)

			for _, msg := range []struct {
				typ remote.WriteMessageType
				req any
			}{
				{typ: remote.WriteV1MessageType, req: &prompb.WriteRequest{}},
				{typ: remote.WriteV2MessageType, req: &writev2.Request{Symbols: []string{""}}},
			} {
				mu.Lock()
				gotPath = ""
				gotQuery = ""
				mu.Unlock()

				_, err = api.Write(context.Background(), msg.typ, msg.req)
				require.NoError(t, err)

				mu.Lock()
				require.Equal(t, tc.wantPath, gotPath)
				require.Equal(t, tc.wantQuery, gotQuery)
				mu.Unlock()
			}
		})
	}
}
