package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func startPrometheus() {
	requestGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "uberdns_api_requests",
		},
		[]string{
			"type",
		},
	)

	go func() {
		for {
			ticker := time.NewTicker(500 * time.Millisecond)
			defer ticker.Stop()
			for range ticker.C {
				requestGauge.WithLabelValues("authorized").Set(float64(requestCounter.Count()))
				requestGauge.WithLabelValues("unauthorized").Set(float64(unauthorizedRequestCounter.Count()))
			}
		}
	}()

	prometheus.MustRegister(requestGauge)
	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", prometheusPort), nil))

}
