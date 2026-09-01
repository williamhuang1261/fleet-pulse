package metrics

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func TestExporterServesPrometheusFormat(t *testing.T) {
	c := NewCollector()
	c.SampleWindow = 50 * time.Millisecond // keep the test fast

	reg := prometheus.NewRegistry()
	reg.MustRegister(NewExporter(c))

	handler := promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("GET %s: %v", srv.URL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	body := string(bodyBytes)

	wantMetrics := []string{
		"fleet_pulse_cpu_percent",
		"fleet_pulse_mem_used_percent",
		"fleet_pulse_disk_used_percent",
		"fleet_pulse_net_bytes_sent_total",
		"fleet_pulse_net_bytes_recv_total",
	}
	for _, m := range wantMetrics {
		if !strings.Contains(body, m) {
			t.Errorf("response missing metric %q; body:\n%s", m, body)
		}
	}
}
