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

func TestLoadDiscordConfig(t *testing.T) {
	t.Setenv("PYLON_DISCORD_WEBHOOK", "https://discord.com/api/webhooks/123/abc")
	t.Setenv("PYLON_DISCORD_BOT_TOKEN", "Bot.Token.Here")
	t.Setenv("PYLON_DISCORD_GUILD_ID", "guild-123")
	t.Setenv("PYLON_DISCORD_CHANNEL_ID", "chan-456")

	cfg := Load()

	if cfg.DiscordWebhook != "https://discord.com/api/webhooks/123/abc" {
		t.Errorf("expected webhook URL, got %q", cfg.DiscordWebhook)
	}
	if cfg.DiscordBotToken != "Bot.Token.Here" {
		t.Errorf("expected bot token, got %q", cfg.DiscordBotToken)
	}
	if cfg.DiscordGuildID != "guild-123" {
		t.Errorf("expected guild ID, got %q", cfg.DiscordGuildID)
	}
	if cfg.DiscordChannelID != "chan-456" {
		t.Errorf("expected channel ID, got %q", cfg.DiscordChannelID)
	}
}

func TestLoadDiscordConfigDefaults(t *testing.T) {
	// Clear all discord env vars
	t.Setenv("PYLON_DISCORD_WEBHOOK", "")
	t.Setenv("PYLON_DISCORD_BOT_TOKEN", "")
	t.Setenv("PYLON_DISCORD_GUILD_ID", "")
	t.Setenv("PYLON_DISCORD_CHANNEL_ID", "")

	cfg := Load()

	if cfg.DiscordWebhook != "" {
		t.Errorf("expected empty webhook, got %q", cfg.DiscordWebhook)
	}
	if cfg.DiscordBotToken != "" {
		t.Errorf("expected empty bot token, got %q", cfg.DiscordBotToken)
	}
	if cfg.DiscordGuildID != "" {
		t.Errorf("expected empty guild ID, got %q", cfg.DiscordGuildID)
	}
	if cfg.DiscordChannelID != "" {
		t.Errorf("expected empty channel ID, got %q", cfg.DiscordChannelID)
	}
}
