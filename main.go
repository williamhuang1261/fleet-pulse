// Command fleet-pulse polls a host's own vitals and either prints them as
// JSON or serves them on a Prometheus-compatible /metrics endpoint.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/williamhuang1261/fleet-pulse/internal/metrics"
)

func main() {
	once := flag.Bool("once", false, "collect a single snapshot, print it, and exit")
	interval := flag.Duration("interval", 5*time.Second, "how often to poll host vitals in stdout mode")
	listen := flag.String("listen", "", "if set, serve Prometheus metrics on this address (e.g. :9090) instead of printing to stdout")
	flag.Parse()

	collector := metrics.NewCollector()

	if *listen != "" {
		runServer(collector, *listen)
		return
	}

	ctx := context.Background()

	if *once {
		snap, err := collector.Collect(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "collect: %v\n", err)
			os.Exit(1)
		}
		printSnapshot(snap)
		return
	}

	ticker := time.NewTicker(*interval)
	defer ticker.Stop()

	for {
		snap, err := collector.Collect(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "collect: %v\n", err)
			os.Exit(1)
		}
		printSnapshot(snap)
		<-ticker.C
	}
}

// runServer blocks, serving Prometheus-format host metrics at /metrics.
func runServer(collector *metrics.Collector, addr string) {
	reg := prometheus.NewRegistry()
	reg.MustRegister(metrics.NewExporter(collector))

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	log.Printf("fleet-pulse: serving /metrics on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		fmt.Fprintf(os.Stderr, "listen: %v\n", err)
		os.Exit(1)
	}
}

func printSnapshot(snap metrics.Snapshot) {
	enc := json.NewEncoder(os.Stdout)
	if err := enc.Encode(snap); err != nil {
		fmt.Fprintf(os.Stderr, "encode: %v\n", err)
	}
}
