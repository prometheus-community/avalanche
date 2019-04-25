package main

import (
	"fmt"
	"log"

	"github.com/open-fresh/avalanche/metrics"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	metricCount    = kingpin.Flag("metric-count", "Number of metrics to serve.").Default("500").Int()
	labelCount     = kingpin.Flag("label-count", "Number of labels per-metric.").Default("10").Int()
	seriesCount    = kingpin.Flag("series-count", "Number of series per-metric.").Default("10").Int()
	metricLength   = kingpin.Flag("metricname-length", "Modify length of metric names.").Default("5").Int()
	labelLength    = kingpin.Flag("labelname-length", "Modify length of label names.").Default("5").Int()
	valueInterval  = kingpin.Flag("value-interval", "Change series values every {interval} seconds.").Default("30").Int()
	labelInterval  = kingpin.Flag("series-interval", "Change series_id label values every {interval} seconds.").Default("60").Int()
	metricInterval = kingpin.Flag("metric-interval", "Change __name__ label values every {interval} seconds.").Default("120").Int()
	port           = kingpin.Flag("port", "Port to serve at").Default("9001").Int()
	remoteURL      = kingpin.Flag("remote-url", "URL to send samples via remote_write API.").URL()
	remoteTenant   = kingpin.Flag("remote-tenant", "Tenant ID to include in remote_write send").Default("1").String()
)

func main() {
	kingpin.Version("0.2")
	kingpin.CommandLine.Help = "avalanche - metrics test server"

	kingpin.Parse()
	stop := make(chan struct{})
	defer close(stop)
	err := metrics.RunMetrics(*metricCount, *labelCount, *seriesCount, *metricLength, *labelLength, *valueInterval, *labelInterval, *metricInterval, stop)
	if err != nil {
		log.Fatal(err)
	}
	if *remoteURL != nil {
		// First cut: just send the metrics once then exit
		err = metrics.SendRemoteWrite(**remoteURL, *remoteTenant)
		if err != nil {
			log.Fatal(err)
		}
		return
	}
	fmt.Printf("Serving ur metrics at localhost:%v/metrics\n", *port)
	err = metrics.ServeMetrics(*port)
	if err != nil {
		log.Fatal(err)
	}
}
