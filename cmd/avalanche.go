package main

import (
	"fmt"
	"log"

	"github.com/open-fresh/avalanche/metrics"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	metricCount        = kingpin.Flag("metric-count", "Number of metrics to serve.").Default("500").Int()
	labelCount         = kingpin.Flag("label-count", "Number of labels per-metric.").Default("10").Int()
	seriesCount        = kingpin.Flag("series-count", "Number of series per-metric.").Default("10").Int()
	metricLength       = kingpin.Flag("metricname-length", "Modify length of metric names.").Default("5").Int()
	labelLength        = kingpin.Flag("labelname-length", "Modify length of label names.").Default("5").Int()
	valueInterval      = kingpin.Flag("value-interval", "Change series values every {interval} seconds.").Default("30").Int()
	labelInterval      = kingpin.Flag("series-interval", "Change series_id label values every {interval} seconds.").Default("60").Int()
	metricInterval     = kingpin.Flag("metric-interval", "Change __name__ label values every {interval} seconds.").Default("120").Int()
	port               = kingpin.Flag("port", "Port to serve at").Default("9001").Int()
	remoteURL          = kingpin.Flag("remote-url", "URL to send samples via remote_write API.").URL()
	remoteBatchSize    = kingpin.Flag("remote-send-batch", "how many samples to send with each remote_write API request.").Default("2000").Int()
	remoteSamplesCount = kingpin.Flag("remote-samples-count", "how many samples to send in total to the remote_write API.").Default("100").Int()
)

func main() {
	kingpin.Version("0.3")
	kingpin.CommandLine.Help = "avalanche - metrics test server"
	kingpin.Parse()

	stop := make(chan struct{})
	defer close(stop)
	updateNotify, err := metrics.RunMetrics(*metricCount, *labelCount, *seriesCount, *metricLength, *labelLength, *valueInterval, *labelInterval, *metricInterval, stop)
	if err != nil {
		log.Fatal(err)
	}

	if *remoteURL != nil {
		if (**remoteURL).Host == "" || (**remoteURL).Scheme == "" {
			log.Fatal("remote host and scheme can't be empty")
		}
		if *remoteBatchSize <= 0 {
			log.Fatal("remote send batch size should be more than zero")
		}
		// First cut: just send the metrics once then exit
		err := metrics.SendRemoteWrite(**remoteURL, *remoteBatchSize, *remoteSamplesCount, *valueInterval, *labelInterval, *metricInterval, updateNotify)
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
