package config

import "os"

// Config holds pylon configuration.
type Config struct {
	CalURL string // base URL for the cal service API
}

// Load reads configuration from environment variables and flags.
func Load() *Config {
	calURL := os.Getenv("PYLON_CAL_URL")
	if calURL == "" {
		calURL = "http://localhost:8085"
	}
	return &Config{CalURL: calURL}
}
