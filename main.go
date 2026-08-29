// Command fleet-pulse polls a host's own vitals and prints them as JSON.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/williamhuang1261/fleet-pulse/internal/metrics"
)

func main() {
	once := flag.Bool("once", false, "collect a single snapshot, print it, and exit")
	interval := flag.Duration("interval", 5*time.Second, "how often to poll host vitals")
	flag.Parse()

	collector := metrics.NewCollector()
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

func printSnapshot(snap metrics.Snapshot) {
	enc := json.NewEncoder(os.Stdout)
	if err := enc.Encode(snap); err != nil {
		fmt.Fprintf(os.Stderr, "encode: %v\n", err)
	}
}
