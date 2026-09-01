package metrics

import (
	"context"
	"testing"
	"time"
)

func TestCollect(t *testing.T) {
	cases := []struct {
		name    string
		ctx     func() (context.Context, context.CancelFunc)
		wantErr bool
	}{
		{
			name: "live host",
			ctx: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), 5*time.Second)
			},
			wantErr: false,
		},
		{
			name: "canceled context",
			ctx: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx, func() {}
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := NewCollector()
			c.SampleWindow = 50 * time.Millisecond // keep the test fast

			ctx, cancel := tc.ctx()
			defer cancel()

			snap, err := c.Collect(ctx)

			if tc.wantErr {
				if err == nil {
					t.Error("Collect() error = nil, want an error")
				}
				return
			}

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
		})
	}
}
