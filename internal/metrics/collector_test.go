package metrics

import (
	"context"
	"testing"
	"time"
)

func TestCollect(t *testing.T) {
	c := NewCollector()
	c.SampleWindow = 50 * time.Millisecond // keep the test fast

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	snap, err := c.Collect(ctx)
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	if snap.Timestamp.IsZero() {
		t.Error("Timestamp is zero")
	}
	if snap.CPUPercent < 0 || snap.CPUPercent > 100 {
		t.Errorf("CPUPercent = %v, want in [0, 100]", snap.CPUPercent)
	}
	if snap.MemUsedPercent <= 0 || snap.MemUsedPercent > 100 {
		t.Errorf("MemUsedPercent = %v, want in (0, 100]", snap.MemUsedPercent)
	}
	if snap.DiskUsedPercent <= 0 || snap.DiskUsedPercent > 100 {
		t.Errorf("DiskUsedPercent = %v, want in (0, 100]", snap.DiskUsedPercent)
	}
}

func TestCollectRespectsCanceledContext(t *testing.T) {
	c := NewCollector()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := c.Collect(ctx); err == nil {
		t.Error("Collect() with a canceled context: want error, got nil")
	}
}
