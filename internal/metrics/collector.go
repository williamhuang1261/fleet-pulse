// Package metrics collects a bare-metal host's own vitals: CPU, memory,
// disk and network usage.
package metrics

import (
	"context"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	gopsnet "github.com/shirou/gopsutil/v3/net"
)

// Snapshot is one point-in-time reading of the host's vitals.
type Snapshot struct {
	Timestamp         time.Time `json:"timestamp"`
	CPUPercent        float64   `json:"cpu_percent"`
	MemUsedPercent    float64   `json:"mem_used_percent"`
	DiskUsedPercent   float64   `json:"disk_used_percent"`
	NetBytesSentTotal uint64    `json:"net_bytes_sent_total"`
	NetBytesRecvTotal uint64    `json:"net_bytes_recv_total"`
}

// Collector reads host vitals for a single mount point.
type Collector struct {
	// DiskPath is the mount point disk usage is reported for.
	DiskPath string
	// SampleWindow is how long cpu.PercentWithContext blocks to measure
	// CPU usage over. A shorter window returns faster but is noisier.
	SampleWindow time.Duration
}

// NewCollector returns a Collector with sane defaults for a Linux host.
func NewCollector() *Collector {
	return &Collector{
		DiskPath:     "/",
		SampleWindow: 500 * time.Millisecond,
	}
}

// Collect takes one reading of the host's current vitals.
func (c *Collector) Collect(ctx context.Context) (Snapshot, error) {
	snap := Snapshot{Timestamp: time.Now()}

	cpuPercents, err := cpu.PercentWithContext(ctx, c.SampleWindow, false)
	if err != nil {
		return Snapshot{}, err
	}
	if len(cpuPercents) > 0 {
		snap.CPUPercent = cpuPercents[0]
	}

	vmem, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return Snapshot{}, err
	}
	snap.MemUsedPercent = vmem.UsedPercent

	diskUsage, err := disk.UsageWithContext(ctx, c.DiskPath)
	if err != nil {
		return Snapshot{}, err
	}
	snap.DiskUsedPercent = diskUsage.UsedPercent

	netCounters, err := gopsnet.IOCountersWithContext(ctx, false)
	if err != nil {
		return Snapshot{}, err
	}
	if len(netCounters) > 0 {
		snap.NetBytesSentTotal = netCounters[0].BytesSent
		snap.NetBytesRecvTotal = netCounters[0].BytesRecv
	}

	return snap, nil
}
