package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	// Clear all env vars to ensure defaults.
	t.Setenv("PYLON_CAL_URL", "")
	t.Setenv("PYLON_DISCORD_WEBHOOK", "")
	t.Setenv("PYLON_DISCORD_BOT_TOKEN", "")
	t.Setenv("PYLON_DISCORD_GUILD_ID", "")
	t.Setenv("PYLON_DISCORD_CHANNEL_ID", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.CalURL != "http://localhost:8085" {
		t.Errorf("expected default CalURL, got %q", cfg.CalURL)
	}
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

func TestLoadEnvOverride(t *testing.T) {
	t.Setenv("PYLON_CAL_URL", "https://cal.example.com")
	t.Setenv("PYLON_DISCORD_WEBHOOK", "https://discord.com/api/webhooks/123/abc")
	t.Setenv("PYLON_DISCORD_BOT_TOKEN", "Bot.Token.Here")
	t.Setenv("PYLON_DISCORD_GUILD_ID", "guild-123")
	t.Setenv("PYLON_DISCORD_CHANNEL_ID", "chan-456")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.CalURL != "https://cal.example.com" {
		t.Errorf("expected CalURL from env, got %q", cfg.CalURL)
	}
	if cfg.DiscordWebhook != "https://discord.com/api/webhooks/123/abc" {
		t.Errorf("expected webhook from env, got %q", cfg.DiscordWebhook)
	}
	if cfg.DiscordBotToken != "Bot.Token.Here" {
		t.Errorf("expected bot token from env, got %q", cfg.DiscordBotToken)
	}
	if cfg.DiscordGuildID != "guild-123" {
		t.Errorf("expected guild ID from env, got %q", cfg.DiscordGuildID)
	}
	if cfg.DiscordChannelID != "chan-456" {
		t.Errorf("expected channel ID from env, got %q", cfg.DiscordChannelID)
	}
}

func TestParseFullConfig(t *testing.T) {
	input := `# pylon configuration

[cal]
url = https://cal.jredh.com

[discord]
webhook = https://discord.com/api/webhooks/999/xyz
bot_token = my-bot-token
guild_id = g-001
channel_id = c-002
`

	cfg := &Config{CalURL: "http://localhost:8085"}
	if err := cfg.parse(strings.NewReader(input)); err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if cfg.CalURL != "https://cal.jredh.com" {
		t.Errorf("CalURL = %q, want %q", cfg.CalURL, "https://cal.jredh.com")
	}
	if cfg.DiscordWebhook != "https://discord.com/api/webhooks/999/xyz" {
		t.Errorf("DiscordWebhook = %q", cfg.DiscordWebhook)
	}
	if cfg.DiscordBotToken != "my-bot-token" {
		t.Errorf("DiscordBotToken = %q", cfg.DiscordBotToken)
	}
	if cfg.DiscordGuildID != "g-001" {
		t.Errorf("DiscordGuildID = %q", cfg.DiscordGuildID)
	}
	if cfg.DiscordChannelID != "c-002" {
		t.Errorf("DiscordChannelID = %q", cfg.DiscordChannelID)
	}
}

func TestParseCommentsAndBlanks(t *testing.T) {
	input := `
# This is a comment
   # Indented comment

[cal]
# url = http://ignored.example.com
url = http://actual.example.com

`

	cfg := &Config{CalURL: "http://localhost:8085"}
	if err := cfg.parse(strings.NewReader(input)); err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if cfg.CalURL != "http://actual.example.com" {
		t.Errorf("CalURL = %q, want %q", cfg.CalURL, "http://actual.example.com")
	}
}

func TestParseUnknownSectionIgnored(t *testing.T) {
	input := `[unknown]
key = value

[cal]
url = http://custom.example.com
`

	cfg := &Config{CalURL: "http://localhost:8085"}
	if err := cfg.parse(strings.NewReader(input)); err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if cfg.CalURL != "http://custom.example.com" {
		t.Errorf("CalURL = %q", cfg.CalURL)
	}
}

func TestParseUnknownKeyIgnored(t *testing.T) {
	input := `[cal]
url = http://example.com
bogus_key = whatever
`

	cfg := &Config{CalURL: "http://localhost:8085"}
	if err := cfg.parse(strings.NewReader(input)); err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if cfg.CalURL != "http://example.com" {
		t.Errorf("CalURL = %q", cfg.CalURL)
	}
}

func TestParseValueWithEquals(t *testing.T) {
	input := `[cal]
url = http://example.com/path?a=1&b=2
`

	cfg := &Config{CalURL: "http://localhost:8085"}
	if err := cfg.parse(strings.NewReader(input)); err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if cfg.CalURL != "http://example.com/path?a=1&b=2" {
		t.Errorf("CalURL = %q, want URL with = in query string", cfg.CalURL)
	}
}

func TestParseWhitespaceHandling(t *testing.T) {
	input := `[cal]
  url  =  http://example.com  
`

	cfg := &Config{CalURL: "http://localhost:8085"}
	if err := cfg.parse(strings.NewReader(input)); err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if cfg.CalURL != "http://example.com" {
		t.Errorf("CalURL = %q, want trimmed value", cfg.CalURL)
	}
}

func TestParseEmptyFile(t *testing.T) {
	cfg := &Config{CalURL: "http://localhost:8085"}
	if err := cfg.parse(strings.NewReader("")); err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if cfg.CalURL != "http://localhost:8085" {
		t.Errorf("CalURL = %q, expected default preserved", cfg.CalURL)
	}
}

func TestEnvOverridesFile(t *testing.T) {
	input := `[cal]
url = http://from-file.example.com

[discord]
webhook = http://from-file-webhook
bot_token = file-token
guild_id = file-guild
channel_id = file-channel
`

	cfg := &Config{CalURL: "http://localhost:8085"}
	if err := cfg.parse(strings.NewReader(input)); err != nil {
		t.Fatalf("parse error: %v", err)
	}

	// Verify file values loaded.
	if cfg.CalURL != "http://from-file.example.com" {
		t.Fatalf("file CalURL not loaded: %q", cfg.CalURL)
	}
	if cfg.DiscordBotToken != "file-token" {
		t.Fatalf("file bot token not loaded: %q", cfg.DiscordBotToken)
	}

	// Now simulate env override.
	t.Setenv("PYLON_CAL_URL", "http://from-env.example.com")
	t.Setenv("PYLON_DISCORD_WEBHOOK", "")
	t.Setenv("PYLON_DISCORD_BOT_TOKEN", "env-token")
	t.Setenv("PYLON_DISCORD_GUILD_ID", "")
	t.Setenv("PYLON_DISCORD_CHANNEL_ID", "")

	cfg.applyEnv()

	// Env set -> overrides file.
	if cfg.CalURL != "http://from-env.example.com" {
		t.Errorf("CalURL = %q, want env override", cfg.CalURL)
	}
	if cfg.DiscordBotToken != "env-token" {
		t.Errorf("DiscordBotToken = %q, want env override", cfg.DiscordBotToken)
	}

	// Env empty -> file value preserved.
	if cfg.DiscordWebhook != "http://from-file-webhook" {
		t.Errorf("DiscordWebhook = %q, want file value preserved when env empty", cfg.DiscordWebhook)
	}
	if cfg.DiscordGuildID != "file-guild" {
		t.Errorf("DiscordGuildID = %q, want file value preserved", cfg.DiscordGuildID)
	}
	if cfg.DiscordChannelID != "file-channel" {
		t.Errorf("DiscordChannelID = %q, want file value preserved", cfg.DiscordChannelID)
	}
}

func TestLoadFromFile(t *testing.T) {
	// Create a temp pylonrc file.
	dir := t.TempDir()
	rcFile := filepath.Join(dir, ".pylonrc")
	content := `[cal]
url = https://cal.test.example.com

[discord]
webhook = https://discord.test/webhook
`
	if err := os.WriteFile(rcFile, []byte(content), 0644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	// Clear env vars.
	t.Setenv("PYLON_CAL_URL", "")
	t.Setenv("PYLON_DISCORD_WEBHOOK", "")
	t.Setenv("PYLON_DISCORD_BOT_TOKEN", "")
	t.Setenv("PYLON_DISCORD_GUILD_ID", "")
	t.Setenv("PYLON_DISCORD_CHANNEL_ID", "")

	// Load from the temp file directly via parse.
	cfg := &Config{CalURL: "http://localhost:8085"}
	f, err := os.Open(rcFile)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()

	if err := cfg.parse(f); err != nil {
		t.Fatalf("parse: %v", err)
	}

	if cfg.CalURL != "https://cal.test.example.com" {
		t.Errorf("CalURL = %q", cfg.CalURL)
	}
	if cfg.DiscordWebhook != "https://discord.test/webhook" {
		t.Errorf("DiscordWebhook = %q", cfg.DiscordWebhook)
	}
}

func TestParseMalformedLineIgnored(t *testing.T) {
	input := `[cal]
url = http://example.com
this line has no equals sign
another bad line
url_again = http://second.example.com
`

	cfg := &Config{CalURL: "http://localhost:8085"}
	if err := cfg.parse(strings.NewReader(input)); err != nil {
		t.Fatalf("parse error: %v", err)
	}

	// First url wins, then url_again is unknown key so ignored.
	if cfg.CalURL != "http://example.com" {
		t.Errorf("CalURL = %q", cfg.CalURL)
	}
}

func TestParsePartialSections(t *testing.T) {
	// Only cal section, no discord.
	input := `[cal]
url = http://only-cal.example.com
`

	cfg := &Config{CalURL: "http://localhost:8085"}
	if err := cfg.parse(strings.NewReader(input)); err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if cfg.CalURL != "http://only-cal.example.com" {
		t.Errorf("CalURL = %q", cfg.CalURL)
	}
	if cfg.DiscordWebhook != "" {
		t.Errorf("DiscordWebhook = %q, expected empty", cfg.DiscordWebhook)
	}
}
