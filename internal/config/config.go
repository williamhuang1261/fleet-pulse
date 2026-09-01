// Package config loads fleet-pulse's YAML configuration file.
package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds the settings a running fleet-pulse agent needs. Zero values
// are never used directly; Default() supplies the starting point so a config
// file only has to name the fields it wants to change.
type Config struct {
	// Interval is how often the agent polls host vitals in stdout mode.
	Interval time.Duration `yaml:"interval"`
	// Listen is the address /metrics is served on. Empty means stdout mode.
	Listen string `yaml:"listen"`
}

// Default returns the settings fleet-pulse runs with when nothing overrides
// them: a 5 second poll interval and stdout mode (no Prometheus server).
func Default() Config {
	return Config{
		Interval: 5 * time.Second,
		Listen:   "",
	}
}

// Load reads a YAML file at path into a Config that starts from Default(),
// so a file naming only "listen:" still gets the default interval.
func Load(path string) (Config, error) {
	cfg := Default()

	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}
