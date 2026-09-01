package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	cases := []struct {
		name    string
		yaml    string
		wantCfg Config
	}{
		{
			name: "overrides both fields",
			yaml: "interval: 1s\nlisten: \":9091\"\n",
			wantCfg: Config{
				Interval: 1 * time.Second,
				Listen:   ":9091",
			},
		},
		{
			name: "only listen set keeps the default interval",
			yaml: "listen: \":9090\"\n",
			wantCfg: Config{
				Interval: 5 * time.Second,
				Listen:   ":9090",
			},
		},
		{
			name:    "empty file keeps every default",
			yaml:    "",
			wantCfg: Default(),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "config.yaml")
			if err := os.WriteFile(path, []byte(tc.yaml), 0o644); err != nil {
				t.Fatalf("writing test config: %v", err)
			}

			got, err := Load(path)
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if got != tc.wantCfg {
				t.Errorf("Load() = %+v, want %+v", got, tc.wantCfg)
			}
		})
	}
}

func TestLoadMissingFile(t *testing.T) {
	if _, err := Load("/nonexistent/config.yaml"); err == nil {
		t.Error("Load() with a missing file: want error, got nil")
	}
}
