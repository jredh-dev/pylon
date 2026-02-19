package config

import (
	"testing"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		envVal  string
		wantURL string
	}{
		{
			name:    "default when env not set",
			envVal:  "",
			wantURL: "http://localhost:8085",
		},
		{
			name:    "custom URL from env",
			envVal:  "https://cal.example.com",
			wantURL: "https://cal.example.com",
		},
		{
			name:    "custom URL with port",
			envVal:  "http://10.0.0.1:9090",
			wantURL: "http://10.0.0.1:9090",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envVal != "" {
				t.Setenv("PYLON_CAL_URL", tt.envVal)
			} else {
				t.Setenv("PYLON_CAL_URL", "")
			}

			cfg := Load()
			if cfg.CalURL != tt.wantURL {
				t.Errorf("expected CalURL %q, got %q", tt.wantURL, cfg.CalURL)
			}
		})
	}
}
