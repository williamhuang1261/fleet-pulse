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
	"github.com/williamhuang1261/fleet-pulse/internal/config"
	"github.com/williamhuang1261/fleet-pulse/internal/metrics"
	"github.com/williamhuang1261/fleet-pulse/internal/secrets"
)

func main() {
	once := flag.Bool("once", false, "collect a single snapshot, print it, and exit")
	interval := flag.Duration("interval", 5*time.Second, "how often to poll host vitals in stdout mode")
	listen := flag.String("listen", "", "if set, serve Prometheus metrics on this address (e.g. :9090) instead of printing to stdout")
	configPath := flag.String("config", "", "path to a YAML config file; explicit flags still override its values")
	secretsFile := flag.String("secrets-file", "", "path to an age-encrypted file holding the /metrics bearer token; unset leaves /metrics open")
	identityFile := flag.String("identity-file", "", "path to the age identity (private key) that decrypts --secrets-file; required if --secrets-file is set")
	flag.Parse()

	cfg := config.Default()
	if *configPath != "" {
		loaded, err := config.Load(*configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "config: %v\n", err)
			os.Exit(1)
		}
		cfg = loaded
	}

	// Explicit flags win over the config file. flag.Visit only calls back
	// for flags the user actually set, so an unset flag never clobbers a
	// value the config file provided.
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "interval":
			cfg.Interval = *interval
		case "listen":
			cfg.Listen = *listen
		}
	})

	collector := metrics.NewCollector()

	if cfg.Listen != "" {
		var bearerToken string
		if *secretsFile != "" {
			if *identityFile == "" {
				fmt.Fprintln(os.Stderr, "--secrets-file requires --identity-file")
				os.Exit(1)
			}
			identity, err := secrets.LoadIdentity(*identityFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "identity: %v\n", err)
				os.Exit(1)
			}
			token, err := secrets.DecryptBearerToken(*secretsFile, identity)
			if err != nil {
				fmt.Fprintf(os.Stderr, "secrets: %v\n", err)
				os.Exit(1)
			}
			bearerToken = token
		}
		runServer(collector, cfg.Listen, bearerToken)
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

	ticker := time.NewTicker(cfg.Interval)
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

// runServer blocks, serving Prometheus-format host metrics at /metrics. If
// bearerToken is non-empty, /metrics requires an "Authorization: Bearer
// <bearerToken>" header; empty leaves the endpoint open, a documented
// default for a trusted internal network.
func runServer(collector *metrics.Collector, addr, bearerToken string) {
	reg := prometheus.NewRegistry()
	reg.MustRegister(metrics.NewExporter(collector))

	var metricsHandler http.Handler = promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
	if bearerToken != "" {
		metricsHandler = secrets.RequireBearerToken(bearerToken, metricsHandler)
		log.Print("fleet-pulse: /metrics is gated by a bearer token")
	} else {
		log.Print("fleet-pulse: /metrics is open (no --secrets-file set)")
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", metricsHandler)

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
