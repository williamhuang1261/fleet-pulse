package metrics

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
)

// Exporter adapts a Collector to the prometheus.Collector interface. Unlike a
// background-polling exporter, it takes one live Collect reading per scrape,
// so /metrics always reflects the host's current vitals rather than a
// possibly-stale cached value.
type Exporter struct {
	collector *Collector

	cpuDesc     *prometheus.Desc
	memDesc     *prometheus.Desc
	diskDesc    *prometheus.Desc
	netSentDesc *prometheus.Desc
	netRecvDesc *prometheus.Desc
}

// NewExporter returns an Exporter that reads vitals from c on every scrape.
func NewExporter(c *Collector) *Exporter {
	return &Exporter{
		collector: c,
		cpuDesc: prometheus.NewDesc(
			"fleet_pulse_cpu_percent",
			"Current CPU utilization, in percent.",
			nil, nil,
		),
		memDesc: prometheus.NewDesc(
			"fleet_pulse_mem_used_percent",
			"Current memory utilization, in percent.",
			nil, nil,
		),
		diskDesc: prometheus.NewDesc(
			"fleet_pulse_disk_used_percent",
			"Current disk utilization at the collector's mount point, in percent.",
			nil, nil,
		),
		netSentDesc: prometheus.NewDesc(
			"fleet_pulse_net_bytes_sent_total",
			"Cumulative bytes sent over all network interfaces since boot.",
			nil, nil,
		),
		netRecvDesc: prometheus.NewDesc(
			"fleet_pulse_net_bytes_recv_total",
			"Cumulative bytes received over all network interfaces since boot.",
			nil, nil,
		),
	}
}

// Describe implements prometheus.Collector.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- e.cpuDesc
	ch <- e.memDesc
	ch <- e.diskDesc
	ch <- e.netSentDesc
	ch <- e.netRecvDesc
}

// Collect implements prometheus.Collector. It takes one live reading from the
// underlying Collector and emits it as the current value of every metric.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	snap, err := e.collector.Collect(context.Background())
	if err != nil {
		ch <- prometheus.NewInvalidMetric(e.cpuDesc, err)
		return
	}

	ch <- prometheus.MustNewConstMetric(e.cpuDesc, prometheus.GaugeValue, snap.CPUPercent)
	ch <- prometheus.MustNewConstMetric(e.memDesc, prometheus.GaugeValue, snap.MemUsedPercent)
	ch <- prometheus.MustNewConstMetric(e.diskDesc, prometheus.GaugeValue, snap.DiskUsedPercent)
	ch <- prometheus.MustNewConstMetric(e.netSentDesc, prometheus.CounterValue, float64(snap.NetBytesSentTotal))
	ch <- prometheus.MustNewConstMetric(e.netRecvDesc, prometheus.CounterValue, float64(snap.NetBytesRecvTotal))
}
